# Agentrix

Agentrix es una plataforma universitaria para concursos de agentes. El repositorio contiene el MVP de extremo a extremo para **Starfighter**, el único juego admitido: una API y un worker en Go coordinan bots Python persistentes, aislados con Bubblewrap, y un motor autoritativo headless en Rust (`agentrix_engine`). La web React reproduce los replays publicados.

El proyecto **no está certificado para producción**. El estado verificado y las brechas abiertas están en el informe de consolidación y en `docs/`.

## Cadena canónica

```text
User → Agent → Submission → ContestEntry → Match → MatchSlot → MatchRun → Result/Replay → RankingSnapshot
```

- Una `Submission` es un artefacto inmutable, direccionado por SHA-256. El worker la admite en el sandbox (`validating → ready | rejected`).
- Una `ContestEntry` fija la submission exacta con la que compite un agente.
- Programar una partida crea en una sola transacción un `MatchRun`, su `ExecutionSpec` sellado por hash y un `match_job`, todo protegido por `Idempotency-Key`.
- El worker ejecuta **solo** el `ExecutionSpec`. El commit exige lease vigente y fencing token. Resultados y replay se confirman juntos, y el replay se publica por outbox.
- Los rankings se recalculan de forma transaccional a partir de las corridas confirmadas. Un snapshot es una fotografía publicada e inmutable.

## Componentes

```text
web (nginx + React) ──/api/v1──> API Go ──> PostgreSQL 16
                                   │           ▲
                              artifacts        │ leases, fencing, outbox
                                   │           │
                                   └── worker Go ──> bots Python (bubblewrap + rlimits)
                                                └──> starfighter-engine (Rust, agentrix-engine/1)
```

| Binario | Función |
| --- | --- |
| `cmd/api` | HTTP, sesiones, RBAC por capacidades. Nunca ejecuta bots. Subcomando `healthcheck`. |
| `cmd/worker` | Reserva, ejecución, heartbeat, commit y reconciliación: leases vencidos, admisión, publicación de replays, rankings y GC. |
| `cmd/migrate` | Migraciones Goose (`up`, `status`). La API y el worker **no** migran; verifican la versión del esquema y los objetos críticos. |
| `cmd/bootstrap` | Datos de demo reales: crea el admin mediante el caso de uso y hace todo lo demás por HTTP. Es idempotente. |
| `cmd/smoke` | Verificación HTTP de 12 pasos contra un stack vivo. |

## Requisitos

- Go según `go.mod`.
- Rust estable. Motor en `../agentrix_engine`, en el commit fijado por `engine.lock`.
- Python 3 y `bwrap` para el sandbox.
- Node 22 y npm (`web/package-lock.json`).
- PostgreSQL 16.
- Docker o Podman con Compose para el stack completo.

## Verificación

```bash
make check                      # gofmt (solo verificación), vet, tests unitarios con -race y contratos OpenAPI↔router
make test-integration           # PostgreSQL + sandbox + motor reales; falla si algún test se salta
make engine-test                # fmt, clippy y tests del motor
cd web && npm run lint && npm test && npm run test:parity && npm run build && npm run size
```

`make test-integration` necesita `DB_HOST`, `DB_PORT`, `DB_USER` y `DB_PASSWORD` de un PostgreSQL de pruebas desechable, más `AGENTRIX_ENGINE_BIN`. Los tests crean y eliminan bases propias.

## Demo con Compose

```bash
./script/checkout_engine.sh                                      # motor en el commit de engine.lock
docker compose up --build -d --wait postgres api worker web      # volúmenes vacíos → migrate → api/worker/web
docker compose --profile demo run --rm bootstrap                 # admin, jugador, bots, concurso, partida real, replay, ranking
docker compose --profile smoke run --rm smoke                    # 12 comprobaciones HTTP
cd web && E2E_BASE_URL=http://127.0.0.1:3000 npx playwright test # e2e escritorio y móvil
```

Con Podman, `podman-compose` acepta los mismos argumentos. La web queda en `http://127.0.0.1:3000` y la API en `:8080`. Las credenciales de demo están en `.env.demo`, que solo vale para `MODE=demo`.

El worker necesita `seccomp=unconfined`, `apparmor=unconfined` y `systempaths=unconfined` para que Bubblewrap pueda crear namespaces y montar un `/proc` propio. Sin esas opciones **no arranca** (falla cerrado) en lugar de ejecutar bots sin aislamiento. En hosts Ubuntu 24.04 además hay que poner `kernel.apparmor_restrict_unprivileged_userns=0`.

## Configuración

`MODE` puede ser `dev`, `test`, `demo` o `production`.

- Fuera de `dev`, `ACCESS_SECRET` es obligatorio, debe tener ≥ 32 bytes y no puede ser un placeholder.
- En producción, `COOKIE_SECURE=true` es obligatorio. CORS no acepta `*`.
- `AGENTRIX_SANDBOX` puede ser `bubblewrap`, `podman` o `direct`. `direct` solo se permite en `dev`.

## Documentación

La navegación empieza en [docs/index.md](docs/index.md). Arquitectura: [docs/architecture/](docs/architecture/). Decisiones: [docs/decisions/](docs/decisions/). El contrato HTTP es [open-api/openapi.yaml](open-api/openapi.yaml): un único archivo, que un test compara con las rutas y políticas reales. Los documentos bajo `docs/archive/` son históricos.
