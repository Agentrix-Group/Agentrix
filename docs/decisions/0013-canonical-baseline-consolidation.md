# ADR 0013: Consolidación sobre una línea base canónica

## Estado
Aceptado (Accepted). Fecha: 2026-09-21. Origen: auditoría del 2026-09-21 y prompt maestro de implementación.

## Contexto
La auditoría encontró contratos duplicados y divergentes. Los principales eran:

- varios esquemas SQL y migraciones incrementales incompatibles;
- tokens de refresco en `localStorage`;
- un timestep de 17 ms presentado como 60 Hz;
- ejecuciones que releían manifests vivos;
- commits de resultados sin fencing;
- replays servidos antes de estar sellados;
- rutas del frontend y de la API fuera del contrato OpenAPI.

Los tests de fase 0 reprodujeron 8 de 9 fallos P0 del backend y 3 de 3 del frontend.

## Decisión
1. **Esquema único.**
   - `00001_canonical.sql` reemplaza todas las migraciones anteriores (`TargetSchemaVersion = 1`).
   - Las máquinas de estado e inmutabilidades se aplican con triggers en PostgreSQL y con tablas equivalentes en `src/model/states.go`.
   - La API y el worker verifican la versión y los objetos críticos, y no migran.
2. **Cadena canónica.** `User → Agent → Submission → ContestEntry → Match → MatchSlot → MatchRun → Result/Replay → RankingSnapshot`. El rol "participant" no existe.
3. **RBAC por capacidades con namespace.**
   - Cada ruta declara una `Policy` y `authorize` falla cerrado.
   - El principal se resuelve desde la base de datos en cada request.
4. **Sesiones.**
   - Access token JWT HS256 en memoria del navegador.
   - Refresh en cookie HttpOnly, con rotación, detección de reuso por familia y cabecera CSRF.
5. **Ejecución.**
   - `ScheduleRun` es transaccional e idempotente.
   - El worker ejecuta solo el `ExecutionSpec` sellado.
   - Todo cambio de estado de un job verifica lease y fencing token.
   - Un reintento siempre crea un run nuevo.
6. **Tiempo exacto.** `tick_rate` racional de extremo a extremo (spec, protocolo del motor, replay, visor).
7. **Replay.**
   - Formato `agentrix-replay/2` (gzip NDJSON).
   - El commit deja el replay en staging y un outbox lo publica tras verificar el digest.
8. **Rankings.**
   - Recálculo transaccional bajo advisory lock, con `ranking_applied_runs` como procedencia.
   - Los snapshots son inmutables.
9. **Sandbox fail-closed.**
   - Bubblewrap con `prlimit` dentro del namespace, o Podman.
   - `direct` solo en `dev`.
   - En Compose, el worker necesita `seccomp`, `apparmor` y `systempaths` en `unconfined`. El proceso del worker sigue sin privilegios (uid 65532).

## Consecuencias
- No hay migración desde bases anteriores. Una base existente con otro esquema hace que la API se niegue a arrancar (`ErrSchemaDrift` / `ErrPendingMigrations`), y la demo parte de volúmenes vacíos.
- Formatos anteriores de replay (v1, zstd) y el campo `tick_hz` se rechazan.
- El motor cambió su contrato (`initialize_match` estricto, `tickRate`). `engine.lock` fija el commit del motor `523469b`, que contiene esos cambios.
- Relajar el perfil de seguridad del contenedor del worker es el costo explícito de ejecutar Bubblewrap dentro de un contenedor. La frontera de aislamiento de los bots es el sandbox interno.
