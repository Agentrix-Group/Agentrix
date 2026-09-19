# ADR-0006: Commit transaccional de partidas, tabla match_runs y publicación atómica de replay

## Estado

accepted

## Fecha

2026-09-19

## Contexto

Antes de esta decisión, la finalización de una partida presentaba riesgos de consistencia, condición de carrera y publicación espuria de artefactos:

1. **Escrituras no transaccionales:** El supervisor Go ejecutaba mutaciones separadas para guardar resultados individuales (`CreateResult`), actualizar el estado de la partida (`UpdateMatch`) y marcar el trabajo como completado. Si el worker fallaba a mitad de camino, la base de datos quedaba en un estado inconsistente (resultados guardados pero partida en estado pendiente, o viceversa).
2. **Escrituras de workers zombi:** Un worker que perdiera su lease debido a congestión o pausa de GC podía seguir ejecutando y commitear resultados extemporáneos en la base de datos, sobrescribiendo el trabajo de un segundo worker que ya hubiera tomado la partida con un fencing token superior.
3. **Ausencia de auditoría de intentos (`match_runs`):** No existía una entidad en base de datos que registrara cada ejecución intentada, el worker responsable, el fencing token utilizado y el estado de la corrida (`running`, `committed`, `aborted`, `superseded`).
4. **Replays no atómicos ni sellados:** Los replays se escribían directamente en su ruta canónica (`replays/<id>.ndjson`) mientras la simulación avanzaba. Partidas fallidas o interrumpidas dejaban archivos de replay parciales e inválidos accesibles como si fuesen definitivos. Además, no se persistía el digest criptográfico SHA-256 del replay ni se garantizaba su inmutabilidad frente a abortos.

## Decisión

1. **Tabla y modelo `match_runs`:**
   - Se crea la tabla `match_runs` con:
     - `id VARCHAR(64) PRIMARY KEY` (`run_id` UUID generado por intento).
     - `match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE CASCADE`.
     - `worker_id VARCHAR(128) NOT NULL`.
     - `fencing_token BIGINT NOT NULL`.
     - `status VARCHAR(32) NOT NULL CHECK (status IN ('running', 'committed', 'aborted', 'superseded'))`.
     - `started_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`.
     - `finished_at TIMESTAMPTZ`.
     - `heartbeat_at TIMESTAMPTZ`.
   - Se añade una restricción única parcial:
     `CREATE UNIQUE INDEX idx_match_runs_committed ON match_runs(match_id) WHERE status = 'committed';`
     Esto garantiza que a nivel de base de datos **solo un intento de ejecución puede ser canónico**.

2. **Commit atómico con validación de fencing (`CommitMatchResult`):**
   - Se introduce el método `CommitMatchResult(ctx, commit)` en el repositorio y servicio.
   - En una única transacción ACID de PostgreSQL:
     a. Se realiza `SELECT fencing_token, status FROM match_jobs WHERE match_id = $1 FOR UPDATE`.
     b. Si el `fencing_token` de la base de datos no coincide con el del intento actual, se rechaza la transacción con `ErrFencingTokenMismatch` (worker zombi).
     c. Si el trabajo ya figura como `completed`, se rechaza con `ErrMatchAlreadyCommitted`.
     d. Se actualiza `match_jobs` a `completed`.
     e. Se actualiza `matches` a `finished` (o `failed`), asignando `replay_id` y `finished_at`.
     f. Se inserta el registro canónico en `match_runs` con estado `committed`.
     g. Se insertan atómicamente todos los registros en `results`.

3. **Publicación atómica de replay inmutable:**
   - El stream de replay se escribe inicialmente en un archivo temporal (`replays/tmp/<replay_id>.ndjson`).
   - Al concluir la simulación, se sella el replay, se calcula su digest SHA-256 y tamaño en bytes, y se comprime opcionalmente a `.zst`.
   - Únicamente tras el éxito confirmado de `CommitMatchResult` en la base de datos, los archivos temporales se mueven/renombran atómicamente a su ruta canónica (`replays/<replay_id>.ndjson`).
   - Si la simulación falla, el lease se pierde o la transacción es rechazada, los archivos temporales son purgados y nunca quedan publicados como canónicos.

## Consecuencias

- Se erradica por completo la posibilidad de que un worker zombi sobrescriba un resultado legítimo.
- Cada intento de ejecución queda auditado formalmente en `match_runs`.
- Todo replay publicado en el almacenamiento es completo, inmutable, sellado y respaldado por una transacción comprometida.

## Evidencia relacionada

- `script/database/03_match_runs_postgresql.sql`: esquema DDL y restricciones.
- `src/model/match_run.go` y `src/model/match.go`: DTOs y tipos de commit.
- `src/repository/matches.go`: implementación de `CommitMatchResult` con `FOR UPDATE`.
- `src/service/replays.go`: gestión de archivo temporal y publicación atómica.
- `src/executor/executor.go` y `src/executor/worker.go`: orquestación del commit cercado.
