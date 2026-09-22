# ADR-0002: Starfighter con Avian para el MVP

> [!WARNING]
> Decisión sustituida parcialmente. La separación Go/Rust, Starfighter como único juego oficial y el objetivo de 60 Hz permanecen vigentes; la permanencia de Avian2D y el duelo de dos jugadores fueron reemplazados por [ADR-0013](0013-starfighter-rapier-and-free-for-all.md).

## Estado

superseded (2026-09-22)

Aceptada originalmente el 2026-09-19.

## Fecha

2026-09-19

## Contexto

El MVP necesita cerrar primero seguridad, ejecución única y despliegue reproducible. El código actual ya implementa Starfighter sobre Bevy y Avian2D. Documentos anteriores fijaron Rapier antes de disponer de sim-core, benchmarks o pruebas multiplataforma, y mezclaron el alcance inmediato con Gym y multi-juego.

## Decisión

- Starfighter es el único juego oficial del MVP.
- Go conserva el plano de control y coordina bots y engine.
- Rust conserva autoridad sobre simulación, percepciones, eventos, puntaje y resultado.
- El MVP mantiene Bevy y Avian2D.
- El timestep objetivo es 60 Hz exactos; la combinación actual de 17 ms y 60 Hz es una divergencia pendiente de código.
- Antes de Gym se extraerá un `sim-core` consumido por el evaluador oficial.
- El segundo juego será discreto y sin física y validará la frontera multi-juego.
- Gym ejecutará el mismo core y soportará vectorización; no reimplementará reglas en Python.
- Rapier con determinismo mejorado permanece candidato. La decisión se tomará solo después de benchmarks y conformidad en x86_64 y ARM64.

## Consecuencias

- Rapier no aparece como dependencia ni como requisito del MVP.
- El executor no se generaliza por anticipado.
- Replays y experimentos futuros registrarán versión de juego, digest del engine, hash de configuración, seed y plataforma.
- El hash de estado sirve para detectar divergencia, no para declarar determinismo sin pruebas.

## Evidencia relacionada

- `agentrix_engine/Cargo.toml`: dependencia Avian2D.
- `src/executor/executor.go`: coordinación de bots y engine.
- `src/game/registry.go`: Starfighter como única entrada actual.
- `docs/architecture/training-and-determinism.md`: criterios para la decisión futura.

## Sustituye

Sustituye la selección de Rapier contenida en [ADR-0001](0001-external-rust-engine.md). Conserva de esa decisión la separación entre Go y el motor Rust externo.
