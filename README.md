# Agentrix

Agentrix es una plataforma universitaria para concursos de agentes. El repositorio contiene un MVP experimental de extremo a extremo para **Starfighter**, el único juego admitido actualmente: una API y un worker en Go coordinan bots Python persistentes y un motor autoritativo headless en Rust; la web React reproduce snapshots públicos guardados como NDJSON.

El proyecto **no está certificado para producción**. La simulación y el visor existen, pero el aislamiento de bots, la recuperación de trabajos, el sellado de resultados y el despliegue todavía tienen brechas documentadas.

## Estado actual

| Componente | Estado | Evidencia principal |
| --- | --- | --- |
| API REST y worker Go | Parcial | Se construyen en un único binario y proceso; todavía no se despliegan por separado. |
| Motor Starfighter | Implementado | `agentrix_engine` usa Bevy y Avian2D y expone `agentrix-engine/1` por JSON Lines. |
| Bots Python | Parcial | El proceso persiste durante la partida y usa tick 0, pero el sandbox no es apto para producción. |
| PostgreSQL | Parcial | Guarda datos estructurados y trabajos; la reserva carece de heartbeat y fencing. |
| Replay | Parcial | NDJSON progresivo con snapshots públicos y hashes; faltan sello inmutable y metadatos completos de reproducción. |
| Web | Implementado para el corte público | React, tema claro y visor Canvas 2D posterior a la partida. No hay directo por WebSocket. |
| Multi-juego, Gym y sim-core | No implementado | Son etapas posteriores al cierre del MVP seguro. |

## Componentes

```text
web React ──HTTP──> API Go ──> PostgreSQL / artifacts
                         │
                         └── worker Go
                               ├── bots Python persistentes
                               ├── agentrix-engine/1
                               └── motor Rust Starfighter
```

Go conserva las acciones y percepciones específicas del juego como JSON opaco. Rust valida esas acciones, ejecuta las reglas, produce percepciones privadas y emite el snapshot público y el resultado autoritativos.

## Modelo relacional de base de datos

El esquema relacional en PostgreSQL modela el ciclo de vida completo de la plataforma: control de accesos (RBAC), torneos, bots y versiones, cola de orquestación autoritativa, resultados y clasificaciones deterministas:

```mermaid
erDiagram
    ROLES {
        string id PK
        string description
        boolean active
    }
    PERMISSIONS {
        string id PK
        string description
        boolean active
    }
    ROLE_PERMISSIONS {
        string role_id FK
        string permission_id FK
        boolean active
    }
    USERS {
        string id PK
        string username UK
        string email UK
        string password
        string role_id FK
        boolean active
    }
    USER_ROLES {
        string user_id FK
        string role_id FK
    }
    SESSIONS {
        string id PK
        string user_id FK
        string token_hash UK
        boolean is_revoked
    }
    GAMES {
        string id PK
        string name
        string manifest_path
        int min_players
        int max_players
    }
    CATEGORIES {
        string id PK
        string description
        boolean active
    }
    CONTESTS {
        string id PK
        string game_id FK
        string category_id FK
        string state
        string status
    }
    AGENTS {
        string id PK
        string owner_user_id FK
        string game_id FK
        string name
    }
    SUBMISSIONS {
        string id PK
        string agent_id FK
        int version
        string code_path
        string status
    }
    CONTEST_ENTRIES {
        string id PK
        string contest_id FK
        string agent_id FK
        string user_id FK
        string submission_id FK
        string status
    }
    MATCHES {
        string id PK
        string contest_id FK
        string game_id FK
        string committed_run_id FK
        string status
        bigint seed
    }
    MATCH_SLOTS {
        string id PK
        string match_id FK
        int slot_index
        string contest_entry_id FK
        string submission_id FK
    }
    MATCH_JOBS {
        string id PK
        string match_id FK
        string status
        bigint fencing_token
    }
    MATCH_RUNS {
        string id PK
        string match_id FK
        string worker_id
        string status
        int attempt
    }
    RESULTS {
        string id PK
        string match_id FK
        string match_run_id FK
        string submission_id FK
        int score
        int rank
    }
    REPLAYS {
        string id PK
        string match_id FK
        string match_run_id FK
        string file_path
        int duration_ticks
    }
    RANKINGS {
        string id PK
        string contest_id FK
        string agent_id FK
        string user_id FK
        int score
        int points
        int rank
    }
    CONTEST_RANKINGS_SNAPSHOTS {
        string id PK
        string contest_id FK
        int version
        string published_by FK
    }

    ROLES ||--o{ ROLE_PERMISSIONS : "asigna"
    PERMISSIONS ||--o{ ROLE_PERMISSIONS : "contiene"
    ROLES ||--o{ USERS : "rol_principal"
    USERS ||--o{ USER_ROLES : "tiene"
    ROLES ||--o{ USER_ROLES : "asigna"
    USERS ||--o{ SESSIONS : "inicia"

    GAMES ||--o{ CONTESTS : "define_juego"
    CATEGORIES ||--o{ CONTESTS : "clasifica"

    USERS ||--o{ AGENTS : "es_propietario"
    GAMES ||--o{ AGENTS : "valido_para"
    AGENTS ||--o{ SUBMISSIONS : "versiona"

    CONTESTS ||--o{ CONTEST_ENTRIES : "inscribe"
    AGENTS ||--o{ CONTEST_ENTRIES : "participa"
    USERS ||--o{ CONTEST_ENTRIES : "autoriza"
    SUBMISSIONS ||--o{ CONTEST_ENTRIES : "bloquea_version"

    CONTESTS ||--o{ MATCHES : "programa"
    GAMES ||--o{ MATCHES : "reglas_motor"

    MATCHES ||--o{ MATCH_SLOTS : "asigna_slots"
    SUBMISSIONS ||--o{ MATCH_SLOTS : "ejecuta_codigo"
    CONTEST_ENTRIES ||--o{ MATCH_SLOTS : "origen_inscripcion"

    MATCHES ||--o{ MATCH_JOBS : "encola_trabajo"
    MATCHES ||--o{ MATCH_RUNS : "ejecuta_intento"

    MATCHES ||--o{ RESULTS : "produce"
    MATCH_RUNS ||--o{ RESULTS : "certifica"
    SUBMISSIONS ||--o{ RESULTS : "puntua"

    MATCHES ||--o{ REPLAYS : "graba_evento"
    MATCH_RUNS ||--o{ REPLAYS : "asocia_traza"

    CONTESTS ||--o{ RANKINGS : "computa_posicion"
    AGENTS ||--o{ RANKINGS : "posiciona_bot"
    USERS ||--o{ RANKINGS : "atribuye_usuario"
    CONTESTS ||--o{ CONTEST_RANKINGS_SNAPSHOTS : "publica_version"
```

Los scripts modulares de definición se encuentran organizados en [`db/database/`](db/database/) y las semillas canónicas en [`db/data/`](db/data/).

## Requisitos de desarrollo

- Go según `go.mod` (actualmente 1.25).
- Rust estable según `agentrix_engine/rust-toolchain.toml`.
- Python 3 para los bots.
- Node.js y npm compatibles con `web/package-lock.json`.
- PostgreSQL para persistencia y cola autoritativas.
- Repositorio hermano `agentrix_engine` para construir el motor.

## Comandos comprobados

```bash
GOCACHE=/tmp/agentrix-go-cache go build -mod=readonly ./...
GOCACHE=/tmp/agentrix-go-cache go vet -mod=readonly ./...
cd web && npm test -- --run && npm run build
cd ../../agentrix_engine && cargo test --locked
```

`go test -mod=readonly ./...` es el comando correcto para la suite Go, pero al 2026-09-19 falla en `src/executor` cuando Bubblewrap es detectable y los procesos de prueba no consiguen iniciar dentro del entorno restringido. No debe presentarse como una validación verde hasta resolver y volver a ejecutar esos casos.

Para construir el motor y el binario Go:

```bash
make build-engine
make build
```

## Arranque rápido y verificación de demo (Fase 7 / Fase 9)

El stack completo de Agentrix incluye plano de control en Go, base de datos PostgreSQL con migraciones versionadas (Goose v4), motor de simulación Starfighter en Rust, bots de ejemplo en Python y frontend web en React + Vite.

1. **Levantar el stack completo:**
   ```bash
   make demo-up
   # o alternativamente: docker compose up --build -d
   ```
2. **Inicializar y poblar con datos y ejecuciones auténticas:**
   ```bash
   make demo-reset
   # ejecuta migraciones automáticas y cmd/bootstrap con simulación real del motor
   ```
3. **Ejecutar verificación de humo integral (13 pasos):**
   ```bash
   make demo-smoke
   # verifica: liveness, readiness, login admin/pilot, RBAC (403), submissions,
   # slots, simulación con motor y bots reales, timeout, resultados, replay zstd y ranking
   ```
4. **Detener el stack:**
   ```bash
   make demo-down
   ```

`make build` ejecuta formateo con escritura, pruebas y compilación. Revise primero el worktree. Los targets de base de datos modifican PostgreSQL y no deben ejecutarse como comprobación rutinaria.

## Documentación

La fuente de navegación es [docs/index.md](docs/index.md). El alcance real del MVP está en [docs/product/mvp-scope.md](docs/product/mvp-scope.md) y la única hoja de ruta vigente en [docs/roadmap/current.md](docs/roadmap/current.md).

Los documentos bajo `docs/archive/` son históricos y no dirigen desarrollo nuevo.
