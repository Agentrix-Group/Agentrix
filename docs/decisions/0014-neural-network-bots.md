# ADR-0014: Bots con redes neuronales

## Estado

accepted

## Fecha

2026-09-22

## Contexto

Un bot de Agentrix es hoy un ZIP con exactamente `agentrix.json` y `bot.py`
(1 MiB cada uno, 2 MiB en total). La plataforma guarda y monta solo
`bot.py`, y el bot corre con el Python del sistema, sin dependencias: en
producción, Bubblewrap dentro de la imagen del worker (`debian` +
`python3`); con Podman, `python:3.11-slim` con 256 MB. Un bot que dependa de
pesos de una red neuronal no puede traerlos ni cargarlos.

Los artefactos se guardan en un volumen compartido entre API y worker
(`ARTIFACTS_DIR`), así que un paquete de varios archivos puede montarse sin
transferencias durante la partida.

## Decisión

### Paquete v2

- El ZIP contiene `agentrix.json`, uno o más módulos `.py` y, opcionalmente,
  archivos de modelo bajo `model/` con extensión `.onnx`, `.safetensors`,
  `.npz` o `.json`. El punto de entrada sigue siendo el declarado en el
  manifiesto (`bot.py`).
- `agentrix.json` agrega `runtime`: `python-stdlib` (por defecto, el actual)
  o `python-ml-cpu`.
- Límites: 50 MB por paquete, 20 MB por archivo de modelo, 1 MiB por módulo
  `.py`, sin directorios fuera de la raíz y `model/` y sin rutas relativas
  ni absolutas peligrosas. Contra ZIP bombs se revisa el tamaño declarado
  antes de descomprimir y la lectura se corta en el límite, así que la
  memoria nunca supera los límites del paquete. Se descartó un límite de
  razón de compresión: rechazaba pesos legítimos muy repetitivos.
- Validación estática en la admisión: `.json` debe ser JSON válido; `.npz`
  no puede contener arrays de objetos de Python (bloquea pickle);
  `.safetensors` debe tener cabecera y tipos de dato válidos; `.onnx` se
  valida por tamaño y, en la prueba de admisión, cargándolo.
- Se guarda el paquete completo, con el SHA-256 de cada archivo. El digest
  de artefacto de cada slot en la `ExecutionSpec` pasa a ser el del paquete.
- Los paquetes v1 (`agentrix.json` + `bot.py`) se siguen aceptando.

### Ejecución

- El sandbox monta la carpeta del paquete completa, en solo lectura, en
  `/bot`, tanto con Bubblewrap como con Podman.
- El runtime `python-ml-cpu` es un entorno Python fijo con `numpy`,
  `onnxruntime` y `safetensors` en versiones fijadas por hash. Se instala en
  la imagen del worker y en local con `make bot-runtimes`; nunca se
  instalan paquetes durante una partida y el bot no tiene red.
- Cada bot usa un solo hilo de CPU (paridad entre bots y reproducibilidad),
  con 1 GB de RAM, 10 s para cargar su modelo en `init` y los 2 s por tick
  actuales.
- La admisión ejecuta el bot con su runtime: debe cargar su modelo y
  responder un tick de prueba dentro de esos límites.

### Caída de un bot (decisión del 2026-09-22)

- Un bot cuyo proceso termina durante la partida (excepción no capturada,
  falta de memoria, salida del proceso) queda descalificado con causa
  `crash`: su nave sale en ese tick, como una descalificación por timeout
  (ADR-0013). Antes su nave quedaba quieta hasta ser destruida.
- No descalifican los fallos de la plataforma: un bot que no pudo arrancar
  (por ejemplo, runtime no instalado en el worker) o una partida cancelada
  siguen informándose como `crashed`.
- La admisión conserva los últimos 2 KB de stderr del bot para explicar el
  rechazo.

### Fuera de alcance

Entrenar redes con el motor real (entorno Gym) sigue la etapa Gym del
roadmap. Este ADR permite subir y ejecutar redes, no entrenarlas.

## Criterios de aceptación

| Fase | Criterio |
| --- | --- |
| N0 | Este ADR; suites Go, integración y web en verde antes de empezar |
| N1 | Paquete v2 y admisión: un test de rechazo por caso (ruta `../`, extensión no permitida, tamaño, ZIP bomb, `.npz` con pickle, JSON inválido, cabecera `.safetensors` corrupta); los paquetes v1 siguen admitiéndose |
| N2 | Un bot que lee su modelo desde `/bot` juega una partida real con replay sellado; escribir en `/bot` falla |
| N3 | Bots de ejemplo con `.onnx`, `.safetensors` y `.npz` juegan una partida de 5; RAM máxima medida y bajo el límite; un bot que excede la RAM queda descalificado; el runtime `python-stdlib` no cambia |
| N4 | La admisión rechaza, con el motivo, un modelo demasiado lento y uno demasiado pesado |
| N5 | Formulario web con runtime y archivos; guía y plantillas ONNX y NPZ que se admiten y juegan |
| N6 | Integración completa con `-race`: 0 fallas y 0 saltados; partida de 5 con bots ML y clásicos mezclados y replay sellado |

## Consecuencias

- La imagen del worker crece con el runtime ML (numpy + onnxruntime).
- Construir el runtime requiere descargar paquetes una vez; después se
  construye con versiones y hashes fijos.
- Bubblewrap no limita memoria por sí solo; el mecanismo de límite de RAM
  se mide y se elige en N3 (riesgo: reservas de memoria virtual de
  onnxruntime por encima del uso real).
- Los modelos son datos, pero el código del bot sigue siendo arbitrario: la
  seguridad sigue dependiendo del sandbox, no del formato del modelo. Aun
  así se prefieren formatos que no ejecutan código al cargarse.
