"""Bot de ejemplo: red en model/policy.onnx, inferencia con onnxruntime.

Un solo hilo de inferencia: el sandbox asigna un núcleo por bot.
"""
import numpy as np
import onnxruntime as ort

import policy

options = ort.SessionOptions()
options.intra_op_num_threads = 1
options.inter_op_num_threads = 1
SESSION = ort.InferenceSession("model/policy.onnx", options, providers=["CPUExecutionProvider"])


def infer(inputs):
    x = np.asarray([inputs], dtype=np.float32)
    return SESSION.run(["logits"], {"features": x})[0][0].tolist()


policy.run(infer)
