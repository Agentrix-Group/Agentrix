# Protocolo entre Go y el motor

## Frontera

`agentrix-engine/1` usa JSON Lines sobre `stdin/stdout` de un subproceso headless:

- Go escribe comandos por `stdin`.
- Rust escribe únicamente sobres del protocolo por `stdout`.
- Los diagnósticos del motor pertenecen a `stderr`.

Los schemas en [`protocol/engine/v1/`](../../protocol/engine/v1/README.md) son el contrato de wire. Los tipos Go y Rust deben conformarse a ellos; ninguno sustituye por sí solo la validación del contrato completo.

## Ciclo

```text
engine_ready
initialize_match -> match_initialized(state 0)
advance_tick(0)  -> tick_completed(state 1)
...
finish_match     -> match_completed
shutdown         -> shutdown_ack
```

Cada sobre incluye `protocolVersion`, `type`, `matchId`, `sequence` y `payload`. La secuencia es monotónica por emisor.

## Propiedad de datos

- Go conoce el sobre y los estados técnicos de cada bot.
- Go conserva `payload` de acción, percepciones y snapshot como JSON opaco.
- Rust conoce y valida el esquema de acción Starfighter.
- Rust produce percepciones privadas por slot y un snapshot público separado.
- El motor nunca inicia ni habla directamente con bots.

## Validación actual

| Regla | Estado |
| --- | --- |
| Go valida versión de respuestas Rust | Implementado |
| Go valida secuencia de respuestas Rust | Implementado |
| Go limita línea y stderr del motor | Implementado |
| Worker valida tick de percepciones y acciones | Implementado |
| Rust valida campos de `WireAction` | Implementado |
| Rust valida `protocolVersion` de comandos | No implementado |
| Rust valida secuencia de comandos | No implementado |
| Ambos extremos validan `matchId` contra la sesión | No implementado |
| Runtime valida cada payload contra los JSON Schema | No implementado |
| `engineDigest` se entrega y persiste | No implementado |

Deserializar un campo no demuestra que fue comprobado. La certificación del MVP requiere pruebas negativas bilaterales para versión, secuencia, partida, tipos, campos requeridos, tamaños y orden del ciclo de vida.

## Evolución

Los cambios incompatibles crean una versión nueva del protocolo. Una partida registra la versión y el digest exactos del motor; nunca consulta “la versión actual” para explicar historia.
