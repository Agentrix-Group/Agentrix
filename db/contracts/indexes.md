# Índices y rutas de consulta

No se declara particionamiento sin una carga que lo justifique. Los índices
siguen las rutas concretas siguientes y las fixtures ejecutan `EXPLAIN` con
sequential scan deshabilitado para comprobar que las formas son utilizables.

| Ruta | Predicado / orden | Índice |
| --- | --- | --- |
| claim de worker | disponibles por prioridad y fecha | `match_jobs_claim` parcial |
| recuperación | leases vencidos | `match_jobs_expired_lease` parcial |
| monitor de partidas | batch + state + fecha | `matches_batch_state` |
| scoring por tarea | contest + entry + task + submitted | `submissions_score_candidates` parcial |
| juicio vigente | submission + task oficial efectivo | `judgements_one_effective_official` parcial y único |
| candidato de score | contest/task/submission/effective | `judgements_scoring` |
| revisión vigente | contest + revision descendente | `score_revisions_latest_complete` parcial |
| ranking renderizado | revision + division + rank + entry | `score_rows_ranking` |
| snapshot histórico | publication + division + rank + entry | `scoreboard_publication_rows_order` |
| eventos | contest + sequence / no aplicados | `contest_events_stream`, `score_events_unapplied` |
| auditoría | entity type/id/time | `audit_events_entity` |

Las consultas deben medirse de nuevo con cardinalidades reales mediante
`EXPLAIN (ANALYZE, BUFFERS)` antes de añadir o retirar índices. La suite no hace
promesas de latencia ni fuerza índices redundantes sobre PK/UNIQUE.

