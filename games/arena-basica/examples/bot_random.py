#!/usr/bin/env python3
"""
Example Agent: Random Wanderer
Picks random valid actions in Arena Basica.

Speaks the persistent bot<->executor protocol (JSON Lines over
stdin/stdout, one process for the whole match): handshake -> init ->
one perception/action exchange per tick -> end. See
src/executor/bot_session.go on the Go side for the exact contract.
"""

import sys
import json
import random

PROTOCOL_VERSION = "1.0"


def send(msg):
    sys.stdout.write(json.dumps(msg) + "\n")
    sys.stdout.flush()


def choose_action(player_id, perception):
    players = perception.get("players", {}) if isinstance(perception, dict) else {}
    me = players.get(player_id, {})
    if not me or not me.get("alive", True):
        return {"type": "REST"}

    actions = ["UP", "DOWN", "LEFT", "RIGHT", "REST"]
    if me.get("energy", 0) >= 15:
        actions.append("ATTACK")
    if me.get("energy", 0) >= 10:
        actions.append("SHIELD")

    choice = random.choice(actions)
    return {"type": choice, "debug_message": "Random decision"}


def main():
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            msg = json.loads(line)
        except json.JSONDecodeError:
            continue

        msg_type = msg.get("type")

        if msg_type == "handshake":
            send({"type": "handshake_ack", "protocol_version": PROTOCOL_VERSION})
        elif msg_type == "init":
            pass  # No per-match setup needed for a random policy.
        elif msg_type == "perception":
            tick = msg.get("tick", 0)
            player_id = msg.get("player_id", "")
            perception = msg.get("perception") or {}
            action = choose_action(player_id, perception)
            send({"type": "action", "tick": tick, "action": action})
        elif msg_type == "end":
            break


if __name__ == "__main__":
    main()
