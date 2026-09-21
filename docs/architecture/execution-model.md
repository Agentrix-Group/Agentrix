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

## ExecutionSpec

`ScheduleRun` construye un `ExecutionSpec` (`agentrix-execution-spec/1`) dentro de la misma transacción que crea el `MatchRun` y su `match_job`. El spec fija:

- semilla y `tick_rate` racional;
- límites de ejecución;
- juego, versión, config y `config_hash`;
- digest y versión del motor;
- perfil de sandbox;
- por cada slot: `submission_id`, clave y SHA-256 del artefacto, runtime y entrypoint.

El spec se sella con JSON canónico y SHA-256. El worker lo parsea de forma estricta (`ParseExecutionSpec`), verifica el hash, verifica el digest de cada artefacto y del motor, y **no vuelve a leer** manifests ni submissions vivas.

## Procesos de bots

Cada slot inicia un proceso Python que recibe `init` una vez y se reutiliza durante toda la partida. Cada percepción exige una acción del mismo tick. Los turnos corren en paralelo, con deadline de escritura y de respuesta. Las causas de término de un bot son `timeout`, `crash`, `invalid_action`, `protocol_error` y `resource_limit`. Un slot sin percepción en un tick se reporta como `inactive`.

## Aislamiento

`AGENTRIX_SANDBOX` selecciona el runtime:

| Runtime | Frontera | Uso |
| --- | --- | --- |
| `bubblewrap` | `--unshare-all` (incluye la red), `--cap-drop ALL`, `--clearenv`, solo `/usr` en lectura, `/tmp` tmpfs con tamaño fijo, bot montado en solo lectura, `prlimit` **dentro** del namespace (AS, CPU, NPROC, FSIZE, NOFILE, core 0) | Por defecto; contenedor del worker |
| `podman` | `--network none`, `--read-only`, `--cap-drop ALL`, `no-new-privileges`, memoria, pids, cpu y ulimits | Alternativa rootless |
| `direct` | ninguna | Solo `MODE=dev`; la configuración lo rechaza en cualquier otro modo |

Si el runtime no pasa la prueba de arranque, el worker **no arranca** (falla cerrado). Los tests adversariales de `src/executor/sandbox_test.go` cubren fork bomb, bomba de memoria, bucle infinito, límite de CPU, inundación de salida, tamaño de archivo, red, archivos del host y secretos del entorno, procesos huérfanos y fallo de arranque, y pasan contra el sandbox real. En un contenedor, Bubblewrap necesita `seccomp=unconfined`, `apparmor=unconfined` y `systempaths=unconfined` (ver `docker-compose.yml`).

## Tiempo de simulación

`tick_rate` es una razón exacta `{numerator, denominator}` (60/1 = 60 Hz). El motor calcula `step = denominator / numerator` sin aproximación en milisegundos. El replay guarda la misma razón y el visor deriva de ella el intervalo por tick. `tick_hz` y `fixed_timestep_ms` ya no existen en ningún contrato.

El timestep afecta a la simulación y al replay, no a un reloj de pared. Cada bot conserva un presupuesto de respuesta independiente.

## Cola y propiedad de un trabajo

1. `ReserveNext` toma el job pendiente más antiguo con el digest de motor del worker (`FOR UPDATE SKIP LOCKED`) y le asigna un lease y un `fencing_token` salido de una secuencia.
2. El worker renueva el lease con heartbeats cada `LeaseTTL/3`. Si pierde el lease, aborta y descarta el replay en staging.
3. `StartRun`, `Heartbeat`, `CommitRun` y `FailRun` bloquean el job y verifican estado, propietario, token, run, partida y lease vigente. Un worker cercado recibe `ErrFenced` y no escribe nada.
4. `CommitRun` escribe en una sola transacción: resultados validados contra los slots, metadata del replay (`staging`) y el estado terminal de job, run y partida.
5. El reaper marca `timed_out` los leases vencidos. Un reintento crea **un run nuevo** (attempt + 1); los intentos anteriores quedan auditables.

## Fallos

| Origen | Tratamiento |
| --- | --- |
| Bot | Consecuencia competitiva dentro del resultado del motor; sin reintento |
| Motor, sandbox o infraestructura | Run `failed`; si la clase es reintentable, nuevo run |
| Worker o host | El lease vence; el reaper cierra el run y otro worker toma el reintento con un token nuevo |
| Persistencia | Sin commit no hay resultado; el staging huérfano lo recoge el GC |
| Publicación del replay | El outbox reintenta con backoff; el resultado ya es oficial |
