# Informe de consolidación — 2026-09-21

Ejecución del prompt maestro contra la auditoría del 2026-09-21. Base: Agentrix `adc732f`, agentrix_engine `239f236`. Los cambios quedaron commiteados localmente en la rama `consolidation/canonical-baseline` de ambos repositorios (motor `523469b`, fijado en `engine.lock`). No se hizo push.

## 1. Completado y verificado

| Área | Evidencia (ejecutada en esta máquina) |
| --- | --- |
| Fase 0: reproducción de los P0 sobre el código heredado | Backend: 8 de 9 tests P0 en rojo. El noveno (replay dentro del commit) resultó un falso positivo de la auditoría: Postgres aborta la transacción. Frontend: 3 de 3 en rojo (activate sin status, logout sin backend, refresh en `localStorage`). |
| Esquema canónico único | `TestDatabase_FreshMigrationRollbackAndReapply`, `TestDatabase_SchemaCheckDetectsMissingConstraint`, `TestDatabase_ConstraintsRejectInvalidRows`, `TestDatabase_CapabilityCatalogMatchesCode` |
| RBAC y sesiones | `TestSecurity_*` (7 tests): acceso horizontal, revocación de rol inmediata, estado de usuario explícito, rotación atómica del refresh, reuso que revoca la familia, CORS, tokens. `policy_test.go` cubre las políticas de ruta. |
| ScheduleRun transaccional e idempotente, fencing, leases | `TestOrchestration_*` (8 tests): commit cercado, reintento que crea un run nuevo, recuperación por lease vencido, workers concurrentes sin duplicar trabajo, `ScheduleRun` concurrente, sin motor no hay run, cancelación, verificación de digests |
| Concursos, inscripción y admisión | `TestCompetition_*` (5 tests) |
| Rankings transaccionales y snapshots | `TestRankings_*` (3 tests) |
| Ciclo competitivo completo | `TestLifecycle_CompetitiveMatchEndToEnd` |
| Motor real y determinismo | `TestEngine_RealMatchAndDeterminismCertification`: 1573 ticks; 3 reejecuciones con hash idéntico. Además `TestEngine_DigestMismatchIsRejected` y `TestEngine_ProtocolExamples`. |
| Sandbox Bubblewrap | 9 tests adversariales en `src/executor/sandbox_test.go`: fork bomb, memoria, bucle, CPU, salida, tamaño de archivo, red/host/secretos, huérfanos, fallo de arranque |
| Motor Rust | `cargo fmt --check`, `clippy -D warnings` y 22 tests en verde (con los cambios locales) |
| Contrato OpenAPI | `openapi_contract_test.go`: router ↔ spec (48 operaciones, rutas y políticas) y DTO ↔ schemas |
| Frontend | ESLint con 0 avisos; vitest 32/32; paridad i18n de 11 catálogos; build OK; entry 78.8 KB gzip (presupuesto 110), total 102.8 KB (presupuesto 260); `npm audit --omit=dev`: 0 vulnerabilidades |
| Compose desde volúmenes vacíos (podman-compose 1.6.0) | Servicios `postgres → migrate → api/worker/web` sanos. `bootstrap` produjo una partida real, replay publicado y snapshot v1. **Smoke: 12/12.** |
| E2E Playwright contra el stack Compose | 8/8 (escritorio y móvil): sin overflow horizontal, 404, sesión restaurada por cookie sin tokens en `localStorage`, logout, replay verificado con movimiento reducido |

Totales: `make check` sin fallos. `make test-integration`: 42 tests, 0 fallidos, **0 omitidos** (PostgreSQL 16 real, bwrap y motor real).

## 2. Implementado pero no ejecutable en este entorno

| Elemento | Por qué no se ejecutó |
| --- | --- |
| Workflow de CI (`.github/workflows/ci.yml`) | Requiere push. No se ejecutó. |
| `docker compose` (Docker Engine) | Solo hay podman. Lo verificado es `podman-compose`. `--wait` y `additional_contexts` con Docker quedan sin comprobar. |
| Sysctl de AppArmor en Ubuntu 24.04 (CI) | Host Arch; no aplica aquí. |
| `PodmanRuntime` (sandbox alternativo) | Implementado y seleccionable, pero **sin tests**. Los adversariales cubren solo Bubblewrap. |

## 3. Pendiente

- Publicar el commit del motor `523469b` (rama `consolidation/canonical-baseline`). Ya está fijado en `engine.lock`, pero CI hace checkout desde GitHub, así que ese commit tiene que existir en el remoto antes de que los jobs `engine`/`integration`/`e2e` puedan pasar.
- Tests unitarios de páginas del frontend (Admin, Contest, Match, Agents). Hoy solo `RankingsPage` y los flujos e2e cubren páginas.
- El visor de replay no tiene atajos de teclado; los controles son botones accesibles.
- La API no vuelve a verificar el digest al servir un replay: lo verifica al publicarlo y expone el SHA-256.
- El trigger de `replays` no restringe transiciones entre `staging` y `publish_failed` (ver nota en §7).
- `go test ./...` recorre también un paquete Go dentro de `web/node_modules` (`flatted`). Es inocuo, pero conviene excluirlo.
- Binarios `worker` y `fake-engine` en la raíz, salidos de `go build`. Ya están en `.gitignore`; se pueden borrar.

## 4. Bloqueadores externos

- **Push de ambas ramas**: CI necesita que el commit del motor y la rama de Agentrix estén en GitHub.
- **Producción**: TLS y proxy real, secretos gestionados, backups de PostgreSQL y del almacén de artefactos, y política de retención. Fuera del alcance del MVP y no certificados.

## 5. Decisiones aplicadas

Ver [ADR 0013](../decisions/0013-canonical-baseline-consolidation.md). Resumen:

- esquema canónico único;
- cadena canónica sin rol "participant";
- capacidades con namespace y políticas por ruta que fallan cerrado;
- refresh en cookie HttpOnly con rotación y detección de reuso;
- `ExecutionSpec` sellado como única fuente de ejecución;
- lease + fencing en todo cambio de estado de un job;
- un reintento es siempre un run nuevo;
- `tick_rate` racional;
- `agentrix-replay/2` con outbox;
- rankings con procedencia (`ranking_applied_runs`);
- sandbox fail-closed.

Conflicto técnico resuelto con evidencia: Bubblewrap dentro de un contenedor falla al montar `/proc` (`Can't mount proc on /newroot/proc: Operation not permitted`). Hace falta `systempaths=unconfined`. Se probó que funciona en podman y es la misma opción que usa Docker. Sin esa opción, el worker **no arranca**.

## 6. Eliminado (respaldo en `/tmp/agentrix-legacy-backup`)

- **Migraciones** `00001_init_schema` … `00004_deterministic_rankings_and_scoring_policy`. En su lugar queda `00001_canonical.sql`.
- **Scripts de base** `script/database/00_init_postgresql.sql`, `script/data/00_seeds_postgresql.sql` y `script/setup_postgres.sh`.
- **Modelos heredados** en `src/model/*`: agent, category, contest, game, match, match_run, replay, result, scoring_policy, session, state_machine, submission y user. Se sustituyen por `entities.go`, `states.go`, `identity.go`, `scoring.go`, `execution_spec.go`, `ranking.go` y `snapshot.go`.
- **Paquetes heredados**:
  - `src/common/*`, `src/auth/auth.go` y `src/auth/session.go`;
  - `src/connection/queue.go` (cola en memoria);
  - en `src/executor/*`: `sandbox.go`, `fallback.go` y `bot_session.go`;
  - en `src/game/*`: `game.go`, `registry.go` y `validate.go`;
  - `src/replay/zstd.go`.
- **Monolito** `main.go` y `Dockerfile`, reemplazados por `cmd/*` y los tres Dockerfiles.
- **OpenAPI fragmentado** (14 archivos y `components/`). Queda un único `open-api/openapi.yaml`.
- **Frontend heredado**: 9 componentes, `api/types.js` y 13 archivos de test que probaban la API anterior.

## 7. Matriz de estados y transiciones

Fuente: `src/model/states.go`. Los triggers de `00001_canonical.sql` aplican las mismas transiciones.

| Entidad | Estado | Transiciones permitidas |
| --- | --- | --- |
| Contest | draft | registration_open, cancelled |
| | registration_open | registration_closed, cancelled |
| | registration_closed | running, cancelled |
| | running | finished, cancelled |
| | finished / cancelled | archived |
| | archived | — |
| ContestEntry | enrolled | withdrawn, disqualified |
| | withdrawn / disqualified | — |
| Submission | validating | ready, rejected |
| | ready | disabled |
| | rejected / disabled | — |
| Match | scheduled | queued, cancelled |
| | queued | running, failed, cancelled |
| | running | finished, failed |
| | failed | queued (reintento = run nuevo), cancelled |
| | finished / cancelled | — |
| MatchRun | created | reserved, aborted |
| | reserved | running, failed, timed_out, aborted |
| | running | committed, failed, timed_out, aborted |
| | committed / failed / timed_out / aborted | — (terminales e inmutables) |
| MatchJob | pending | reserved, cancelled |
| | reserved | completed, failed |
| | completed / failed / cancelled | — |
| Replay (publicación) | staging | published, publish_failed |
| | publish_failed | published (reintento del outbox con backoff) |
| | published | — (inmutable) |

Nota: en `replays`, el trigger solo aplica la inmutabilidad de `published` y de los campos de identidad (digest, tamaño, claves). El orden `staging → publish_failed → published` lo impone el código (`PublishPendingReplays`), no la base de datos.

## 8. Matriz de roles y capacidades

Fuente: seed de `00001_canonical.sql`. `TestDatabase_CapabilityCatalogMatchesCode` garantiza que coincide con `src/model/identity.go`.

| Capacidad | admin | organizer | player | referee | spectator |
|---|---|---|---|---|---|
| `users:read:own` | ✓ | ✓ | ✓ | ✓ |  |
| `users:read:any` | ✓ | ✓ |  |  |  |
| `users:update:own` | ✓ | ✓ | ✓ | ✓ |  |
| `users:update:any` | ✓ |  |  |  |  |
| `users:roles:manage` | ✓ |  |  |  |  |
| `agents:create` | ✓ |  | ✓ |  |  |
| `agents:read:own` | ✓ |  | ✓ |  |  |
| `agents:read:any` | ✓ | ✓ |  |  |  |
| `agents:update:own` | ✓ |  | ✓ |  |  |
| `agents:update:any` | ✓ |  |  |  |  |
| `submissions:create:own` | ✓ |  | ✓ |  |  |
| `submissions:read:own` | ✓ |  | ✓ |  |  |
| `submissions:read:any` | ✓ | ✓ |  |  |  |
| `contests:view` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `contests:manage` | ✓ | ✓ |  |  |  |
| `entries:create:own` | ✓ |  | ✓ |  |  |
| `entries:manage:any` | ✓ | ✓ |  |  |  |
| `matches:view` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `matches:create` | ✓ | ✓ |  |  |  |
| `matches:run` | ✓ | ✓ |  | ✓ |  |
| `matches:cancel` | ✓ | ✓ |  |  |  |
| `rankings:view` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `rankings:publish` | ✓ | ✓ |  |  |  |
| `replays:view` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `admin:access` | ✓ | ✓ |  |  |  |

## 9. Matriz de rutas y políticas

Fuente: `x-agentrix-policy` en `open-api/openapi.yaml`, verificado contra el router por `openapi_contract_test.go`. `a | b` indica que basta cualquiera de las dos capacidades. Las variantes `:own`/`:any` se resuelven además por propiedad en el servicio.

| Método | Ruta | Política |
|---|---|---|
| GET | `/health/live` | pública |
| GET | `/health/ready` | pública |
| POST | `/api/v1/auth/register` | pública |
| POST | `/api/v1/auth/login` | pública |
| POST | `/api/v1/auth/refresh` | pública |
| POST | `/api/v1/auth/logout` | pública |
| POST | `/api/v1/auth/logout-all` | `users:read:own` |
| GET | `/api/v1/me` | `users:read:own` |
| GET | `/api/v1/users` | `users:read:any` |
| POST | `/api/v1/users` | `users:roles:manage` |
| GET | `/api/v1/users/{id}` | `users:read:own` \| `users:read:any` |
| PATCH | `/api/v1/users/{id}` | `users:update:own` \| `users:update:any` |
| PUT | `/api/v1/users/{id}/roles` | `users:roles:manage` |
| PUT | `/api/v1/users/{id}/status` | `users:update:any` |
| GET | `/api/v1/games` | `contests:view` |
| GET | `/api/v1/games/{id}` | `contests:view` |
| GET | `/api/v1/agents` | `agents:read:own` \| `agents:read:any` |
| POST | `/api/v1/agents` | `agents:create` |
| GET | `/api/v1/agents/{id}` | `agents:read:own` \| `agents:read:any` |
| PATCH | `/api/v1/agents/{id}` | `agents:update:own` \| `agents:update:any` |
| PUT | `/api/v1/agents/{id}/status` | `agents:update:own` \| `agents:update:any` |
| GET | `/api/v1/agents/{id}/submissions` | `submissions:read:own` \| `submissions:read:any` |
| POST | `/api/v1/agents/{id}/submissions` | `submissions:create:own` |
| GET | `/api/v1/submissions/{id}` | `submissions:read:own` \| `submissions:read:any` |
| POST | `/api/v1/submissions/{id}/disable` | `agents:update:own` \| `agents:update:any` |
| GET | `/api/v1/contests` | `contests:view` |
| POST | `/api/v1/contests` | `contests:manage` |
| GET | `/api/v1/contests/{id}` | `contests:view` |
| PATCH | `/api/v1/contests/{id}` | `contests:manage` |
| POST | `/api/v1/contests/{id}/transitions` | `contests:manage` |
| GET | `/api/v1/contests/{id}/entries` | `contests:view` |
| POST | `/api/v1/contests/{id}/entries` | `entries:create:own` \| `entries:manage:any` |
| PUT | `/api/v1/contests/{id}/entries/{entryId}/submission` | `entries:create:own` \| `entries:manage:any` |
| POST | `/api/v1/contests/{id}/entries/{entryId}/withdraw` | `entries:create:own` \| `entries:manage:any` |
| POST | `/api/v1/contests/{id}/entries/{entryId}/disqualify` | `entries:manage:any` |
| GET | `/api/v1/contests/{id}/rankings` | `rankings:view` |
| POST | `/api/v1/contests/{id}/rankings/recalculate` | `rankings:publish` |
| GET | `/api/v1/contests/{id}/rankings/snapshots` | `rankings:view` |
| POST | `/api/v1/contests/{id}/rankings/snapshots` | `rankings:publish` |
| GET | `/api/v1/contests/{id}/rankings/snapshots/{version}` | `rankings:view` |
| GET | `/api/v1/matches` | `matches:view` |
| POST | `/api/v1/matches` | `matches:create` |
| GET | `/api/v1/matches/{id}` | `matches:view` |
| POST | `/api/v1/matches/{id}/runs` | `matches:run` |
| POST | `/api/v1/matches/{id}/cancel` | `matches:cancel` |
| GET | `/api/v1/replays/{id}` | `replays:view` |
| GET | `/api/v1/replays/{id}/stream` | `replays:view` |
| GET | `/api/v1/admin/readiness` | `admin:access` |

## 10. Matriz de compatibilidad

| Contrato | Versión | Productor | Consumidor | Verificación |
| --- | --- | --- | --- | --- |
| ExecutionSpec | `agentrix-execution-spec/1` | `service.ScheduleRun` | `executor.Execute` | Hash canónico al sellar y al parsear; parser estricto |
| Protocolo del motor | `agentrix-engine/1` (`tickRate` racional, `initialize_match` estricto) | worker Go | `starfighter-engine` 0.3.0 (commit `523469b`) | `TestEngine_ProtocolExamples`, `protocol_contract_test.go`, tests Rust |
| Juego | starfighter (manifest con `tick_rate` 60/1, sin `tick_hz`) | `games/starfighter/manifest.yaml` | `game.LoadRegistry` (`KnownFields`) y `StarfighterConfig` (`deny_unknown_fields`) | Loader estricto en ambos lados |
| Replay | `agentrix-replay/2` (gzip NDJSON) | `replay.Writer` | `replay.Decode` y `web/src/viewer/replayParser.js` | Tests Go y vitest; v1 y zstd se rechazan |
| Bot | protocolo de líneas JSON: `init` y luego una percepción por tick | `executor/session.go` | bots Python (`games/starfighter/examples`) | Admisión en sandbox, lifecycle, e2e |
| HTTP | OpenAPI 3 (`open-api/openapi.yaml`) | `src/server` | `web/src/service/apiService.js`, `cmd/smoke` | Test de contrato, smoke 12/12, e2e 8/8 |
| Esquema de base | `TargetSchemaVersion = 1` | `cmd/migrate` | API y worker (`CheckSchemaCompatible`) | Tests de base de datos |

## 11. Comandos ejecutados (resultado final)

```text
go build ./... && go vet ./...                                    OK
make check                                                         OK (0 FAIL)
make test-integration   (PG16 podman :55432, bwrap, motor real)    42 PASS, 0 FAIL, 0 SKIP
go test -race -count=3 -run ConcurrentWorkersDoNotDuplicate        OK (assertion no determinista corregida)
cargo fmt --check && cargo clippy -D warnings && cargo test        OK, 22 tests
npx eslint . --max-warnings=0                                      OK
npx vitest run                                                     32/32
npm run test:parity / build / size                                 OK / OK / 78.8 KB entry
npm audit --omit=dev --audit-level=high                            0 vulnerabilidades
podman-compose build                                               api 44.6 MB, worker 175 MB, web 50 MB
podman-compose up -d                                               todos sanos (tras systempaths=unconfined)
podman-compose --profile demo run --rm bootstrap                   OK
podman-compose --profile smoke run --rm smoke                      12/12 PASSED
E2E_BASE_URL=http://127.0.0.1:3000 npx playwright test             8/8
git diff --check                                                   OK
```

## 12. Riesgos que impiden declarar el MVP listo

1. El commit del motor fijado en `engine.lock` todavía no está en el remoto.
2. CI no se ha ejecutado nunca con estos cambios.
3. El perfil de seguridad del contenedor del worker está relajado (seccomp, apparmor y systempaths en `unconfined`). El aislamiento de bots depende enteramente de Bubblewrap y rlimits. Una revisión externa del sandbox sigue pendiente.
4. No hay migración desde bases de datos heredadas. Cualquier instancia existente debe recrearse.
