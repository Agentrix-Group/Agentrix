# ADR-001: Motor de Juego Externo en Rust (Bevy + Rapier) y Protocolo de Integración

## Estado
Aprobado (2026-09-15)

## Contexto
El prototipo inicial de Agentrix incluía una implementación provisional en Go del juego de demostración (`ArenaBasicaEngine`) ejecutada dentro del mismo proceso del backend. Esta solución acoplaba la simulación al runtime de Go, impedía el uso de motores de física avanzados y mezclaba la autoridad de las reglas de juego con el plano de control administrativo.

Para los concursos oficiales de Agentrix, se requiere un entorno de simulación de alto rendimiento, determinista, seguro y con capacidades de física y colisiones espaciales 2D/3D robustas, manteniendo el plano de control administrativo en Go.

## Decisión Arquitectónica

Se adopta formalmente la separación estricta entre el **Plano de Control y Supervisión (Agentrix en Go)** y el **Motor de Juego Autoritativo (Rust con Bevy y Rapier)**.

### 1. Responsabilidades por Componente

#### A. Agentrix (Go)
- Cuentas de usuario e identidad.
- Autenticación (JWT/sesiones) y autorización RBAC basada en capacidades.
- Gestión de concursos, categorías, inscripciones y calendario.
- Gestión de agentes y envíos de código (submissions).
- Programación de partidas y cola de trabajos.
- Supervisión de procesos (sandboxing de agentes y subproceso del motor).
- Persistencia de negocio y almacenamiento de resultados y replays.
- Transmisión de eventos de simulación sanitizados hacia espectadores.

#### B. Supervisor de Ejecución (Worker en Go)
- Iniciar y detener el subproceso del motor.
- Iniciar, aislar y supervisar los procesos no confiables de los agentes.
- Aplicar límites de recursos (CPU, memoria, timeout) a los agentes.
- Enviar percepciones permitidas a cada agente y recolectar sus acciones.
- Validar estructuralmente los mensajes de los agentes.
- Detectar timeouts, crashes y violaciones de protocolo de los agentes.
- Entregar las acciones validadas al motor en cada tick.
- Recolectar del motor los eventos, hashes y resultados finales.
- Diferenciar rigurosamente fallos del motor (fallos de plataforma) de fallos de los agentes (sanciones/eliminaciones competitivas).

#### C. Motor Oficial (Rust + Bevy + Rapier)
- Ejecutable externo independiente ejecutado en modo *headless*.
- **Bevy**: administración del mundo, entidades, componentes, recursos, sistemas y ciclo de ticks con timestep fijo.
- **Rapier**: resolución de físicas, colisiones y consultas espaciales.
- Autoritativo respecto a la simulación física, las reglas del juego y el resultado competitivo.
- Cálculo de perspectivas y percepciones para cada participante.
- Emisión de eventos por tick y hashes de estado deterministas.
- **No** administra usuarios, sesiones, base de datos, colas ni red.
- **No** ejecuta agentes de los participantes ni accede a sus scripts/binarios.

#### D. Agentes Participantes
- Procesos externos aislados, no confiables.
- Reciben del supervisor únicamente la percepción autorizada para su slot.
- Retornan una acción dentro del tiempo límite asignado.
- No tienen acceso a red, sistema de archivos sensible ni a la memoria de otros bots.

#### E. Frontend Web
- Recibe eventos y snapshots proyectados vía API / canal en vivo.
- Reconstruye la representación visual en el cliente (ej. con canvas / PixiJS).
- No transmite imágenes renderizadas por el motor.
- No decide resultados ni ejecuta lógica autoritativa.

### 2. Protocolo de Comunicación (`agentrix-engine/1`)
- La comunicación entre Agentrix y el motor se realiza mediante **JSON Lines** sobre los descriptores estándar del proceso:
  - `stdin`: Agentrix envía comandos al motor (`initialize_match`, `advance_tick`, `finish_match`, `shutdown`).
  - `stdout`: El motor responde **exclusivamente** con mensajes del protocolo (`engine_ready`, `match_initialized`, `tick_completed`, `match_completed`, `engine_error`, `shutdown_ack`). No se permite texto libre en `stdout`.
  - `stderr`: Reservado exclusivamente para logs estructurados o diagnósticos del motor.
- Todos los mensajes están tipados, versionados y contenidos en un sobre común con identificador de partida y secuencia monotónica.

### 3. Retiro del Motor Provisional Go
- La implementación en memoria de `ArenaBasicaEngine` en Go queda clasificada como código experimental provisional y se retira en favor de la abstracción `EngineClient` y el motor externo.
- Para propósitos de pruebas automatizadas y desarrollo local en ausencia del binario Rust, se provee un `fake-engine` independiente que implementa fielmente la especificación `agentrix-engine/1`.

## Consecuencias
- **Positivas:**
  - Desacoplamiento total entre el ciclo de vida del backend y las físicas del juego.
  - Posibilidad de desarrollar el motor en Rust sin dependencias de la base de datos o el código de Agentrix.
  - Simulación reproducible y de alto rendimiento.
  - Aislamiento de seguridad: si el motor falla o un bot crashea, Agentrix permanece en línea.
- **Negativas / Desafíos:**
  - Sobrecarga de serialización JSON Lines sobre pipes IPC (mitigada mediante mensajes compactos y tamaños de línea acotados).
  - Necesidad de coordinar procesos hijos y gestionar pipes/señales POSIX de manera robusta.
