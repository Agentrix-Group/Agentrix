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

Avian2D es la implementación actual y permanece durante el MVP. Rapier con opciones de determinismo mejorado es solo candidato.

La decisión se tomará después de extraer `sim-core`, con:

- benchmark de throughput y memoria;
- escenarios de colisión representativos;
- comparación reproducible en x86_64 y ARM64;
- pruebas de conformidad del resultado;
- costo de integración con Bevy y Gym vectorizado.

No se reserva Rapier exclusivamente para Gym ni se asume que una biblioteca garantiza determinismo sin evidencia del sistema completo.

## Orden

`certificación MVP -> sim-core -> segundo juego -> Gym -> decisión Rapier`
