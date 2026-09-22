# Operaciones, aislamiento y errores

Todas las operaciones mutables deben ejecutarse en una transacción. Se
recomienda `READ COMMITTED` para envío y workers, y `SERIALIZABLE` para flujos
administrativos que combinen comprobaciones externas. Las constraints siguen
siendo la última defensa.

| Operación | Entrada / salida | Locks en orden | Idempotencia | Error estable |
| --- | --- | --- | --- | --- |
| `register_team_for_contest` | contest, team, division, actor, aprobación tardía → entry | contest `FOR SHARE`; membresía/división | contest/team devuelve la fila previa | `22023` ventana; `42501` membresía/aprobación |
| `transition_contest_state` | contest, nuevo estado, actor, audit ID, razón → contest | contest `FOR UPDATE` | mismo estado devuelve contest | `55000` arista inválida; `22023` demasiado temprano |
| `submit_program` | entry, task, actor, artefacto, toolchain, key → submission | entry, task, contest `FOR SHARE`; índice único | `(entry, task, key)` devuelve la fila previa | `42501` membresía; `22023` ventana/artefacto; `54000` límite |
| `change_submission_disposition` | evento, submission, estado, razón → submission | submission `FOR UPDATE`; luego evento de score | UUID de evento y key derivada | `22023` estado/razón |
| `seal_evaluation_batch` | batch, digest, fecha → batch | batch `FOR UPDATE`; roster en orden de ID | mismo digest devuelve batch | `23514` roster vacío, no listo o incompatible |
| `seal_match` | match, fecha → match | match, batch; release; seats | estado ya sellado devuelve match | `23514` propósito, duplicado o cantidad de jugadores |
| `enqueue_match` | job, match, prioridad, disponibilidad → job | match `FOR UPDATE` | `UNIQUE(match_id)` | `55000` match no sellado |
| `claim_match_job` | attempt, worker, reloj → attempt | índice de jobs; `FOR UPDATE SKIP LOCKED`; job; match | attempt UUID | `P0002` no hay trabajo |
| `heartbeat_match_attempt` | attempt, fencing, reloj → attempt | attempt y job | reemplaza heartbeat del mismo token | `40001` token viejo/lease vencido |
| `record_match_seat_result` | attempt, fencing, seat, resultado → row | attempt y job `FOR SHARE` | misma PK y mismos valores | `40001` token viejo; `23505` valores divergentes |
| `accept_match_attempt` | attempt, fencing, digests, salida → match | match, attempt, job; FK diferida al accepted attempt | mismo accepted attempt devuelve match | `40001` fencing; `23514` digest/resultados/replay; `23505` ya aceptado |
| `fail_match_attempt` | attempt, fencing, origen, código → job | attempt, job, match | terminal por attempt | `40001` fencing viejo |
| `verify_judgement` | judgement, jurado, fecha → judgement | judgement; rol de concurso | estado verified | `42501` sin jurado; `55000` estado |
| `make_judgement_effective` | judgement, fecha, event key → judgement | judgement, submission, batch | effective/event key | `23505` doble efectivo; `23514` evidencia |
| `apply_rejudge_batch` | batch, fecha → batch | rejudge batch; submissions por UUID; judgements por item | estado applied | `40001` juicio anterior cambió; todo rollback ante un item inválido |
| `rebuild_scoreboard*` | revision, contest, cutoff → revision | advisory lock por contest; revision building | UUID de revisión | `55000` revisión no editable |
| `publish_scoreboard` | publication, revision, audiencia, cutoff → publication | advisory lock contest/audiencia; revisión `FOR SHARE` | UUID de publicación | `23514` cutoff inconsistente |

## Evitar anomalías

- Lost update y doble aceptación: el job y la partida se bloquean; el fencing
  creciente invalida workers atrasados y la FK `(match_id, accepted_attempt_id)`
  obliga a que el intento sea de esa partida.
- Write skew de rejuicio: primero se bloquea el lote, luego todas las submissions
  ordenadas por UUID, después juicios. Dos lotes sobre el mismo envío no pueden
  promover candidatos distintos.
- Revisión parcial: las celdas/filas solo son mutables mientras la revisión está
  `building`; publicar exige `complete` y copia filas/celdas en la transacción.
- At-least-once: keys de envío, scheduling, eventos, jobs y publicaciones tienen
  `UNIQUE`; repetir una operación no acumula score dos veces.
- La FK circular única es `matches.accepted_attempt_id`. Es `DEFERRABLE INITIALLY
  DEFERRED` porque el intento necesita primero la partida. La aceptación actualiza
  intento, partida y job en una sola transacción.

Los errores `worker_infrastructure` permanecen en attempt/job y nunca producen
una derrota. `bot` identifica una falla técnica del programa; solo se vuelve
resultado competitivo si la política y el seat result aceptado lo expresan.
