# Replay y resultados

## Estado actual

El worker abre un archivo NDJSON antes de iniciar el bucle y escribe:

1. metadata;
2. snapshot público del tick 0;
3. snapshots públicos secuenciales por cada estado resultante;
4. resultado final.

Cada snapshot contiene un `stateHash` producido por Rust. El escritor comprueba ticks consecutivos, coincidencia entre envoltura y snapshot y que el hash final coincida con el último frame.

Si el ejecutable `zstd` existe, se crea además `<replay>.ndjson.zst` conservando el raw. La consulta prefiere actualmente el raw y solo descomprime Zstandard si aquel falta. Por tanto, la compresión es una copia auxiliar, no el artefacto canónico ni una política de retención.

## Limitaciones actuales

- El hash encadenado detecta cambios en snapshots, pero no firma ni vuelve inmutable el archivo.
- La metadata no registra `game_version`, `engine_digest`, `config_hash` ni plataforma.
- El registro `Replay` no se inserta como entidad independiente; la partida conserva el ID.
- Replay, resultados y partida se escriben en operaciones separadas sin commit idempotente ni fencing.
- Una falla parcial puede dejar artefactos o filas inconsistentes.

## Objetivo aprobado

- Escritura progresiva para no perder toda la evidencia ante una caída.
- Metadata completa antes del primer snapshot.
- Snapshot público separado de percepciones privadas.
- Sello final por digest sobre el stream completo.
- Almacenamiento inmutable después del commit.
- Commit idempotente asociado a job, intento y fencing token.
- Una única ejecución válida origina el resultado vigente; intentos anteriores permanecen auditables.
- HTTP sirve únicamente replays completos, sellados y autorizados.

## Resultado

Rust decide ganador, puntuaciones, posiciones y causa de término. Go valida la coherencia estructural y persiste el resultado; no recalcula reglas Starfighter. Un resultado oficial futuro nace provisional y solo se confirma según la política competitiva, capacidad todavía no implementada.

## Renderer

La web consume el NDJSON y dibuja Canvas 2D. No ejecuta el engine ni integra las físicas. Controles como scrub, pausa y velocidad dependen de snapshots completos. No existe directo por WebSocket en el MVP actual.
