# Base de datos canónica de Agentrix

Esta carpeta define una base PostgreSQL nueva y construible desde cero para el
modelo competitivo de Agentrix. Es una única fuente editable: los módulos de
`schema/`, `functions/` y `views/`. No existe un `init.sql` duplicado, historial
de migraciones ni compatibilidad retroactiva.

## Frontera con la aplicación actual

El código Go actual en `src/database/database.go`, `src/database/migrations/` y
`cmd/migrate` usa Goose y espera otro esquema en `public`. `db/setup.sh` instala
este diseño en el esquema `agentrix`; **no integra ni vuelve compatible la
aplicación actual**. `script/setup_postgres.sh` también sigue siendo una ruta
distinta. La integración posterior corresponde al usuario y no forma parte de
este cambio.

PostgreSQL almacena compromisos, planificación, resultados y trazabilidad. No
ejecuta bots, no interpreta acciones de Starfighter y no arbitra reglas Rust.

## Instalación limpia

Requiere PostgreSQL 15 o posterior y una base ya creada, vacía y elegida de
forma explícita. El instalador no crea bases, roles ni contraseñas, no carga
fixtures y no ejecuta `DROP DATABASE`.

```bash
export PGHOST=127.0.0.1
export PGPORT=5432
export PGUSER=agentrix_owner
export PGDATABASE=agentrix_dev_clean
db/setup.sh
```

La inspección previa rechaza una base que ya contenga el esquema `agentrix` o
relaciones de usuario. Todos los módulos se aplican mediante `psql -X -1 -v
ON_ERROR_STOP=1`; cualquier error revierte la instalación completa. Una segunda
ejecución falla antes de modificar objetos.

La reconstrucción destructiva existe solo para una base local dedicada cuyo
nombre empiece por `agentrix_dev_`. Enumera los objetos que eliminará, exige una
confirmación exacta y recrea los esquemas de usuario, nunca la base:

```bash
export AGENTRIX_CONFIRM_REBUILD="$PGDATABASE"
db/setup.sh --rebuild-local-dev
```

No use esta opción con datos que deban conservarse.

## Pruebas de aceptación

Las pruebas necesitan otra base vacía y desechable con nombre único. El runner
no crea ni borra la base:

```bash
export PGDATABASE=agentrix_test_20260922_001
export AGENTRIX_CONFIRM_TEST="$PGDATABASE"
db/tests/run.sh
```

El runner instala el esquema, demuestra que la segunda instalación se rechaza,
carga `seed/test_fixtures.sql`, ejecuta diez suites con rollback y dos carreras
reales en conexiones separadas. Las fixtures fijan políticas concretas solo
para pruebas; no son defaults de producto. El runner deja la base de prueba
intacta para inspección y nunca elimina una base existente.

## Orden canónico

1. `schema/00_foundation.sql`: dominios y validadores cerrados.
2. `schema/01_*` a `08_*`: identidad, dominio competitivo y operación.
3. `functions/00_*` a `04_*`: invariantes y casos de uso transaccionales.
4. `views/00_scoreboards.sql`: lectura interna, publicada y de cola.
5. `functions/05_access_contracts.sql`: autenticación y revocación a `PUBLIC`.

`setup.sh` enumera el orden de forma deliberada y falla si falta un módulo.

## Semántica cerrada

- La unidad clasificada es `contest_entry`; un individuo usa un equipo de una
  persona.
- Un `submission` es una versión inmutable de un programa para una entrada y
  una tarea. Código nuevo implica otro envío.
- Solo juicios efectivos de tandas `official` o `rejudge` alimentan el ranking.
- `best_score` y las políticas de puntos eligen el mayor valor válido; ICPC
  elige el primer aceptado y añade tiempo más penalización por intentos previos.
- Una partida sellada fija release, roster, artefactos, semilla, escenario y
  digest. Los envíos posteriores pertenecen a una tanda posterior.
- Scores de tandas con rivales distintos solo se combinan cuando una política
  versionada lo expresa; `reference_kind` deja visible esa decisión.
- El freeze filtra por `submissions.submitted_at`, no por hora de juicio. Jurado
  ve la revisión interna actual; una publicación es un snapshot inmutable.
- Los puestos empatados usan `rank` o `dense_rank`, configurado en el concurso.

Admisión tardía, límites de envío, DQ, visibilidad, feedback privado, empate,
auto-juego y replay obligatorio son columnas o versiones explícitas. Fórmulas
del juego, retención de blobs y duración de sesiones no tienen defaults ocultos:
deben configurarse por el producto o su operador.

## Retención y blobs

Las FKs usan `RESTRICT` para conservar autoría, juicios, rejuicios, resultados y
publicaciones. Archivar significa cambiar estado o mover el blob manteniendo su
`storage_key` y SHA-256; no borrar filas necesarias para reproducir. Un artefacto
`pending` no es publicable, `failed` conserva el motivo y `artifact_outbox`
permite reconciliación idempotente. La política temporal de retención queda a
cargo de operación y debe respetar esas referencias.

Consulte [operations.md](contracts/operations.md),
[relations.md](contracts/relations.md), [authorization.md](contracts/authorization.md)
y [policies.md](contracts/policies.md), además de [indexes.md](contracts/indexes.md).
El estado de ejecución de esta entrega está en [VERIFICATION.md](VERIFICATION.md).
