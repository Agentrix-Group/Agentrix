# Catálogo PK / FK / UNIQUE

Las referencias listadas son las que gobiernan pertenencia y scoring; cada DDL
contiene además checks de estado, fecha, rango y exclusividad.

## Identidad y equipos

| Tabla | PK | FK principales | UNIQUE relevante |
| --- | --- | --- | --- |
| `users` | `id` | — | username/email normalizados |
| `user_credentials` | `id` | `user_id→users` | una credencial activa por usuario (parcial) |
| `sessions` | `id` | user, sesión anterior | digest; sesión anterior |
| `roles` / `permissions` | `id` | — | `key` |
| `role_permissions` | role, permission | roles, permissions | PK |
| `user_roles` | user, role, granted_at | users, roles, grantor | un rol activo (parcial) |
| `teams` | `id` | creador→users | slug |
| `team_memberships` | `id` | team, user, grantor | periodo; una membresía activa (parcial) |
| `audit_events` | `id` | actor→users | — |

## Artefactos, juegos y políticas

| Tabla | PK | FK principales | UNIQUE relevante |
| --- | --- | --- | --- |
| `artifacts` | `id` | — | SHA/size listo (parcial) |
| `artifact_outbox` | `id` | artifact | idempotency key |
| `games` | `id` | — | slug |
| `game_releases` | `id` | game; manifest/engine/rules artifacts | game/version; id/game |
| `evaluation_policy_versions` | `id` | — | kind/algorithm/digest |
| `scoring_policy_versions` | `id` | — | kind/algorithm/digest |
| `languages` | `id` | — | key |
| `toolchains` | `id` | language | language/version |
| `baseline_programs` | `id` | release, executable artifact | release/name/version; id/release |

## Concurso, tareas y envíos

| Tabla | PK | FK principales | UNIQUE relevante |
| --- | --- | --- | --- |
| `contests` | `id` | creator | slug |
| `contest_divisions` | `id` | contest | contest/key; contest/id |
| `contest_entries` | `id` | contest, team, `(contest,division)` | contest/team; contest/id |
| `contest_user_roles` | contest,user,role,granted_at | contest, users, roles | uno activo (parcial) |
| `contest_tasks` | `id` | contest, release, eval/scoring version | contest/code; contest/position; composite de definición |
| `task_toolchains` | task,toolchain | `(contest,task)`, toolchain | PK |
| `test_groups` | `id` | `(contest,task)` | task/key; task/id |
| `test_cases` | `id` | `(task,group)`, scenario | task/key; task/id |
| `task_baselines` | task,baseline,purpose | task, baseline | PK |
| `submissions` | `id` | `(contest,entry)`, `(contest,task)`, task/toolchain, source | entry/task/idempotency; composites para juicios |
| `submission_disposition_events` | `id` | submission, actor | — |
| `build_attempts` | `id` | submission, toolchain, artifacts | submission/number; submission/id |

## Tandas, partidas y ejecución

| Tabla | PK | FK principales | UNIQUE relevante |
| --- | --- | --- | --- |
| `evaluation_batches` | `id` | `(contest,task)`, release, policy versions | id/contest/task; id/release |
| `batch_roster` | `id` | `(batch,contest,task)`, composite submission, build/baseline, artifact | batch/id; entry/submission/baseline parciales |
| `matches` | `id` | `(batch,contest,task)`, batch/release, test case, accepted attempt diferida | batch/schedule key; composites match/batch y match/task |
| `match_seats` | `id` | `(match,batch)`, `(batch,roster item)` | match/index; match/id/batch |
| `match_jobs` | `id` | match | un job por match |
| `match_attempts` | `id` | match, job | match/number; match/id; job/fencing |
| `match_seat_results` | attempt,seat | `(match,attempt)`, `(match,seat)` | PK |
| `match_replays` | attempt | `(match,attempt)`, artifact | artifact |

## Juicio, score y publicación

| Tabla | PK | FK principales | UNIQUE relevante |
| --- | --- | --- | --- |
| `judgements` | `id` | composite submission, batch/task, build, policies, verifier | un oficial efectivo por submission/task (parcial) |
| `judgement_cases` | judgement,case | judgement, test case, artifacts | PK |
| `judgement_matches` | judgement,match,seat | composite judgement/submission/batch; match/batch; match/seat/batch | PK |
| `rejudge_batches` | `id` | contest, actors | contest/idempotency |
| `rejudge_items` | batch,submission | batch, submission, juicios, políticas | candidate judgement |
| `score_revisions` | `id` | contest | contest/number; una building (parcial) |
| `score_events` | identidad bigint | contest, entry/task/revision composites | contest/event key |
| `score_cells` | revision,entry,task | revision/contest, entry, task, submission, judgement | PK |
| `score_rows` | revision,entry | revision/contest, entry, division | PK |
| `scoreboard_publications` | `id` | contest/revision, publisher | contest/audience/number; contest/id |
| `scoreboard_publication_rows` | publication,entry | publication/contest, entry, division | PK |
| `scoreboard_publication_cells` | publication,entry,task | publication/contest, entry, task | PK |

## Comunicación

| Tabla | PK | FK principales | UNIQUE relevante |
| --- | --- | --- | --- |
| `clarification_threads` | `id` | contest/entry, optional task, author | — |
| `clarification_messages` | `id` | thread, author | — |
| `contest_announcements` | `id` | contest, author | — |
| `contest_awards` | `id` | contest/entry, publication | contest/key/entry |
| `contest_events` | sequence | contest | contest/event key |

No hay arrays o JSON de submission IDs. Las relaciones que gobiernan score son
filas y FKs compuestas: `(contest_id, entry_id)`, `(contest_id, task_id)`,
`(batch_id, roster_item_id)`, `(match_id, attempt_id)` y `(match_id, seat_id)`.

