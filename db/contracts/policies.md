# Contratos de políticas y unidades

Las versiones son inmutables y se identifican también por digest. El esquema
rechaza tipos desconocidos y parámetros incompatibles.

| Tipo | Parámetros validados | Resultado |
| --- | --- | --- |
| `icpc_pass_fail` | `wrong_submission_penalty_minutes >= 0` | puntos máximos al primer accepted; penalización en segundos desde inicio más intentos incorrectos previos |
| `best_score` | `higher_is_better=true` | mayor `judgements.task_points` |
| `aggregate_points` | `higher_is_better=true`, agregación `sum`/`weighted_sum` | suma de `judgement_cases.task_points`; el juicio conserva el total explicable |
| `league_points` | win ≥ draw ≥ loss | `wins*win_points + draws*draw_points + losses*loss_points` |

`engine_score` es una magnitud emitida por Rust. `task_points` es la conversión
competitiva de una tarea. `wins/draws/losses` alimentan liga.
`penalty_seconds` es tiempo/desempate, nunca puntos. `rank` es posición global.
Cada versión declara `disqualification_task_points`; las fixtures usan cero. La
descalificación de toda la inscripción es el estado explícito
`contest_entries.status='disqualified'`, que la excluye de las filas oficiales.

`evaluation_policy_versions.kind` admite `fixed_cases`,
`reference_opponents`, `round_robin` y `league_schedule`; cada uno exige una
marca de suite/roster/schedule o rondas y decisión explícita de auto-juego.

Una tanda declara `reference_kind`. `fixed_suite` y `normalized_reference`
permiten comparar solo bajo el algoritmo versionado correspondiente;
`sealed_roster` no afirma comparabilidad con otra tanda. La base nunca combina
scores contra rivales cambiantes sin que el juicio apunte a la versión de
scoring que realizó esa conversión.

El productor del plan calcula `roster_digest` y `specification_digest` sobre la
serialización canónica versionada del roster y de la partida. PostgreSQL compara
esos compromisos al aceptar el worker, pero no vuelve a implementar la
serialización del orquestador ni confunde un hash con prueba multiplataforma de
determinismo.

Los límites numéricos exactos, DQ, emparejamiento, visibilidad de código,
feedback privado, tokens y retención son configuración de concurso/política u
operación, no constantes universales. Las cifras de `seed/test_fixtures.sql`
son exclusivamente fixtures.
