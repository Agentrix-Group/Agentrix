#!/usr/bin/env python3
"""
Example Agent: Hunter Aggressor
Chases nearest opponent and attacks when adjacent.

Speaks the persistent bot<->executor protocol (JSON Lines over
stdin/stdout, one process for the whole match): handshake -> init ->
one perception/action exchange per tick -> end. See
src/executor/bot_session.go on the Go side for the exact contract.
"""

import sys
import json

PROTOCOL_VERSION = "1.0"


def send(msg):
    sys.stdout.write(json.dumps(msg) + "\n")
    sys.stdout.flush()


def choose_action(player_id, perception):
    players = perception.get("players", {}) if isinstance(perception, dict) else {}
    me = players.get(player_id)
    if not me or not me.get("alive", True):
        return {"type": "REST"}

    my_x, my_y = me.get("x", 0), me.get("y", 0)
    my_hp = me.get("hp", 0)
    my_energy = me.get("energy", 0)

    closest_opp = None
    min_dist = 9999
    for opp_id, opp in players.items():
        if opp_id == player_id or not opp.get("alive", True):
            continue
        dist = abs(my_x - opp.get("x", 0)) + abs(my_y - opp.get("y", 0))
        if dist < min_dist:
            min_dist = dist
            closest_opp = opp

    if not closest_opp:
        return {"type": "REST"}

    if min_dist <= 1:
        if my_hp < 20 and my_energy >= 10:
            return {"type": "SHIELD", "debug_message": "Low HP shield"}
        if my_energy >= 15:
            return {"type": "ATTACK", "debug_message": "Adjacent attack"}
        return {"type": "REST", "debug_message": "Resting for energy"}

    if my_energy < 20:
        return {"type": "REST", "debug_message": "Recovering energy"}

    dx = closest_opp.get("x", 0) - my_x
    dy = closest_opp.get("y", 0) - my_y
    if abs(dx) > abs(dy):
        move = "RIGHT" if dx > 0 else "LEFT"
    else:
        move = "DOWN" if dy > 0 else "UP"

    return {"type": move, "debug_message": f"Chasing target (dist={min_dist})"}


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
            pass
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
