# AGENTRIX: MASTER ARCHITECTURE, PROTOCOL REALIGNMENT & MULTI-ENGINE ROADMAP
**Status:** Canonical RFC / Architecture Roadmap  
**Target Branch:** `bot-protocol-generalization` (commit `ef8d3b5`)  
**Workspace Model:** Monorepo (`agentrix/`)  
**Target Games:** Starfighter (MVP Authoritative), Grid/Card/Simulations (Extensible)

---

## 1. RESUMEN EJECUTIVO Y ANÁLISIS DE LA RAMA

La rama `bot-protocol-generalization` resuelve la persistencia de procesos y estandariza JSON Lines para la comunicación de bots. Sin embargo, no está lista para producción debido a tres bloqueadores de bajo nivel:

1. **Desincronización de Tick 0:** Desfase entre la percepción inicial del motor (tick 0) y la expectativa de recepción del Worker en Go (tick 1).
2. **Fallback Inválido:** El Worker carga por defecto scripts de Arena Básica (`bot_hunter.py`), emitiendo acciones discretas (`ATTACK`) incompatibles con la física vectorial de Starfighter (`thrust`, `turn`, `shoot`, `shield`).
3. **Contaminación de Buffer por Timeout:** Procesos lentos siguen vivos tras el timeout, desfasando la cola de lectura y generando fallos continuos de `tick mismatch`.

Además, `agentrix_engine` está fuertemente acoplado a Starfighter. Para habilitar extensibilidad y soporte de Gym/RL sin romper la plataforma, se define esta arquitectura en capas.

---

## 2. DECISIONES DE ARQUITECTURA (ADR)

* **ADR-01: Runtime Común vs. Motores de Juego Independientes**  
  No se usarán librerías compartidas dinámicas (`.so`/`.dylib`) de Rust por riesgos de estabilidad de ABI y dependencias. Se compilará un binario independiente por juego (`starfighter-engine`, `grid-engine`) que implemente el protocolo común `agentrix-engine/1`.

* **ADR-02: Política de Físicas (Avian2D vs. Rapier2D)**  
  * **MVP Inmediato:** Mantener **Bevy + Avian2D**. Avian está integrado nativamente al ECS de Bevy y la mecánica base de Starfighter ya funciona.  
  * **Extensión Gym / RL:** Implementar un arnés de benchmark (10,000 ticks, colisiones masivas) para evaluar `Rapier2D` con `enhanced-determinism` frente a Avian. Solo se migrará a Rapier si Avian diverge materialmente entre plataformas durante entrenamientos locales en Gymnasium.

* **ADR-03: Payloads Opacos en Go**  
  Go (API y Worker) **no debe conocer** las estructuras internas de los juegos (`thrust`, `shield`, `hp`). Trata percepciones, acciones y snapshots como JSON en crudo (`json.RawMessage`).

* **ADR-04: Modelo de Ticks Estricto**  
  $$\text{State}[0] \rightarrow \text{Perception}[0] \rightarrow \text{Action}[0] \rightarrow \text{State}[1] \rightarrow \text{Perception}[1]$$  
  El índice de tick reportado por bots, motor, worker y replay debe ser exactamente el mismo en cada instante.

* **ADR-05: Replay Autorizado en Streaming NDJSON**  
  El visor web no re-simula físicas. Rust emite un snapshot público sanitizado por tick. Go lo transmite y persiste en formato NDJSON, comprimido posteriormente con Zstandard.

* **ADR-06: Aislamiento y Política Rígida de Timeout**  
  Cualquier bot que no responda en la ventana temporal asignada es terminado inmediatamente (`SIGKILL`) y descalificado de la partida. Esto previene envenenamiento de pipes.

---

## 3. ESTRUCTURA DEL MONOREPO

```text
agentrix/
├── contracts/
│   ├── engine/                     # Esquemas agentrix-engine/1
│   │   ├── init.schema.json
│   │   ├── tick_step.schema.json
│   │   └── match_result.schema.json
│   ├── bot/                        # Esquemas agentrix-bot/1
│   │   ├── bot_init.schema.json
│   │   ├── bot_perception.schema.json
│   │   └── bot_action.schema.json
│   └── games/                      # Esquemas específicos de juegos
│       └── starfighter/
│           ├── action.schema.json
│           ├── perception.schema.json
│           └── public_snapshot.schema.json
├── engine/
│   ├── runtime/                    # Runtime Rust común
│   │   ├── Cargo.toml
│   │   └── src/
│   │       ├── protocol.rs         # Framing stdin/stdout
│   │       ├── lifecycle.rs        # Loop de ejecución
│   │       ├── clock.rs            # Timestep fijo (dt = 1/60s)
│   │       ├── hash.rs             # Hashing canónico SHA-256
│   │       └── traits.rs           # SDK Trait (AgentrixGame)
│   └── benchmarks/                 # Pruebas comparativas Avian vs Rapier
├── games/
│   └── starfighter/
│       ├── Cargo.toml
│       ├── manifest.yaml           # Metadatos, agentes y binario
│       ├── src/
│       │   ├── main.rs             # Binario: bin/starfighter-engine
│       │   ├── game.rs             # Implementación de AgentrixGame
│       │   ├── components.rs       # Entidades Bevy (Ship, Bullet)
│       │   ├── physics.rs          # Integración Avian2D
│       │   └── serialization.rs    # Snapshots y percepciones
│       └── examples/               # Bots de referencia
│           ├── bot_hunter.py
│           └── bot_evasive.py
├── backend/
│   ├── cmd/
│   │   ├── api/                    # Servidor REST
│   │   └── worker/                 # Coordinador de partidas
│   ├── internal/
│   │   ├── engine/                 # Driver del proceso de motor Rust
│   │   ├── bot/                    # Driver de bots (JSON Lines)
│   │   ├── queue/                  # Cola Postgres FOR UPDATE SKIP LOCKED
│   │   └── replay/                 # Escritor streaming NDJSON
│   └── go.mod
├── web/
│   ├── src/
│   │   ├── components/replay/      # Controles, línea temporal y lienzo
│   │   └── renderers/
│   │       └── starfighter/        # Renderizado Canvas 2D / PixiJS
│   └── package.json
├── Makefile
└── README.md
```

---

## 4. CONTRATOS Y PROTOCOLOS UNIFICADOS

### 4.1 Trait del SDK Rust (`engine/runtime/src/traits.rs`)

```rust
use serde::{Deserialize, Serialize};
use serde_json::Value as JsonValue;
use std::collections::BTreeMap;

pub type PlayerId = String;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GameMetadata {
    pub id: String,
    pub version: String,
    pub min_players: usize,
    pub max_players: usize,
    pub tick_rate: u32,
    pub max_ticks: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MatchConfig {
    pub match_id: String,
    pub seed: u64,
    pub player_ids: Vec<PlayerId>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TickOutput {
    pub perceptions: BTreeMap<PlayerId, JsonValue>,
    pub public_snapshot: JsonValue,
    pub state_hash: [u8; 32],
    pub is_finished: bool,
    pub match_result: Option<MatchResult>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MatchResult {
    pub winner_id: Option<PlayerId>,
    pub reason: String,
    pub scores: BTreeMap<PlayerId, f64>,
}

pub trait AgentrixGame: Send + Sync {
    fn metadata(&self) -> GameMetadata;
    fn initialize(&mut self, config: MatchConfig) -> Result<(), String>;
    fn initial_perceptions(&self) -> BTreeMap<PlayerId, JsonValue>;
    fn initial_public_snapshot(&self) -> JsonValue;
    fn compute_state_hash(&self) -> [u8; 32];
    fn advance(&mut self, actions: BTreeMap<PlayerId, JsonValue>) -> TickOutput;
    fn current_tick(&self) -> u64;
}
```

### 4.2 Protocolo Motor <-> Worker (`agentrix-engine/1`)

* **Inicialización (Worker -> Engine):**
```json
{"type":"init","matchId":"m-101","seed":42,"players":["p1","p2"]}
```

* **Respuesta Ready & Tick 0 (Engine -> Worker):**
```json
{
  "type": "ready",
  "tick": 0,
  "stateHash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "perceptions": {
    "p1": {"myself": {"x": 100.0, "y": 200.0, "angle": 0.0, "health": 100}},
    "p2": {"myself": {"x": 800.0, "y": 200.0, "angle": 3.14, "health": 100}}
  },
  "publicSnapshot": {
    "tick": 0,
    "entities": [
      {"id": "p1", "type": "ship", "x": 100.0, "y": 200.0, "rot": 0.0, "hp": 100},
      {"id": "p2", "type": "ship", "x": 800.0, "y": 200.0, "rot": 3.14, "hp": 100}
    ],
    "events": []
  }
}
```

* **Paso de Simulación (Worker -> Engine):**
```json
{
  "type": "step",
  "tick": 0,
  "actions": {
    "p1": {"thrust": "FORWARD", "turn": "NONE", "shoot": true, "shield": false},
    "p2": {"thrust": "NONE", "turn": "LEFT", "shoot": false, "shield": false}
  }
}
```

* **Resultado del Tick (Engine -> Worker):**
```json
{
  "type": "tick_result",
  "tick": 1,
  "stateHash": "9b12d...",
  "finished": false,
  "perceptions": { "p1": {}, "p2": {} },
  "publicSnapshot": { "tick": 1, "entities": [], "events": [] }
}
```

### 4.3 Protocolo Worker <-> Bot (`agentrix-bot/1`)

* **Handshake Inicial (Worker -> Bot):**
```json
{"type":"init","protocol":"agentrix-bot/1","matchId":"m-101","playerId":"p1","gameId":"starfighter","seed":42}
```

* **Confirmación (Bot -> Worker):**
```json
{"type":"ready","name":"HunterV1"}
```

* **Percepción (Worker -> Bot):**
```json
{"type":"perception","tick":0,"data":{"myself":{"x":100.0,"y":200.0,"angle":0.0,"health":100}}}
```

* **Acción (Bot -> Worker):**
```json
{"type":"action","tick":0,"data":{"thrust":"FORWARD","turn":"NONE","shoot":true,"shield":false}}
```

* **Cierre (Worker -> Bot):**
```json
{"type":"end","reason":"eliminated","winnerId":"p2"}
```

---

## 5. CORRECCIONES QUIRÚRGICAS EN GO

### 5.1 Gestión de Tick y Timeout Rígido (`backend/internal/bot/session.go`)

```go
package bot

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type BotSession struct {
	cmd     *exec.Cmd
	scanner *bufio.Scanner
	encoder *json.Encoder
}

type BotEnvelope struct {
	Type string          `json:"type"`
	Tick uint64          `json:"tick"`
	Data json.RawMessage `json:"data"`
}

func (s *BotSession) Step(ctx context.Context, tick uint64, perception json.RawMessage) (json.RawMessage, error) {
	out := BotEnvelope{
		Type: "perception",
		Tick: tick,
		Data: perception,
	}
	if err := s.encoder.Encode(&out); err != nil {
		return nil, fmt.Errorf("error al serializar percepción: %w", err)
	}

	resCh := make(chan BotEnvelope, 1)
	errCh := make(chan error, 1)

	go func() {
		if !s.scanner.Scan() {
			if err := s.scanner.Err(); err != nil {
				errCh <- err
			} else {
				errCh <- fmt.Errorf("EOF prematuro del bot")
			}
			return
		}
		var incoming BotEnvelope
		if err := json.Unmarshal(s.scanner.Bytes(), &incoming); err != nil {
			errCh <- err
			return
		}
		resCh <- incoming
	}()

	select {
	case <-ctx.Done():
		// Terminación forzosa para evitar arrastre de buffers corruptos
		_ = s.cmd.Process.Kill()
		return nil, fmt.Errorf("timeout en tick %d: proceso eliminado", tick)
	case err := <-errCh:
		_ = s.cmd.Process.Kill()
		return nil, fmt.Errorf("fallo de comunicación en tick %d: %w", tick, err)
	case resp := <-resCh:
		if resp.Type != "action" {
			return nil, fmt.Errorf("tipo de mensaje inesperado: %s", resp.Type)
		}
		if resp.Tick != tick {
			return nil, fmt.Errorf("tick mismatch: esperado %d, recibido %d", tick, resp.Tick)
		}
		return resp.Data, nil
	}
}
```

### 5.2 Resolución Dinámica de Agentes (`backend/internal/worker/executor.go`)

```go
package worker

import (
	"fmt"
	"os"
	"path/filepath"
	"gopkg.in/yaml.v3"
)

type Manifest struct {
	ID     string `yaml:"id"`
	Agents struct {
		Reference []struct {
			ID   string `yaml:"id"`
			Path string `yaml:"path"`
		} `yaml:"reference"`
	} `yaml:"agents"`
}

func ResolveAgentFallback(gameDir string, slot int) (string, error) {
	manifestPath := filepath.Join(gameDir, "manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("no se pudo leer manifest: %w", err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("manifest corrupto: %w", err)
	}

	if len(m.Agents.Reference) == 0 {
		return "", fmt.Errorf("el juego %s no declara agentes de referencia", m.ID)
	}

	chosen := m.Agents.Reference[slot%len(m.Agents.Reference)]
	fullPath := filepath.Join(gameDir, chosen.Path)
	return fullPath, nil
}
```

---

## 6. FORMATO DE REPLAY NDJSON Y COLA POSTGRESQL

### 6.1 Estructura del Replay (`.ndjson`)

```json
{"recordType":"header","gameId":"starfighter","matchId":"m-101","version":"1.0.0","seed":42,"tickRate":60,"players":[{"id":"p1","name":"Hunter"},{"id":"p2","name":"Evasive"}]}
{"recordType":"frame","tick":0,"stateHash":"a3f...","entities":[{"id":"p1","x":100.0,"y":200.0,"rot":0.0,"hp":100,"shield":false},{"id":"p2","x":800.0,"y":200.0,"rot":3.14,"hp":100,"shield":false}],"projectiles":[],"events":[]}
{"recordType":"frame","tick":1,"stateHash":"b8d...","entities":[{"id":"p1","x":102.0,"y":200.0,"rot":0.0,"hp":100,"shield":false},{"id":"p2","x":798.0,"y":200.0,"rot":3.14,"hp":100,"shield":false}],"projectiles":[{"id":"b-1","x":110.0,"y":200.0,"vx":600.0,"vy":0.0}],"events":[{"type":"laser_fired","actorId":"p1"}]}
{"recordType":"footer","totalTicks":450,"winnerId":"p1","reason":"opponent_destroyed","scores":{"p1":1200.0,"p2":150.0}}
```

### 6.2 Cola Concurrente en PostgreSQL

```sql
CREATE TABLE match_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id VARCHAR(64) UNIQUE NOT NULL,
    game_id VARCHAR(32) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'queued',
    worker_id VARCHAR(64),
    lease_until TIMESTAMPTZ,
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Consulta de reserva atómica sin contención:
UPDATE match_jobs
SET status = 'leased',
    worker_id = $1,
    lease_until = NOW() + INTERVAL '2 minutes',
    retry_count = retry_count + 1
WHERE id = (
    SELECT id FROM match_jobs
    WHERE (status = 'queued') OR (status = 'leased' AND lease_until < NOW())
    ORDER BY created_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
RETURNING id, match_id, game_id;
```

---

## 7. ARNÉS DE BENCHMARK: AVIAN2D VS RAPIER2D

Para validar determinismo y rendimiento antes de consolidar extensiones Gym/RL, se ejecutará el benchmark en `engine/benchmarks`:

```text
Configuración:
- 100 naves dinámicas
- 500 proyectiles activos
- Timestep fijo: dt = 1/60s
- 10,000 ticks consecutivos
- Semilla fija: 0x5EEDCAFE
```

| Criterio | Avian2D (MVP Actual) | Rapier2D (`enhanced-determinism`) |
| --- | --- | --- |
| **Integración Bevy** | Nativa directa en ECS | Requiere puente/sincronización |
| **Determinismo Multiplataforma** | Local confiable; no garantizado entre CPUs | Documentado formalmente (IEEE 754) |
| **Rendimiento Headless** | Alto | Muy alto (optimizado en C/Rust) |
| **Uso en Gymnasium/RL** | Requiere Bevy en cada instancia | Se puede desacoplar sin ECS |
| **Decisión Técnica** | **Mantener para el MVP de Starfighter** | **Reservar para Gym Vectorizado nativo** |

---

## 8. PLAN DE IMPLEMENTACIÓN POR FASES

### Fase 1: Estabilización Inmediata del Core (Corte 1)

* [x] Aplicar corrección de Tick 0 en `BotSession` (`backend/internal/bot/session.go`).
* [x] Implementar `ResolveAgentFallback` en Go apuntando al manifest de Starfighter.
* [x] Configurar terminación con `SIGKILL` ante timeouts de bots.
* [x] Añadir target `build-engines` al `Makefile` para compilar `bin/starfighter-engine`.
* [x] Ejecutar prueba de integración Go-Rust con 50 ticks continuos entre Hunter y Evasive.


### Fase 2: Replay Autorizado y Persistencia (Corte 2)

* [ ] Exportar `publicSnapshot` en cada tick desde `starfighter-engine`.
* [ ] Implementar streaming NDJSON en el Worker de Go y compresión Zstd al finalizar.
* [ ] Conectar la reserva de partidas en PostgreSQL con `FOR UPDATE SKIP LOCKED`.

### Fase 3: Visor Web de Starfighter (Corte 3)

* [ ] Retirar el visualizador de cuadrícula 10x10 de Arena Básica.
* [ ] Crear renderer en React con Canvas 2D/PixiJS consumiendo el flujo NDJSON.
* [ ] Implementar barra de tiempo, scrubbing, pausa y velocidades (0.5x, 1x, 2x, 4x).

### Fase 4: Admisión de Bots y Sandboxing (Corte 4)

* [ ] Endpoint para recepción de archivos ZIP (`agentrix.json` + `bot.py`).
* [ ] Validación estática previa y ejecución de tick de prueba (dry-run).
* [ ] Configurar aislamiento rootless OCI sin acceso a red y sistema de archivos de solo lectura.
