# Replay y resultados

## Formato

`agentrix-replay/2` es NDJSON comprimido con gzip:

1. `metadata`: `match_id`, `run_id`, `spec_hash`, `tick_rate` racional, participantes (slot, agente, submission, digest).
2. Un `snapshot` público por tick desde 0, con `state_hash` producido por Rust.
3. `result`: `final_tick` y `final_state_hash`, que deben sellar el último snapshot.

El escritor (`src/replay/ndjson.go`) exige ticks consecutivos y el sello final. El visor (`web/src/viewer/replayParser.js`) vuelve a validar la secuencia, el sello y el formato, y rechaza formatos anteriores.

## Commit y publicación (outbox)

1. El worker escribe el replay en **staging**, con clave derivada del run.
2. `CommitRun` registra en la misma transacción que los resultados la fila `replays` con `publication = staging`, su SHA-256 y su tamaño.
3. `PublishPendingReplays`, desde el worker y el reconciliador, verifica el digest, mueve el archivo a su clave definitiva y marca `published`. Un fallo marca `publish_failed` y reintenta con backoff exponencial.
4. La API sirve un replay solo cuando está `published` y la partida es visible para quien lo pide. El digest se verificó al publicar; la API no lo vuelve a verificar al servirlo, pero expone el SHA-256 para que el cliente lo compare (el smoke lo hace). Mientras tanto, la partida muestra que el resultado es oficial y el replay está en publicación.

Un run perdido o fallido descarta su staging. El GC de huérfanos elimina artefactos sin fila después de un periodo de gracia.

## Resultado

Rust decide ganador, puntuaciones, posiciones y causa de término. Go valida la coherencia estructural (un resultado por slot, posiciones válidas) y persiste. No recalcula reglas de Starfighter. Resultados y replay son inmutables: los triggers de PostgreSQL rechazan cualquier modificación.

## Rankings

Los rankings se recalculan a partir de los runs confirmados de partidas competitivas del concurso, bajo un advisory lock por concurso:

- `ranking_applied_runs` registra exactamente qué runs se proyectaron;
- el digest de ese conjunto se expone como procedencia;
- el orden es determinista, con desempates explícitos, empates en esquema 1224 y descalificados al final.

Un snapshot publicado es inmutable y versionado.

## Renderer

La web descarga el replay, lo valida y lo dibuja en Canvas 2D. No ejecuta el motor. Controles: reproducir/pausar, paso, reinicio y velocidad. Con `prefers-reduced-motion`, el visor arranca pausado. No hay directo por WebSocket.
