#!/usr/bin/env python3
"""
Example Agent: Hunter Aggressor
Chases nearest opponent and attacks when adjacent.
"""

import sys
import json

def main():
    if len(sys.argv) < 3:
        print(json.dumps({"type": "REST"}))
        return

    player_id = sys.argv[1]
    try:
        state = json.loads(sys.argv[2])
    except Exception:
        print(json.dumps({"type": "REST"}))
        return

    players = state.get("players", {})
    me = players.get(player_id)
    if not me or not me.get("alive"):
        print(json.dumps({"type": "REST"}))
        return

    my_x, my_y = me.get("x", 0), me.get("y", 0)
    my_hp = me.get("hp", 0)
    my_energy = me.get("energy", 0)

    # Locate closest alive opponent
    closest_opp = None
    min_dist = 9999

    for opp_id, opp in players.items():
        if opp_id == player_id or not opp.get("alive"):
            continue
        dist = abs(my_x - opp.get("x", 0)) + abs(my_y - opp.get("y", 0))
        if dist < min_dist:
            min_dist = dist
            closest_opp = opp

    if not closest_opp:
        print(json.dumps({"type": "REST"}))
        return

    # Adjacent: Attack or Shield
    if min_dist <= 1:
        if my_hp < 20 and my_energy >= 10:
            print(json.dumps({"type": "SHIELD", "debug_message": "Low HP shield"}))
            return
        if my_energy >= 15:
            print(json.dumps({"type": "ATTACK", "debug_message": "Adjacent attack"}))
            return
        print(json.dumps({"type": "REST", "debug_message": "Resting for energy"}))
        return

    # If low energy, recover before rushing
    if my_energy < 20:
        print(json.dumps({"type": "REST", "debug_message": "Recovering energy"}))
        return

    # Navigate towards target
    dx = closest_opp.get("x", 0) - my_x
    dy = closest_opp.get("y", 0) - my_y

    if abs(dx) > abs(dy):
        move = "RIGHT" if dx > 0 else "LEFT"
    else:
        move = "DOWN" if dy > 0 else "UP"

    print(json.dumps({"type": move, "debug_message": f"Chasing target (dist={min_dist})"}))

if __name__ == "__main__":
    main()
