"""Genera model/policy.{npz,safetensors,onnx} con los mismos pesos.

Herramienta de desarrollo (no corre en el sandbox). Requiere numpy,
safetensors y onnx:

    uv run --with numpy==2.5.3 --with safetensors==0.8.0 --with onnx==1.23.0 \
        python games/starfighter/examples/neural/make_models.py

Los pesos están armados a mano para que la política tenga sentido sin
entrenamiento: gira hacia el objetivo, se acerca si está lejos, frena si
está cerca y dispara cuando está alineada y con el arma lista.
"""
import pathlib

import numpy as np
import onnx
from onnx import TensorProto, helper, numpy_helper
from safetensors.numpy import save_file

OUT = pathlib.Path(__file__).parent / "model"

# Entradas: [sin, cos, dist, ready, energy, seen]
w1 = np.zeros((8, 6), dtype=np.float32)
b1 = np.zeros(8, dtype=np.float32)
w1[0, 0] = 1.0                    # h0 = relu(sin): objetivo a la izquierda
w1[1, 0] = -1.0                   # h1 = relu(-sin): objetivo a la derecha
w1[2, 2] = 1.0; b1[2] = -0.6      # h2 = relu(dist - 0.6): lejos
w1[3, 2] = -1.0; b1[3] = 0.35     # h3 = relu(0.35 - dist): cerca
w1[4, 1] = 1.0; b1[4] = -0.995    # h4 = relu(cos - 0.995): alineado
w1[5, 3] = 1.0                    # h5 = ready
w1[6, 5] = 1.0                    # h6 = seen
b1[7] = 1.0                       # h7 = 1 (sesgo)

w2 = np.zeros((8, 8), dtype=np.float32)
b2 = np.zeros(8, dtype=np.float32)
w2[0, 0] = 40.0                   # LEFT
w2[2, 1] = 40.0                   # RIGHT
w2[1, 7] = 2.0                    # NONE (umbral de giro)
w2[3, 2] = 40.0; w2[3, 7] = 0.5   # FORWARD si lejos
w2[4, 7] = 1.0                    # OFF por defecto
w2[5, 3] = 40.0                   # BRAKE si cerca
w2[6, 7] = 1.0                    # no disparar por defecto
w2[7, 4] = 400.0; w2[7, 5] = 1.0; w2[7, 6] = 1.0; b2[7] = -2.5  # disparar

OUT.mkdir(exist_ok=True)
weights = {"w1": w1, "b1": b1, "w2": w2, "b2": b2}
np.savez(OUT / "policy.npz", **weights)
save_file(weights, OUT / "policy.safetensors")

graph = helper.make_graph(
    [
        helper.make_node("Gemm", ["features", "w1", "b1"], ["z1"], transB=1),
        helper.make_node("Relu", ["z1"], ["h1"]),
        helper.make_node("Gemm", ["h1", "w2", "b2"], ["logits"], transB=1),
    ],
    "starfighter_policy",
    [helper.make_tensor_value_info("features", TensorProto.FLOAT, [1, 6])],
    [helper.make_tensor_value_info("logits", TensorProto.FLOAT, [1, 8])],
    [numpy_helper.from_array(v, name) for name, v in weights.items()],
)
model = helper.make_model(graph, opset_imports=[helper.make_opsetid("", 17)], producer_name="agentrix-example")
model.ir_version = 9
onnx.checker.check_model(model)
onnx.save(model, OUT / "policy.onnx")
print("written:", sorted(p.name for p in OUT.iterdir()))
