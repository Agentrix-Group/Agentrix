# 0010. Cadena de ejecución inmutable, slots ordenados y contratos congelados

- Estado: accepted
- Fecha: 2026-09-20
- Autor: Agentrix Architecture Group

## Contexto

Tras la transición canónica de identidad de `Participant` a `User`, se identificaron brechas críticas en la trazabilidad e integridad de la ejecución:
1. `ContestEntry` no bloqueaba una versión inmutable (`submission_id`), permitiendo que un agente compitiera con código distinto al registrado en el torneo.
2. `Match` no poseía entidad de slots ordenados (`MatchSlot`), delegando los participantes a un array `submission_ids` no tipado.
3. `Result` y `Replay` se asociaban únicamente a `match_id`, sin asociarse al intento concreto (`MatchRun`), haciendo imposible auditar fallos o reintentos.
4. `POST /matches` devolvía únicamente mensajes de texto ("Match <id> scheduled successfully") forzando al cliente web a extraer IDs mediante expresiones regulares.
5. La creación y ejecución de partidas no garantizaban idempotencia transaccional compare-and-set.

## Decisión

### 1. Cadena de Dominio Obligatoria

Se establece y protege formalmente la siguiente jerarquía de dominio:
```text
User (Identidad canónica)
  └── Agent (Bot perteneciente a un usuario)
        └── Submission (Versión inmutable de código/modelo)
              ├── ContestEntry (Inscripción que bloquea una submission_id)
              └── MatchSlot (Slot ordenado 0..N en una partida)
                    └── Match (Partida programada)
                          └── MatchRun (Intento de ejecución con fencing_token)
                                ├── MatchJob (Cola de despacho)
                                ├── Result (Resultados asociados a run_id y slot_id)
                                └── Replay (Grabación asociada a run_id)

Contest ── ScoringPolicy ── Proyección de Ranking (alimentada solo por runs committed)
```

### 2. Máquinas de Estados Canónicas

- **ContestState**: `draft` $\rightarrow$ `published` $\rightarrow$ `registration_open` $\rightarrow$ `registration_closed` $\rightarrow$ `in_progress` $\rightarrow$ `finished` $\rightarrow$ `archived` (excepcionales: `suspended`, `cancelled`).
- **SubmissionStatus**: `uploaded` $\rightarrow$ `validating` $\rightarrow$ `ready` | `rejected` | `failed`; `ready` $\rightarrow$ `superseded`.
- **ContestEntryStatus**: Se adopta el modelo de **Admisión Directa**: `enrolled` $\rightarrow$ `withdrawn` | `disqualified`.
- **MatchStatus**: `draft` $\rightarrow$ `scheduled` $\rightarrow$ `queued` $\rightarrow$ `running` $\rightarrow$ `finished` | `failed` | `cancelled`.
- **MatchRunStatus**: `created` $\rightarrow$ `running` $\rightarrow$ `committed` | `aborted` | `superseded`.
- **MatchJobStatus**: `pending` $\rightarrow$ `reserved` $\rightarrow$ `completed` | `failed`.
- **ReplayStatus**: `staging` $\rightarrow$ `published` | `discarded`.

### 3. Invariantes de Partidas, Slots y Ejecución

1. **Slots inmutables**: Cada partida congelará sus `MatchSlot` (index, `submission_id`, `contest_entry_id`, snapshots de auditoría) en el momento de creación (`scheduled`). Una vez en estado `queued`, los slots no pueden modificarse.
2. **Run comprometido único**: Exactamente un `MatchRun` por partida puede alcanzar el estado `committed`, resguardado en la base de datos por un índice único parcial.
3. **Reintentos seguros**: Cualquier reintento aborta o invalida (`superseded`) el intento anterior y genera un nuevo `MatchRun` con `fencing_token` monotónico.
4. **Resultados y Replay por Run**: `Result` y `Replay` pertenecen a la tupla `(match_id, match_run_id)`. Solo los resultados del run `committed` se proyectan hacia el `Ranking`.
5. **ExecutionSpec**: La ejecución utiliza una especificación serializada normalizada que incluye protocolo, juego, versión del motor (commit/digest fijo), semilla, límites de tiempo/ticks y slots ordenados.

### 4. Contratos de API Estructurados

- `POST /matches` responderá un objeto tipado `MatchResponse` (`id`, `contest_id`, `game_id`, `status`, `slots`, etc.) con código HTTP 201 Created.
- `POST /matches/{id}/run` será idempotente: si la partida ya está encolada o ejecutándose, responderá el recurso existente sin duplicar trabajos ni corromper estados.
- Se prohíbe el uso de regex sobre mensajes de texto humano para la extracción de identificadores o control de flujo.

## Consecuencias

- El esquema de base de datos incorporará `submission_id` en `contest_entries`, la tabla `match_slots`, y las columnas `match_run_id` y `slot_id` en `results` y `replays`.
- Los endpoints HTTP y los servicios Go garantizarán transaccionalidad atómica e idempotencia.
- El frontend consumirá contratos tipados y eliminará cualquier dependencia de expresiones regulares en mensajes informativos.
