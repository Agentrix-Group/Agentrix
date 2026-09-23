"""Bot de ejemplo: pesos en model/policy.npz, inferencia con numpy."""
import numpy as np

import policy

with np.load("model/policy.npz", allow_pickle=False) as data:
    WEIGHTS = {name: data[name] for name in ("w1", "b1", "w2", "b2")}

policy.run(lambda inputs: policy.mlp_numpy(np, WEIGHTS, inputs))
