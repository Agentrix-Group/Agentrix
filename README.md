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

`make build` ejecuta formateo con escritura, pruebas y compilación. Revise primero el worktree. Los targets de base de datos modifican PostgreSQL y no deben ejecutarse como comprobación rutinaria.

## Documentación

La fuente de navegación es [docs/index.md](docs/index.md). El alcance real del MVP está en [docs/product/mvp-scope.md](docs/product/mvp-scope.md) y la única hoja de ruta vigente en [docs/roadmap/current.md](docs/roadmap/current.md).

Los documentos bajo `docs/archive/` son históricos y no dirigen desarrollo nuevo.
