#!/usr/bin/env python3
"""bot_random.py -- reference strategy: uniformly random action every tick.

Thrust/turn/shoot/shield are each sampled independently, with no regard
for the perception it receives. Serves as the comparison floor: any
other reference bot should beat this one, and a game balance change that
makes bot_random.py suddenly competitive against bot_hunter.py or
bot_evasive.py is a signal worth investigating.

Speaks the real Agentrix agent protocol (ATD-007) over stdin/stdout, one
JSON object per line -- no dependency beyond the Python standard library.
"""
import json
import random
import sys

PROTOCOL_VERSION = "1.0"


def send(msg: dict) -> None:
    sys.stdout.write(json.dumps(msg) + "\n")
    sys.stdout.flush()


def read() -> dict:
    line = sys.stdin.readline()
    if not line:
        sys.exit(0)
    return json.loads(line)


def random_action(rng: random.Random) -> dict:
    return {
        "thrust": rng.choice(["FORWARD", "OFF", "BRAKE"]),
        "turn": rng.choice(["LEFT", "RIGHT", "NONE"]),
        "shoot": rng.random() < 0.3,
        "shield": rng.random() < 0.1,
    }


def main() -> None:
    init = read()
    assert init["type"] == "init"
    # Seeded from the explicit protocol inputs so that the same match seed
    # and slot always produce the same action sequence (reproducible replays).
    rng = random.Random(f"{init['seed']}:{init['player_id']}")

    while True:
        msg = read()
        if msg["type"] == "end":
            return
        assert msg["type"] == "perception"
        send({"type": "action", "tick": msg["tick"], "action": random_action(rng)})


if __name__ == "__main__":
    main()
