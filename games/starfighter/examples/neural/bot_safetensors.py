"""Bot de ejemplo: pesos en model/policy.safetensors, inferencia con numpy."""
import numpy as np
from safetensors.numpy import load_file

import policy

WEIGHTS = load_file("model/policy.safetensors")

policy.run(lambda inputs: policy.mlp_numpy(np, WEIGHTS, inputs))
