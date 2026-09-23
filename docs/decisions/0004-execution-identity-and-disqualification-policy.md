# ADR-0004: Identidad de ejecución y política de descalificación

> [!NOTE]
> La resolución de descalificación para partidas de más de dos jugadores y
> su implementación en el motor están en
> [ADR-0013](0013-starfighter-rapier-and-free-for-all.md); en 1 contra 1 el
> resultado sigue siendo el descrito aquí.
> Desde [ADR-0014](0014-neural-network-bots.md), la caída del proceso de un
> bot durante la partida (`crash`) también lo descalifica.

## Estado

accepted

## Fecha

2026-09-19

## Contexto

En una arquitectura con workers distribuidos y reintentos automáticos, una partida lógica (`match_id`) puede ser programada o reintentada múltiples veces si un worker reinicia o pierde conectividad. Para evitar escrituras duplicadas, colisiones y carreras de resultados, se requiere una identidad de ejecución inequívoca.

Asimismo, cuando los agentes participantes cometen infracciones (tiempo excedido, excepciones no capturadas, salida malformada o salida simultánea de ambos participantes), el sistema debe contar con una política determinista y simétrica para resolver el resultado.

## Decisión

### 1. Identidad de ejecución
- Toda ejecución física de una partida posee un `run_id` único (UUIDv4) independiente del `match_id` lógico.
- El modelo transaccional y los metadatos de replay registran:
  - `run_id`: Identificador único de la corrida.
  - `engine_digest`: SHA-256 inmutable del binario o imagen del motor de simulación.
  - `game_version`: Versión semántica del juego (e.g., `starfighter@0.3.0`).
  - `config_hash`: Hash SHA-256 canónico de los parámetros de configuración de la simulación.
  - `seed`: Semilla numérica inicial.
- Toda escritura de resultado y sellado de replay valida que el `run_id` y su `fencing_token` correspondan a la asignación activa.

### 2. Política de fallos y descalificación de agentes
- **Causas normalizadas:** Go y los componentes de observabilidad clasifican las incidencias en:
  - `timeout`: Excedió el tiempo límite por turno o de inicialización.
  - `crash`: Terminación inesperada del proceso, fallo de importación o excepción fatal.
  - `invalid_action`: Acción con sintaxis inválida o fuera del espacio de acción permitido.
  - `protocol_error`: Incumplimiento del contrato de sobres o envelopes.
  - `resource_limit`: OOM (memoria agotada), exceso de PIDs o cuota de salida excedida.
- **Resolución de descalificación simple:**
  - Si un agente es descalificado, el motor le asigna puntaje 0 y rango 2 (estado `eliminated` o `disqualified`).
  - El oponente no infractor es declarado ganador (`winner`), con puntaje 1 y rango 1.
  - La partida concluye con la razón correspondiente (`timeout`, etc.).
- **Resolución de doble descalificación simultánea:**
  - Si ambos agentes fallan o son descalificados en el mismo tick (o durante la inicialización), la resolución es estrictamente simétrica.
  - Ninguno de los dos jugadores obtiene la victoria (`winner: ""`).
  - Ambos reciben puntaje 0 y rango empatado, registrando la razón de terminación como `no_contest` o `double_disqualification`.
  - El motor Rust jamás debe asignar la victoria por orden de iteración de los slots.

## Consecuencias

- Desaparece la ambigüedad en los rankings y resultados ante fallos simultáneos.
- Todo resultado y replay sellado es rastreable a su corrida exacta, versión de motor y configuración física.
