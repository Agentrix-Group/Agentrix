# Cómo subir un bot con red neuronal

Guía para participantes. La decisión que la respalda es
[ADR-0014](../decisions/0014-neural-network-bots.md); ante una diferencia,
mandan el ADR y el código (`src/service/bundle.go`,
`src/executor/ml_runtime.go`).

## Empezar desde una plantilla

La página **Mis agentes** ofrece dos plantillas que se admiten y juegan tal
como se descargan:

| Plantilla | Modelo | Inferencia |
| --- | --- | --- |
| `starfighter-neural-onnx.zip` | `model/policy.onnx` | `onnxruntime` |
| `starfighter-neural-npz.zip` | `model/policy.npz` | `numpy` |

Las dos traen la misma red chica (MLP 6 → 8 → 8) con pesos armados a mano:
sirven para ver el circuito completo, no para ganar. Para usar tu propia red,
reemplazá el archivo de `model/` y adaptá `policy.py` a tus entradas y
salidas. Las fuentes están en `games/starfighter/examples/neural/`, y
`make neural-templates` regenera los ZIP.

## Qué lleva el ZIP

```text
agentrix.json        manifiesto (obligatorio)
bot.py               punto de entrada (obligatorio)
policy.py            otros módulos .py, opcionales, en la raíz
model/policy.onnx    archivos de modelo, solo dentro de model/
```

`agentrix.json`:

```json
{
  "name": "MiBotNeuronal",
  "entrypoint": "bot.py",
  "protocol_version": "1.0",
  "runtime": "python-ml-cpu"
}
```

- `runtime` puede ser `python-stdlib` (por defecto, solo la biblioteca
  estándar) o `python-ml-cpu`. Un bot que importa `numpy`, `onnxruntime` o
  `safetensors` necesita `python-ml-cpu`.
- No se aceptan otros campos en el manifiesto.

| Regla | Límite |
| --- | --- |
| Tamaño del ZIP y del contenido descomprimido | 50 MB |
| Cada archivo de modelo | 20 MB |
| Cada módulo `.py` y `agentrix.json` | 1 MB |
| Cantidad de archivos | 64 |
| Raíz | `agentrix.json` y módulos `.py` con nombre de identificador Python |
| `model/` | `.onnx`, `.safetensors`, `.npz` o `.json`; se permiten subcarpetas |

Validación de los modelos al subir:

- `.npz` no puede contener arrays de objetos: se rechazan los pickles.
- `.safetensors` debe tener una cabecera y tipos de dato válidos.
- `.json` debe ser JSON válido.
- `.onnx` se comprueba cargándolo durante la admisión.

Al elegir el ZIP, el formulario muestra el runtime y la lista de archivos, y
avisa antes de subir si el paquete rompe alguna de estas reglas.

## El runtime `python-ml-cpu`

| Componente | Versión |
| --- | --- |
| Python | 3.12.14 |
| numpy | 2.5.3 |
| onnxruntime | 1.30.0 (CPU) |
| safetensors | 0.8.0 |

- No hay red, no se pueden instalar paquetes y la carpeta del bot, montada
  en `/bot`, es de solo lectura. Los modelos se abren con rutas relativas
  (`model/policy.onnx`), porque el bot arranca con `/bot` como directorio de
  trabajo.
- Cada bot tiene **un núcleo de CPU y 1 GB de RAM**. Configurá la inferencia
  con un hilo, por ejemplo con `intra_op_num_threads = 1` e
  `inter_op_num_threads = 1` en `onnxruntime`.
- El primer turno tiene **10 s** para cargar el modelo y responder; los
  siguientes, el límite normal por tick.
- Si el proceso del bot termina durante una partida (excepción no capturada,
  memoria agotada), el bot queda descalificado y su nave sale de la partida.

## Exportar un modelo

- **PyTorch → ONNX:** `torch.onnx.export(model, ejemplo, "policy.onnx",
  input_names=["features"], output_names=["logits"])`. Usá un opset que
  soporte onnxruntime 1.30.
- **numpy → NPZ:** `np.savez("policy.npz", w1=w1, b1=b1, ...)`. En el bot,
  cargalo con `np.load(..., allow_pickle=False)`.
- **safetensors:** `safetensors.numpy.save_file({"w1": w1, ...},
  "policy.safetensors")`.

`games/starfighter/examples/neural/make_models.py` genera los tres formatos
con los mismos pesos y sirve de ejemplo.

## Qué pasa al subirlo

1. La API revisa el ZIP con las reglas de arriba y lo guarda. La versión
   aparece como **Validando**.
2. El worker corre el bot en el sandbox, con su runtime, durante dos ticks de
   prueba.
3. La versión pasa a **Admitido** o a **Rechazado**. En el historial, **Ver
   causa** muestra el motivo del rechazo.

Solo una versión admitida puede jugar partidas.

| Motivo del rechazo | Qué hacer |
| --- | --- |
| `loading the model and answering the first tick took longer than 10s` | Achicá el modelo o cargalo más rápido |
| `answering tick N took longer than …` | La inferencia por tick es lenta: un hilo y un modelo más chico |
| `exceeded the 1024 MB memory limit` | Reducí el tamaño del modelo o de los buffers |
| `the bot process stopped: …` | Aparece la última línea de error de tu bot, por ejemplo un `ModuleNotFoundError` |
| `python-ml-cpu is not installed on this worker` | Problema de la plataforma, no de tu bot: avisá al administrador |

## Probarlo en tu máquina

Con el repositorio de Agentrix:

```bash
make bot-runtimes                     # arma python-ml-cpu en bin/runtimes
make neural-templates                 # regenera las plantillas
AGENTRIX_REQUIRE_SANDBOX=1 go test -run WebNeuralTemplates ./src/executor/
```

El test admite las dos plantillas en el sandbox y las hace jugar una partida
real contra un bot Ace.
