# Modelo de ejecución

## Autoridad y secuencia

Rust es la autoridad sobre estado, reglas, percepciones, eventos, puntaje y resultado. Go coordina procesos y transporta JSON opaco.

El ciclo canónico comienza en cero:

```text
state[0]
  -> private perception[0]
  -> bot action[0]
  -> engine step[0]
  -> state[1]
  -> private perception[1]
```

El replay contiene el snapshot público de `state[0]` antes de cualquier acción y después un snapshot por estado resultante. Bot, worker, motor y replay deben usar el mismo índice.

## Procesos de bots

**Implementado:** cada slot inicia un proceso Python, envía `init` una vez y reutiliza el proceso para todos los ticks. Cada percepción exige una acción del mismo tick. Timeout, salida inválida o desfase desconectan el proceso.

**Parcial:** el supervisor clasifica estados como `valid`, `timeout`, `invalid_output`, `crashed` o `disqualified`, pero la aplicación de límites del sistema operativo no está completa.

## Aislamiento

### Estado actual

Si `bwrap` está disponible, el bot se ejecuta con red separada, `/proc` aislado y el filesystem completo del host montado en solo lectura. Si no está disponible, se ejecuta directamente con `python3`.

Esto no constituye aislamiento de producción porque:

- expone nombres y contenido legible del host;
- hereda el entorno del proceso;
- no aplica límites comprobables de CPU, memoria, PIDs o output;
- no distingue desarrollo de producción;
- falla abierto cuando falta Bubblewrap.

### Objetivo aprobado

El runtime oficial será rootless, con filesystem mínimo, red deshabilitada, entorno limpio, UID sin privilegios y límites de CPU, memoria, PIDs, tiempo y salida. Producción debe rechazar el trabajo antes de iniciar un bot si esa frontera no está disponible.

## Tiempo de simulación

El objetivo aprobado es **60 Hz exactos**. La configuración actual contiene a la vez `tick_hz: 60` y `fixed_timestep_ms: 17`; el motor calcula `1000 / 17`, es decir, aproximadamente 58,82 Hz. Hasta corregir el contrato y el código, la documentación debe tratarlo como una divergencia conocida y no como 60 Hz exactos.

El timestep afecta simulación y replay, no un reloj de pared que otorgue ventaja a hardware más rápido. Cada bot conserva un presupuesto independiente de respuesta.

## Cola y propiedad de un trabajo

**Implementado:** PostgreSQL reserva con `FOR UPDATE SKIP LOCKED`, registra propietario y recupera trabajos cuya marca temporal supera dos minutos.

**Parcial:** esa marca no se renueva, no existe fencing token y resultados/replay no se confirman mediante una operación idempotente cercada. Una partida larga puede perder su reserva y ser reclamada por otro worker.

### Objetivo aprobado

1. La reserva entrega un lease y un token monotónico de fencing.
2. El worker renueva el lease mientras conserva capacidad.
3. Toda escritura final verifica trabajo, intento y fencing token.
4. El commit de replay, resultados y estado de partida es idempotente.
5. Un intento que perdió el lease no puede publicar ni sobrescribir evidencia.

## Fallos

| Origen | Tratamiento objetivo |
| --- | --- |
| Bot | Consecuencia competitiva definida por el motor; no reintento automático |
| Motor | Intento técnico inválido; API permanece disponible |
| Worker/host | Lease expira y otro worker recupera con nuevo fencing token |
| Persistencia | No se confirma resultado sin commit durable |
| Visor | La partida continúa; replay se consulta después |
