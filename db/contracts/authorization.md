# Autorización y privilegios

## Modelo

`roles` y `permissions` son la fuente única. `user_roles` aplica ámbito global y
`contest_user_roles` ámbito de concurso; `users` no contiene `role_id`.
Los roles esperados son organizador, jurado, competidor, espectador y servicio
técnico, pero sus filas se aprovisionan fuera de la instalación.

`submit_program` exige usuario activo, inscripción aceptada y membresía vigente
en el equipo a la hora oficial de recepción. La membresía revocada no cambia la
autoría histórica. `verify_judgement` exige rol activo `jury` u `organizer` en el
concurso. Las operaciones administrativas restantes deben comprobar en el caso
de uso futuro el permiso concreto antes de invocar la función:

| Capacidad | Comprobación futura obligatoria |
| --- | --- |
| editar concurso/tarea | permiso global o `contest.configure` en ese concurso |
| sellar tanda / programar | `contest.schedule` y task del mismo concurso |
| ignorar/DQ envío | `submission.moderate`, razón y audit event |
| preparar/aplicar rejuicio | `contest.rejudge`; revisor distinto si la política lo exige |
| publicar scoreboard | `contest.publish_scoreboard`; source revision completa |
| worker | identidad técnica con `match.execute`, nunca permisos de jurado |
| clarificaciones | miembro vigente del entry o jurado según dirección del mensaje |

SQL no puede inferir separación de funciones organizativa ni aprobar por sí solo
un blob externo. Esas decisiones se auditan con `audit_events` y se validan antes
de llamar la operación.

## Sesiones

Solo se almacena `token_digest` SHA-256. `authenticate_session` une sesión y
usuario, exige no revocada, no expirada y usuario `active`; una sesión de usuario
desactivado nunca parece válida. Rotación usa `family_id` y
`rotated_from_session_id`. Duración y retención no tienen default en el esquema.

## Roles PostgreSQL

La instalación no crea roles de servidor. Al final revoca schema, tablas,
secuencias y funciones a `PUBLIC`. El propietario conserva administración. Una
instalación real debe crear fuera de este repositorio roles como:

- `agentrix_runtime`: `USAGE` del schema y `EXECUTE` solo en operaciones;
- `agentrix_reader`: vistas públicas autorizadas, sin tablas privadas;
- `agentrix_worker`: claim/heartbeat/result/accept/fail;
- `agentrix_jury`: vistas completas y operaciones de jurado;
- `agentrix_owner`: DDL, sin uso por la aplicación.

No conceda escritura directa a `submissions`, roster, seats, attempts,
judgements, score o publicaciones. Código fuente, percepciones privadas, logs y
tests privados requieren grants separados; las vistas públicas no los exponen.

