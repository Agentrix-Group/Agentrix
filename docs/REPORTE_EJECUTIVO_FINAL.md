# Reporte Ejecutivo y Certificación Arquitectural Final — Agentrix

## 1. Resumen Ejecutivo de la Intervención

La evolución arquitectural de **Agentrix** post-transición `Participant → User` ha sido completada en su totalidad a lo largo de las **10 fases** requeridas por el prompt maestro. La plataforma ha evolucionado desde un baseline con deuda técnica e inconsistencias referenciales hacia un sistema distribuido robusto, seguro, auditable y determinista para concursos universitarios de programación de agentes autónomos.

### Métricas de Calidad Comparativas

| Métrica / Dimensión | Estado Inicial (Baseline) | Estado Final Certificado | Impacto / Delta |
| :--- | :--- | :--- | :--- |
| **Integridad de Identidad** | `participant` monolítico como usuario y bot | Modelo canónico: `User` (identidad) $\to$ `ContestEntry` (inscripción) $\to$ `Agent` $\to$ `Submission` | Aislamiento completo de responsabilidades |
| **Suite de Pruebas Go** | Fallos conocidos en `src/executor` | **100% pasando** (todos los paquetes, `src/executor` y `test/integration`) | 0 fallos, 0 tests ignorados en backend |
| **Suite de Pruebas Web (Vitest)** | 16 suites / 105 tests | **19 suites / 116 tests pasando** (100%) | Cobertura de Admin, Contests y Reusable States |
| **Suite de Pruebas Rust Engine** | 19 tests | **19 tests pasando** (0 fallos) | Exactitud 60 Hz e invariantes de percepción |
| **Integridad de Migraciones (Goose)** | v1 inicial con datos estáticos | **v1 $\to$ v4 completamente reversibles e idempotentes** | Rollback y re-migración comprobados |
| **Determinismo de Rankings** | Dependiente del orden de inserción | **100% determinista** con desempate en cascada de 5 niveles | Idempotencia en `ranking_applied_runs` |
| **Seguridad y Sesiones** | JWT simple sin revocación | RBAC dinámico, rotación de refresh tokens, revocación por reuso y Bcrypt coste 12 | Prevención activa de robo de sesión |
| **Verificación de Humo (Smoke)** | Inexistente / manual | **Script automatizado de 13 puntos** (`cmd/smoke/main.go`) | Código de salida 0 garantizado |
| **Builds de Producción** | No empaquetaba frontend en compose | `Dockerfile.web` (Nginx) + `docker-compose.yml` multi-servicio | Stack completo reproducible |

---

## 2. Tabla de Correspondencia de Contratos (OpenAPI $\leftrightarrow$ Código)

Todos los endpoints expuestos en [open-api/openapi.yaml](file:///home/f4nk1/Projects/Agentrix/open-api/openapi.yaml) corresponden estrictamente a la implementación Go y a los modelos de dominio.

| Endpoint OpenAPI | Método | Handler Go / Mapeo | Rol / Permiso Requerido | Entidad de Dominio / Persistencia |
| :--- | :--- | :--- | :--- | :--- |
| `/health/live` | GET | `s.livenessProbe` | Público | Liveness probe del proceso |
| `/health/ready` | GET | `s.readinessProbe` | Público | Verificación de PostgreSQL y worker |
| `/api/v1/auth/login` | POST | `s.login` | Público (Rate-limited) | `model.User`, `auth.SessionManager` |
| `/api/v1/auth/register` | POST | `s.register` | Público (Rate-limited) | `model.User`, Bcrypt cost 12 |
| `/api/v1/auth/refresh` | POST | `s.refreshToken` | Público (Rate-limited) | Rotación de token en `user_sessions` |
| `/api/v1/me` | GET | `s.checkSession` | Autenticado | Perfil canónico y capacidades activas |
| `/api/v1/users` | GET | `s.listUsers` | `admin` / `users:read:any` | Lista de cuentas de usuario en plataforma |
| `/api/v1/users/{id}` | GET, PUT, PATCH | `s.getUser`, `s.updateUser`, `s.activateUser` | `admin` | Mantenimiento de cuentas |
| `/api/v1/me/agents` | GET | `s.getMyAgents` | `pilot`, `admin` | Agentes propios del usuario autenticado |
| `/api/v1/contests` | GET, POST | `s.listPublicContests`, `s.createContest` | Público (GET) / `admin` (POST) | `model.Contest`, `contests` |
| `/api/v1/contests/{id}` | GET, PUT, PATCH | `s.getPublicContest`, `s.updateContest` | Público (GET) / `admin` (PUT) | Estados del torneo (`scheduled`, `in_progress`) |
| `/api/v1/contests/{id}/agents` | GET, POST | `s.listContestAgents`, `s.enrollAgent` | Público (GET) / `pilot` (POST) | `model.ContestEntry`, `contest_entries` |
| `/api/v1/contests/{id}/entries` | GET, POST | `s.listContestEntries`, `s.enrollAgent` | Público (GET) / `pilot` (POST) | Alias unificado de inscripción de agentes |
| `/api/v1/contests/{id}/rankings` | GET | `s.listContestRankings` | Público | `model.Ranking`, vista determinista |
| `/api/v1/contests/{id}/rankings/recalculate`| POST | `s.recalculateRankings` | `admin`, `organizer` | Recálculo atómico en cascada |
| `/api/v1/contests/{id}/rankings/publish` | POST | `s.publishRankingSnapshot` | `admin`, `rankings:publish` | `contest_rankings_snapshots` versionado |
| `/api/v1/contests/{id}/rankings/snapshots` | GET | `s.listRankingSnapshots` | Público | Lista de snapshots oficiales históricos |
| `/api/v1/matches` | GET, POST | `s.listMatches`, `s.createMatch` | Público (GET) / `matches:create` | `model.Match`, slots ordenados |
| `/api/v1/matches/{id}` | GET | `s.getMatch` | Público | Detalle de partida, slots y estado |
| `/api/v1/matches/{id}/run` | POST | `s.runMatch` | `admin`, `matches:run` | Encolamiento en `match_jobs` con fencing |
| `/api/v1/submissions/upload` | POST | `s.uploadSubmissionBundle`| `pilot`, `submissions:upload` | Bundle de bot (.py / tar.gz) a disco |
| `/api/v1/replays/{id}` | GET | `s.getReplay` | Público | Metadatos y checksum de replay |
| `/api/v1/replays/{id}/stream` | GET | `s.streamReplay` | Público | Stream de frames NDJSON descomprimido |

---

## 3. Certificación de Determinismo y Reproducibilidad

1. **Exactitud del Timestep y Frecuencia de Simulación:**
   - La simulación del motor headless Starfighter (`agentrix_engine`) corre a **60 Hz exactos** ($\Delta t = 16.666\text{ ms}$).
   - Verificado mediante pruebas de integración en Rust (`test_custom_starfighter_config_and_exact_60hz`) y certificación e2e en Go (`src/executor/e2e_certification_test.go`).
2. **Determinismo Físico y Hash de Estado:**
   - Para un seed determinado (e.g. `seed = 42`) y un conjunto idéntico de acciones emitidas por los bots, el motor genera exactamente la misma trayectoria de frames y los mismos hashes de estado `state_hash` tick a tick.
   - Verificado en `test/integration/engine_reproducibility_test.go`: dos ejecuciones independientes del motor con idéntico seed y secuencias idénticas de bots producen `state_hash[500]` idéntico bit a bit.
3. **Invariantes de Percepción y Aislamiento:**
   - Percepciones privadas estrictamente desacopladas del snapshot público (`tests::perception_never_leaks_rival_internal_fields`).
   - Un bot nunca recibe vida interna, cooldowns ni intenciones de su rival; solo coordenadas y velocidades observables.
   - Detección de colisiones continua en Rapier 2D impidiendo que balas de alta velocidad atraviesen naves sin impactar (`proptests::bullet_within_real_speed_range_never_tunnels`).
4. **Almacenamiento y Compresión de Replays:**
   - Formato de replay canónico: NDJSON progresivo comprimido con Zstandard (`.zst`).
   - Sello inmutable en base de datos: tamaño en bytes, número total de ticks, frame digest y ganador oficial.

---

## 4. Matriz de Seguridad y Permisos (RBAC)

### Roles y Capacidades

| Rol | Propósito | Capacidades Asignadas en BD |
| :--- | :--- | :--- |
| **`admin`** | Superusuario y operador de la plataforma | Acceso universal (`*`), `admin:access`, gestión de torneos, usuarios y ejecuciones |
| **`organizer`** | Creación y administración de torneos | `contests:create`, `contests:manage`, `matches:create`, `matches:run`, `rankings:publish` |
| **`pilot`** / `player` | Piloto participante de la competencia | `agents:create`, `submissions:upload`, `contests:enroll`, `matches:view`, `replays:view` |
| **`spectator`** | Visitante o espectador público | Lectura de torneos, partidas finalizadas, leaderboards y repeticiones |

### Políticas de Protección Operativa
- **Detección de Reuso de Tokens:** Si un atacante intenta utilizar un token de refresco ya consumido, se revoca atómicamente toda la familia de sesiones (`user_sessions.is_revoked = TRUE`).
- **Rate Limiting:** Los endpoints de autenticación restringen ráfagas sospechosas por IP cliente (10 solicitudes por ventana de 1 minuto).
- **Ofuscación Estricta de Telemetría:** El tracer unificado filtra mediante expresiones regulares:
  - Tokens Bearer y hashes JWT (`[redacted-token]`).
  - Direcciones de correo electrónico privadas (`[redacted-email]`).
  - Rutas absolutas del sistema de archivos interno (`[redacted-path]`).

---

## 5. Guía de Operación y Despliegue

### Comandos de Operación Rápida

```bash
# 1. Levantar el stack completo (PostgreSQL 16, API Go, Worker Go, Web Nginx):
make demo-up
# (o alternativamente: docker compose up --build -d)

# 2. Inicializar base de datos y ejecutar bootstrap con simulación auténtica:
make demo-reset
# (Aplica migraciones 1..4 y ejecuta bots de ejemplo con motor real produciendo replay y snapshot)

# 3. Ejecutar verificación de humo integral (13 puntos):
make demo-smoke
# (Verifica liveness, login admin/pilot, RBAC 403, upload, simulación real de 515 ticks y ranking)

# 4. Detener el stack:
make demo-down
```

### Credenciales de Demostración Iniciales
- **Administrador:** Usuario `admin`, Contraseña `admin123` (Rol `admin`)
- **Piloto Alfa:** Usuario `pilot_alpha`, Contraseña `pilot123` (Rol `pilot`, agente `StarHunter`)
- **Piloto Beta:** Usuario `pilot_beta`, Contraseña `pilot123` (Rol `pilot`, agente `StarEvasive`)

---

## 6. Deuda Técnica, Brechas Remanentes y Mitigaciones

Siguiendo estrictamente la directiva de honestidad técnica de `AGENTS.md`:

| Área | Brecha Documentada | Estado Actual | Mitigación Recomendada para Fase Posterior |
| :--- | :--- | :--- | :--- |
| **Aislamiento de Bots** | Sandbox rootless con Podman/Bubblewrap en servidores compartidos | En entornos de desarrollo y contenedores sin privilegios se utiliza aislamiento por subproceso de Python con límite de recursos | Implementar backend Podman rootless con fail-closed estricto cuando el host cuente con soporte de namespaces |
| **Streaming en Vivo** | Replays visualizables vía NDJSON/Zstd post-partida | Implementado y probado; no existe WebSocket push de frames en vivo tick a tick | Incorporar gateway SSE o WebSocket en Go para streaming en directo de partidas en curso |
| **Soporte Multi-Juego** | Starfighter es el único juego soportado | Arquitectura con interfaces desacopladas (`game_id = 'starfighter'`), pero motor especializado | Preservar invariante hasta certificación completa del MVP según ADR-0002 |

---

## 7. Declaración Formal de Completitud

Certifico que:
- Las **10 fases** del prompt maestro se han implementado directamente en el código fuente de `Agentrix` y `agentrix_engine`.
- Se aplicaron y verificaron las migraciones Goose v1 a v4 con reversibilidad e idempotencia demostradas.
- Se eliminó el concepto de `participant` como entidad monolítica de usuario.
- Todos los comandos de verificación de `AGENTS.md` (Go build, Go test, Go vet, npm test, vite build, cargo test, git diff --check) ejecutan limpiamente con código de salida **0**.
- El stack se encuentra listo para demostración, auditoría y despliegue local o contenerizado.
