# Protocolo Agentrix ↔ motor v1

Esta carpeta contiene los esquemas JSON Schema del protocolo
`agentrix-engine/1`. El worker Go inicia el motor de Starfighter como un
subproceso headless y ambos intercambian un objeto JSON por línea.

Los esquemas son el contrato de datos. Este README describe además el flujo
esperado y distingue sus garantías de las validaciones que todavía faltan.

## Transporte

| Canal | Dirección | Contenido |
| --- | --- | --- |
| `stdin` | Agentrix → motor | Mensajes JSON Lines del protocolo. |
| `stdout` | Motor → Agentrix | Solo mensajes JSON Lines del protocolo. |
| `stderr` | Motor → Agentrix | Logs y diagnósticos operativos. |

Cada mensaje ocupa una línea UTF-8 terminada en `\n`. El supervisor Go limita
la línea leída a 1 MiB; superar ese límite termina la ejecución con error.

## Sobre común

[`envelope.schema.json`](./envelope.schema.json) exige estos campos:

```json
{
  "protocolVersion": "agentrix-engine/1",
  "type": "tick_completed",
  "matchId": "m-12345678-abcd",
  "sequence": 42,
  "payload": {}
}
```

`sequence` comienza en `1` y debe aumentar por emisor. `matchId` puede estar
vacío durante el saludo inicial y el cierre general; después de inicializar la
partida identifica la ejecución activa.

## Flujo

1. El motor emite `engine_ready`.
2. Agentrix envía `initialize_match`; el motor responde `match_initialized`
   con `initialTick: 0`.
3. Para cada transición, Agentrix envía `advance_tick` con la acción de
   `tick N`; el motor devuelve `tick_completed` con el estado de `tick N+1`.
4. Agentrix envía `finish_match`; el motor responde `match_completed`.
5. Agentrix envía `shutdown`; el motor responde `shutdown_ack` y termina.

El motor autoritativo actual está implementado en Rust con Bevy y Avian2D. Go
orquesta los procesos y persiste resultados, pero no interpreta la carga útil
de una acción válida.

## Mensajes

De Agentrix al motor:

- [`initialize_match`](./initialize-match.schema.json): semilla, duración del
  tick, límite de ticks, participantes y configuración del juego.
- [`advance_tick`](./advance-tick.schema.json): acciones o fallos observados
  (`valid`, `timeout`, `invalid_output`, `crashed`, `disqualified`).
- [`finish_match`](./finish-match.schema.json): causa de cierre.
- [`shutdown`](./shutdown.schema.json): cierre ordenado del subproceso.

Del motor a Agentrix:

- [`engine_ready`](./engine-ready.schema.json): versión y capacidades.
- [`match_initialized`](./match-initialized.schema.json): estado y
  percepciones iniciales.
- [`tick_completed`](./tick-completed.schema.json): eventos, hash, snapshot
  público y percepciones del nuevo estado.
- [`match_completed`](./match-completed.schema.json): resultado consolidado.
- [`engine_error`](./engine-error.schema.json): error estructurado.
- [`shutdown_ack`](./shutdown-ack.schema.json): confirmación del cierre.

Los ejemplos válidos e inválidos están en [`examples/`](./examples/).

## Determinismo y frecuencia

La simulación opera a **60 Hz exactos** (`1.0 / tick_hz` de paso manual en Bevy)
utilizando el campo `tickHz: 60.0` en `initialize_match` (con fallback de
compatibilidad a `fixedTimestepMs`). Véase
[`docs/decisions/0005-protocol-state-machine-and-exact-timestep.md`](../../../docs/decisions/0005-protocol-state-machine-and-exact-timestep.md).

Con la misma versión del motor, arquitectura compatible, semilla,
configuración y secuencia de acciones, se obtiene idéntica secuencia de estados,
eventos, resultados y `stateHash`.

## Validación y ciclo de vida

El protocolo implementa validación bilateral estricta:
- Rust y Go validan `protocolVersion` en cada sobre.
- Rust y Go validan secuencia monotónica estricta por emisor (`ERR_INVALID_SEQUENCE`).
- Rust y Go validan consistencia de `matchId` a lo largo de toda la sesión (`ERR_MATCH_ID_MISMATCH`).
- Rust y Go aplican máquinas de estado formales para transiciones de ciclo de vida (`Ready` -> `Initialized` -> `Running` -> `Finished` -> `Shutdown`).
- Los errores son tipados mediante sobres `engine_error` con códigos estructurados (`ERR_INVALID_SEQUENCE`, `ERR_INCOMPATIBLE_VERSION`, `ERR_MATCH_ID_MISMATCH`, `ERR_INVALID_STATE`).
