# Entrenamiento y determinismo

## Estado actual

- Starfighter ejecuta sus reglas dentro del crate Rust acoplado a Bevy y Avian2D.
- No existe `sim-core` separado.
- No existe API Gym, binding Python ni ejecución vectorizada.
- No hay pruebas de conformidad entre x86_64 y ARM64.
- `stateHash` detecta divergencias del snapshot público; no garantiza determinismo por sí solo.

## Objetivo aprobado

El evaluador oficial y Gym deben ejecutar exactamente el mismo `sim-core`. Python nunca reimplementará reglas ni físicas. El adaptador Gym podrá vectorizar múltiples instancias del core sin lanzar una aplicación gráfica.

Cada replay o experimento registrará:

- `game_version`;
- `engine_digest`;
- `config_hash`;
- seed;
- versión de protocolo;
- plataforma y arquitectura relevantes.

## Conformidad

Una afirmación de determinismo requiere repetir la misma configuración y secuencia de acciones y comparar eventos, hashes y resultado. Debe probarse al menos en x86_64 y ARM64. Un hash diferente detecta el problema; hashes iguales en una muestra no constituyen una garantía universal.

## Avian y Rapier

[ADR-0013](../decisions/0013-starfighter-rapier-and-free-for-all.md) reemplaza Avian2D por `rapier2d` 0.35.3 con `enhanced-determinism`. Hasta que el corte termine, la implementación en producción sigue siendo Avian2D.

La migración conserva las exigencias de evidencia:

- benchmark de throughput contra la línea base de Avian (`agentrix_engine/benches/baseline/avian2d-f0.md`);
- escenarios de colisión representativos, incluido el atravesamiento de balas;
- comparación reproducible en x86_64 y ARM64 antes de afirmar determinismo entre plataformas;
- pruebas de conformidad del resultado.

No se asume que una biblioteca garantiza determinismo sin evidencia del sistema completo.

## Orden

`certificación MVP -> Rapier + todos contra todos -> sim-core -> segundo juego -> Gym`
