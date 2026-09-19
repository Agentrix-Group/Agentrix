# Agentrix

Agentrix es una plataforma de concursos para bots. El MVP implementa un único recorrido completo: un participante carga un bot Python, el worker ejecuta un duelo de Starfighter en el motor Rust con Bevy y Avian2D, guarda snapshots públicos autoritativos en NDJSON y los muestra en un visor Canvas de React.

## Alcance del MVP

- Juego único: `starfighter`, dos participantes por partida.
- Agentes: archivos Python 3 dentro de un ZIP con `agentrix.json` y `bot.py`.
- Protocolo de bot por JSON Lines: `init`, `perception`, `action`, `end`.
- Motor autoritativo externo: `bin/starfighter-engine` mediante `agentrix-engine/1` sobre `stdin/stdout`.
- Replays: metadata, snapshots públicos secuenciales desde tick 0 y resultado final en NDJSON.
- Consumo de partidas: cola PostgreSQL con reserva `FOR UPDATE SKIP LOCKED`; cola en memoria como respaldo de desarrollo.
- Interfaz: React con tema claro pastel, iconos Lucide y visor Canvas 2D.

## Recorrido del código

```text
open-api/  -> src/server/ -> src/service/ -> src/repository/ -> src/model/
                                  |
src/connection/ <- src/executor/ <-+-> src/engine/ -> motor Rust
                         |
                    src/replay/ -> artifacts/replays/*.ndjson
```

El backend Go trata la percepción y la acción espacial como JSON opaco. El motor Rust valida las acciones, avanza la simulación y produce las percepciones privadas y el snapshot público.

## Requisitos

- Go 1.25+
- Python 3.10+
- PostgreSQL 14+
- Node.js 22.22+
- El motor hermano `agentrix_engine` compilado como `bin/starfighter-engine`

## Verificación

```bash
GOCACHE=/tmp/agentrix-go-cache go test ./...
cd web && npm test -- --run && npm run build
cd ../../agentrix_engine && cargo test
```

## Ejecución local

```bash
make build-engine
make build
make run
```

El servidor usa `PORT=8080`, `ARTIFACTS_DIR=./artifacts` y las variables `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` y `DB_NAME` para PostgreSQL. Si la conexión no está disponible en desarrollo, la cola de partidas usa su implementación en memoria.

## Paquete de un bot

El ZIP contiene exactamente estos archivos en su raíz:

```text
agentrix.json
bot.py
```

Ejemplo de `agentrix.json`:

```json
{
  "name": "My Starfighter Bot",
  "entrypoint": "bot.py",
  "protocol_version": "1.0"
}
```

La admisión limita el ZIP a 2 MiB, valida rutas y contenido, analiza la sintaxis Python y ejecuta `init` seguido de la percepción del tick 0. El endpoint es `POST /api/v1/submissions/upload` con formulario multipart `agent_id` y `bundle`.

## Contratos

- `contracts/agentrix-submission.schema.json`: manifiesto del ZIP.
- `contracts/game.schema.json`: manifiesto fijo de Starfighter.
- `contracts/replay.schema.json`: tipos de línea del replay NDJSON.
- `protocol/engine/v1/`: contrato versionado Go–Rust.
- `open-api/`: API HTTP.
