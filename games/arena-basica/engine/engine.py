"""
Arena Basica - Reference Game Engine Simulator
Deterministic grid battle implementation for bot development and verification.
"""

import json
import random
from typing import Dict, List, Any

class ArenaBasica:
    def __init__(self, players: List[str], seed: int = 42, max_ticks: int = 100):
        self.players = players
        self.seed = seed
        self.max_ticks = max_ticks
        self.rng = random.Random(seed)
        self.grid_width = 10
        self.grid_height = 10
        self.tick = 0
        self.done = False
        self.winner = None
        self.events = []

        spawns = [(1, 1), (8, 8), (1, 8), (8, 1)]
        self.player_states = {}
        for i, p_id in enumerate(players):
            spawn = spawns[i % len(spawns)]
            self.player_states[p_id] = {
                "id": p_id,
                "x": spawn[0],
                "y": spawn[1],
                "hp": 100,
                "max_hp": 100,
                "energy": 100,
                "shielded": False,
                "alive": True,
                "score": 0
            }

    def get_state(self) -> Dict[str, Any]:
        return {
            "tick": self.tick,
            "grid_width": self.grid_width,
            "grid_height": self.grid_height,
            "players": self.player_states,
            "events": self.events,
            "done": self.done,
            "winner": self.winner
        }

    def step(self, actions: Dict[str, Dict[str, Any]]) -> Dict[str, Any]:
        if self.done:
            return self.get_state()

        self.tick += 1
        self.events = []

        # Reset shields
        for p in self.player_states.values():
            if p["alive"]:
                p["shielded"] = False

        # Phase 1: Shields & Rest
        for p_id, act in actions.items():
            p = self.player_states.get(p_id)
            if not p or not p["alive"]:
                continue
            act_type = act.get("type", "REST")
            if act_type == "SHIELD" and p["energy"] >= 10:
                p["shielded"] = True
                p["energy"] -= 10
                self.events.append(f"{p_id} raised shield")
            elif act_type == "REST":
                p["energy"] = min(100, p["energy"] + 25)
                p["hp"] = min(p["max_hp"], p["hp"] + 5)
                self.events.append(f"{p_id} rested")

        # Phase 2: Movements
        for p_id, act in actions.items():
            p = self.player_states.get(p_id)
            if not p or not p["alive"]:
                continue
            act_type = act.get("type", "REST")
            nx, ny = p["x"], p["y"]
            if act_type == "UP":
                ny = max(0, ny - 1)
            elif act_type == "DOWN":
                ny = min(self.grid_height - 1, ny + 1)
            elif act_type == "LEFT":
                nx = max(0, nx - 1)
            elif act_type == "RIGHT":
                nx = min(self.grid_width - 1, nx + 1)
            else:
                continue

            # Collision check
            collision = any(o["alive"] and o["id"] != p_id and o["x"] == nx and o["y"] == ny for o in self.player_states.values())
            if not collision:
                p["x"], p["y"] = nx, ny
                self.events.append(f"{p_id} moved to ({nx}, {ny})")

        # Phase 3: Attacks
        for p_id, act in actions.items():
            p = self.player_states.get(p_id)
            if not p or not p["alive"] or act.get("type") != "ATTACK":
                continue
            if p["energy"] < 15:
                continue
            p["energy"] -= 15

            for other_id, other in self.player_states.items():
                if other_id == p_id or not other["alive"]:
                    continue
                dist = abs(p["x"] - other["x"]) + abs(p["y"] - other["y"])
                if dist <= 1:
                    dmg = 5 if other["shielded"] else 25
                    other["hp"] -= dmg
                    p["score"] += dmg
                    self.events.append(f"{p_id} attacked {other_id} for {dmg} damage")
                    if other["hp"] <= 0:
                        other["hp"] = 0
                        other["alive"] = False
                        p["score"] += 50
                        self.events.append(f"{other_id} was defeated by {p_id}!")

        alive = [p for p in self.player_states.values() if p["alive"]]
        if len(alive) <= 1 or self.tick >= self.max_ticks:
            self.done = True
            if len(alive) == 1:
                self.winner = alive[0]["id"]
                alive[0]["score"] += 100
                self.events.append(f"Winner: {self.winner}")

        return self.get_state()
