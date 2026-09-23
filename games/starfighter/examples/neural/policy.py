"""Plantilla de política neuronal para Starfighter (ADR-0014).

Una red chica (MLP 6 -> 8 -> 8, ReLU) decide cada tick a partir de
características calculadas desde la percepción. Los pesos se generan con
make_models.py y se guardan en tres formatos equivalentes; cada bot de
ejemplo usa uno:

- bot_npz.py          model/policy.npz          (numpy)
- bot_safetensors.py  model/policy.safetensors  (safetensors + numpy)
- bot_onnx.py         model/policy.onnx         (onnxruntime)

Entradas (FEATURES):
    sin y cos del ángulo entre la nariz y el rival más cercano,
    distancia / 800, arma lista (0/1), energía / 100, rival visible (0/1)
Salidas (8 logits):
    giro [LEFT, NONE, RIGHT], empuje [FORWARD, OFF, BRAKE], disparo [no, sí]

No es un bot fuerte: muestra cómo cargar un modelo desde /bot/model y usarlo
en el protocolo de Agentrix.
"""
import json
import math
import sys

FEATURES = 6
TURNS = ("LEFT", "NONE", "RIGHT")
THRUSTS = ("FORWARD", "OFF", "BRAKE")


def features(perception):
    myself = perception["myself"]
    fx, fy = myself["facing"]["x"], myself["facing"]["y"]
    ready = 1.0 if myself["remaining_bullet_cooldown"] <= 0 else 0.0
    energy = myself["energy"] / 100.0
    rivals = perception["rivals"]
    if not rivals:
        # Sin rival a la vista: apuntar al centro de la arena.
        px, py = -myself["position"]["x"], -myself["position"]["y"]
        seen = 0.0
    else:
        nearest = min(rivals, key=lambda r: math.hypot(r["relative_position"]["x"], r["relative_position"]["y"]))
        px, py = nearest["relative_position"]["x"], nearest["relative_position"]["y"]
        seen = 1.0
    distance = math.hypot(px, py) or 1.0
    # Ángulo desde la nariz hacia el objetivo (positivo = girar a la izquierda).
    angle = math.atan2(fx * py - fy * px, fx * px + fy * py)
    return [math.sin(angle), math.cos(angle), distance / 800.0, ready, energy, seen]


def decode(logits):
    """Convierte los 8 logits en una acción de Agentrix (argmax por grupo)."""
    def pick(values, names):
        return names[max(range(len(values)), key=lambda i: values[i])]

    return {
        "turn": pick(logits[0:3], TURNS),
        "thrust": pick(logits[3:6], THRUSTS),
        "shoot": logits[7] > logits[6],
        "shield": False,
    }


def mlp_numpy(np, weights, inputs):
    """Forward de la MLP con numpy (lo usan los bots npz y safetensors)."""
    x = np.asarray(inputs, dtype=np.float32)
    h = np.maximum(weights["w1"] @ x + weights["b1"], 0.0)
    return (weights["w2"] @ h + weights["b2"]).tolist()


def read():
    line = sys.stdin.readline()
    if not line:
        sys.exit(0)
    return json.loads(line)


def send(message):
    sys.stdout.write(json.dumps(message) + "\n")
    sys.stdout.flush()


def run(policy):
    """Bucle del protocolo: `policy(features) -> logits`."""
    init = read()
    assert init["type"] == "init"
    while True:
        message = read()
        if message["type"] == "end":
            return
        action = decode(policy(features(message["perception"])))
        send({"type": "action", "tick": message["tick"], "action": action})
