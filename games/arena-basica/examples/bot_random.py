#!/usr/bin/env python3
"""
Example Agent: Random Wanderer
Picks random valid actions in Arena Basica.
"""

import sys
import json
import random

def main():
    if len(sys.argv) < 3:
        # Fallback default action
        print(json.dumps({"type": "REST"}))
        return

    player_id = sys.argv[1]
    try:
        state = json.loads(sys.argv[2])
    except Exception:
        print(json.dumps({"type": "REST"}))
        return

    me = state.get("players", {}).get(player_id, {})
    if not me or not me.get("alive"):
        print(json.dumps({"type": "REST"}))
        return

    actions = ["UP", "DOWN", "LEFT", "RIGHT", "REST"]
    if me.get("energy", 0) >= 15:
        actions.append("ATTACK")
    if me.get("energy", 0) >= 10:
        actions.append("SHIELD")

    choice = random.choice(actions)
    print(json.dumps({"type": choice, "debug_message": "Random decision"}))

if __name__ == "__main__":
    main()
