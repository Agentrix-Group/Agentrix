# ADR-0005: Máquina de estados bilateral en el protocolo IPC y simulación exacta a 60 Hz

## Estado

accepted

## Fecha

2026-09-19

## Contexto

El protocolo `agentrix-engine/1` entre el supervisor Go y el motor Rust externo sobre `stdin/stdout` presentaba las siguientes debilidades y divergencias:

1. **Validación unilateral de secuencia y protocolo:** Go validaba `protocolVersion` y `sequence` en las respuestas de Rust, pero el motor Rust ignoraba `protocolVersion`, no verificaba la secuencia monotónica de los comandos de entrada de Go y no validaba la consistencia de `matchId` en cada transición.
2. **Ausencia de máquina de estados estricta en Rust:** el motor aceptaba comandos fuera de ciclo (por ejemplo, `initialize_match` podía invocarse sobre una partida ya activa, o `advance_tick` tras haber finalizado la partida).
3. **Contradicción de frecuencia de simulación:** la arquitectura fijó 60 Hz exactos, pero el contrato usaba `fixedTimestepMs: 17`, provocando que Bevy simulara a `1000.0 / 17.0 ≈ 58.82` Hz.
4. **Constantes del juego hardcodeadas:** parámetros esenciales de Starfighter (dimensiones de arena, daño de proyectil, costo de disparo, reducción de escudo, velocidad de regeneración y vida/energía máximas) estaban fijos como constantes en `src/lib.rs` de `agentrix_engine` en lugar de ser parametrizables vía la configuración autoritativa.

## Decisión

1. **Validación bilateral de secuencia y versión:**
   - Tanto Go como Rust validan estrictamente `protocolVersion == "agentrix-engine/1"` en cada sobre recibido.
   - Rust lleva un contador `expected_in_seq` inicializado en 1. Cualquier comando con secuencia fuera de orden es rechazado inmediatamente con `ERR_INVALID_SEQUENCE`.
   - Ambos extremos validan que `matchId` coincida con la sesión inicializada en todos los mensajes ligados a la partida.

2. **Máquina de estados estricta del motor:**
   - Se formaliza el ciclo de vida del subproceso: `Ready` -> `Initialized` -> `Running` -> `Finished` -> `Shutdown`.
   - Comandos fuera de estado emiten un error estructurado `ERR_INVALID_STATE` (con `fatal: true` o `false` según la gravedad).

3. **Simulación a 60 Hz exactos:**
   - Se añade `tickHz` al contrato `initialize_match`.
   - Cuando `tickHz` está presente o configurado a 60 Hz, el motor Rust configura el paso de Bevy con `Duration::from_secs_f64(1.0 / tick_hz)` exacto, eliminando el error fraccional de los 17 ms.
   - Se mantiene `fixedTimestepMs` para compatibilidad con la visualización y persistencia existentes.

4. **Configuración autoritativa tipada (`StarfighterConfig`):**
   - Se crea la estructura `StarfighterConfig` en Go y Rust con valores predeterminados canónicos.
   - Las rutinas de física y combate de Bevy obtienen sus parámetros desde `Res<Settings>` / `Res<StarfighterConfig>` en lugar de constantes del módulo.

## Consecuencias

- Desaparece la divergencia entre 17 ms y 60 Hz.
- Se previenen desincronizaciones de protocolo, reentradas no controladas y corrupción de estado en el motor.
- Los tests de integración Go↔Rust validan rutas negativas de versión, secuencia, identidad de partida y ciclo de vida.
- La tabla de validación de `docs/architecture/engine-protocol.md` queda completamente implementada.

## Evidencia relacionada

- `agentrix_engine/src/envelope_engine.rs`: validación de entrada, secuencia y máquina de estados.
- `agentrix_engine/src/lib.rs`: `StarfighterConfig` y sistemas desacoplados de constantes hardcodeadas.
- `src/engine/subprocess.go` y `src/engine/types.go`: máquina de estados del cliente Go, `TickHz` y `StarfighterConfig`.
- `protocol/engine/v1/initialize-match.schema.json`: especificación de `tickHz`.
