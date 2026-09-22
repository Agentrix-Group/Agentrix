#!/usr/bin/env python3
"""bot_ace.py -- reference strategy for free-for-all: attacks AND evades.

Designed for Starfighter 0.4.0 (2 to 5 ships, everyone against everyone,
ADR-0013). Every tick it decides between two modes:

- **Evade** when an enemy bullet is about to pass close: it computes the
  closest approach of every enemy bullet (relative position and velocity,
  as the perception gives them), moves sideways out of the path of the most
  urgent one and raises the shield only if the impact is imminent.
- **Attack** otherwise: it picks a target (nearest, preferring damaged
  ships), aims at the intercept point (leading the target's motion) and
  shoots when aligned, keeping a preferred distance instead of ramming.

Each copy gets a small deterministic "personality" (preferred distance,
aim tolerance) from its slot id, so five copies do not fly identically.

Engine facts it relies on (agentrix_engine src/lib.rs): bullets leave the
nose at 1500 u/s added to the ship velocity; the turn rate grows by
0.5 rad/s per tick up to 4 rad/s and decays by the same amount when not
turning; the arena is 2000 x 1000 centered at the origin; asteroids are
NOT in the perception, so they cannot be avoided.

Speaks the real Agentrix agent protocol (ATD-007) over stdin/stdout, one
JSON object per line -- no dependency beyond the Python standard library.
"""
import json
import math
import sys
import zlib

PROTOCOL_VERSION = "1.0"

TICK_SECONDS = 1.0 / 60.0
BULLET_SPEED = 1500.0
BULLET_RANGE_SECONDS = 90 * TICK_SECONDS
TURN_DECEL = 0.5 / TICK_SECONDS  # rad/s^2 while coasting
ARENA_HALF_W = 1000.0
ARENA_HALF_H = 500.0
WALL_MARGIN = 130.0

# A bullet is a threat if it will pass within this distance of the ship's
# center (hull ~25 u + bullet 3 u + margin) within THREAT_HORIZON seconds.
THREAT_MISS_DISTANCE = 42.0
THREAT_HORIZON = 0.55
SHIELD_HORIZON = 0.18
SHOOT_COST = 15.0
ENERGY_RESERVE = 12.0


def send(msg: dict) -> None:
    sys.stdout.write(json.dumps(msg) + "\n")
    sys.stdout.flush()


def read() -> dict:
    line = sys.stdin.readline()
    if not line:
        sys.exit(0)
    return json.loads(line)


def normalize_angle(a: float) -> float:
    while a > math.pi:
        a -= 2 * math.pi
    while a < -math.pi:
        a += 2 * math.pi
    return a


def dot(ax: float, ay: float, bx: float, by: float) -> float:
    return ax * bx + ay * by


class Ace:
    def __init__(self, slot_id: str) -> None:
        # Deterministic per-slot personality (crc32, not hash(): Python's
        # str hash is randomized per process).
        h = zlib.crc32(slot_id.encode("utf-8"))
        self.preferred_distance = 260.0 + (h % 5) * 45.0  # 260..440
        self.aim_slack = 0.9 + ((h >> 8) % 4) * 0.1  # 0.9..1.2
        self.prev_facing_angle = None
        self.omega = 0.0

    # -- steering -----------------------------------------------------------

    def steer(self, target_x: float, target_y: float, facing_angle: float, tolerance: float) -> str:
        """Turn toward a direction, coasting early when the current spin
        would already carry the nose there (avoids overshooting)."""
        diff = normalize_angle(math.atan2(target_y, target_x) - facing_angle)
        stopping = self.omega * self.omega / (2 * TURN_DECEL)
        if diff > tolerance:
            return "NONE" if self.omega > 0 and stopping >= diff else "LEFT"
        if diff < -tolerance:
            return "NONE" if self.omega < 0 and stopping >= -diff else "RIGHT"
        # Inside the window: cancel leftover spin.
        if self.omega > 1.0:
            return "RIGHT"
        if self.omega < -1.0:
            return "LEFT"
        return "NONE"

    # -- perception helpers -------------------------------------------------

    @staticmethod
    def most_urgent_threat(perception: dict):
        me = perception["player_id"]
        best = None
        for bullet in perception["bullets"]:
            if bullet["player_id"] == me:
                continue
            px, py = bullet["relative_position"]["x"], bullet["relative_position"]["y"]
            vx, vy = bullet["relative_velocity"]["x"], bullet["relative_velocity"]["y"]
            vv = dot(vx, vy, vx, vy)
            if vv < 1e-6:
                continue
            tca = -dot(px, py, vx, vy) / vv
            if tca <= 0.0 or tca > THREAT_HORIZON:
                continue
            mx, my = px + vx * tca, py + vy * tca
            miss = math.hypot(mx, my)
            if miss > THREAT_MISS_DISTANCE:
                continue
            if best is None or tca < best[0]:
                best = (tca, vx, vy, mx, my, miss)
        return best

    @staticmethod
    def intercept(rel_x: float, rel_y: float, rvx: float, rvy: float):
        """Point to aim at so a bullet (relative speed BULLET_SPEED) meets a
        target at rel with relative velocity rv. None if unreachable."""
        a = dot(rvx, rvy, rvx, rvy) - BULLET_SPEED * BULLET_SPEED
        b = 2 * dot(rel_x, rel_y, rvx, rvy)
        c = dot(rel_x, rel_y, rel_x, rel_y)
        t = None
        if abs(a) < 1e-6:
            if abs(b) > 1e-6:
                t = -c / b
        else:
            disc = b * b - 4 * a * c
            if disc >= 0:
                root = math.sqrt(disc)
                candidates = [x for x in ((-b - root) / (2 * a), (-b + root) / (2 * a)) if x > 0]
                t = min(candidates) if candidates else None
        if t is None or t <= 0 or t > BULLET_RANGE_SECONDS:
            return None
        return rel_x + rvx * t, rel_y + rvy * t

    def pick_target(self, rivals: list):
        def score(rival):
            rel = rival["relative_position"]
            distance = math.hypot(rel["x"], rel["y"])
            # Up to 150 u of "distance discount" for a badly damaged ship.
            return distance - 1.5 * (100.0 - rival["health"])

        return min(rivals, key=score)

    # -- decision -----------------------------------------------------------

    def choose_action(self, perception: dict) -> dict:
        myself = perception["myself"]
        fx, fy = myself["facing"]["x"], myself["facing"]["y"]
        facing_angle = math.atan2(fy, fx)
        if self.prev_facing_angle is not None:
            self.omega = normalize_angle(facing_angle - self.prev_facing_angle) / TICK_SECONDS
        self.prev_facing_angle = facing_angle

        pos = myself["position"]
        vel = myself["velocity"]
        can_shoot = (
            myself["remaining_bullet_cooldown"] <= 0
            and myself["energy"] >= SHOOT_COST + ENERGY_RESERVE
        )

        # Opportunistic shot at whatever is in front, valid in every mode.
        aim = None
        target = self.pick_target(perception["rivals"]) if perception["rivals"] else None
        if target is not None:
            rel, rv = target["relative_position"], target["relative_velocity"]
            aim = self.intercept(rel["x"], rel["y"], rv["x"], rv["y"]) or (rel["x"], rel["y"])

        def aligned_shot() -> bool:
            if aim is None or not can_shoot:
                return False
            distance = math.hypot(aim[0], aim[1])
            diff = abs(normalize_angle(math.atan2(aim[1], aim[0]) - facing_angle))
            return diff < max(0.04, math.atan2(18.0 * self.aim_slack, distance))

        # 1. Evade the most urgent enemy bullet.
        threat = self.most_urgent_threat(perception)
        if threat is not None:
            tca, vx, vy, mx, my, miss = threat
            speed = math.hypot(vx, vy)
            px, py = -vy / speed, vx / speed  # perpendicular to the bullet path
            if miss > 8.0:
                # Keep moving to the side the bullet already passes on.
                side = 1.0 if dot(px, py, mx, my) >= 0 else -1.0
            else:
                # Dead center: take the side that needs the smaller turn.
                side = 1.0 if dot(px, py, fx, fy) >= 0 else -1.0
            dodge_x, dodge_y = px * side, py * side
            turn = self.steer(dodge_x, dodge_y, facing_angle, 0.25)
            along = dot(fx, fy, dodge_x, dodge_y)
            thrust = "FORWARD" if along > -0.2 else "BRAKE"
            shield = tca < SHIELD_HORIZON and myself["energy"] > 5.0
            return {"thrust": thrust, "turn": turn, "shoot": aligned_shot(), "shield": shield}

        # 2. Stay away from the walls (the engine reflects, which kills speed).
        near_wall = abs(pos["x"]) > ARENA_HALF_W - WALL_MARGIN or abs(pos["y"]) > ARENA_HALF_H - WALL_MARGIN
        heading_out = dot(vel["x"], vel["y"], pos["x"], pos["y"]) > 0
        if near_wall and heading_out:
            turn = self.steer(-pos["x"], -pos["y"], facing_angle, 0.3)
            return {"thrust": "FORWARD", "turn": turn, "shoot": aligned_shot(), "shield": False}

        # 3. No one on radar: patrol toward the center.
        if target is None:
            turn = self.steer(-pos["x"], -pos["y"], facing_angle, 0.3)
            far_from_center = math.hypot(pos["x"], pos["y"]) > 200.0
            return {
                "thrust": "FORWARD" if far_from_center else "OFF",
                "turn": turn,
                "shoot": False,
                "shield": False,
            }

        # 4. Attack: face the intercept point, keep the preferred distance.
        rel = target["relative_position"]
        distance = math.hypot(rel["x"], rel["y"])
        turn = self.steer(aim[0], aim[1], facing_angle, 0.05)
        closing_speed = -dot(target["relative_velocity"]["x"], target["relative_velocity"]["y"], rel["x"], rel["y"]) / max(distance, 1.0)
        facing_target = abs(normalize_angle(math.atan2(rel["y"], rel["x"]) - facing_angle)) < 0.8
        if distance > self.preferred_distance + 80.0 and facing_target:
            thrust = "FORWARD"
        elif distance < self.preferred_distance - 80.0 and closing_speed > 0:
            thrust = "BRAKE"
        else:
            thrust = "OFF"
        return {"thrust": thrust, "turn": turn, "shoot": aligned_shot(), "shield": False}


def main() -> None:
    init = read()
    assert init["type"] == "init"
    ace = Ace(str(init.get("player_id", "")))

    while True:
        msg = read()
        if msg["type"] == "end":
            return
        assert msg["type"] == "perception"
        action = ace.choose_action(msg["perception"])
        send({"type": "action", "tick": msg["tick"], "action": action})


if __name__ == "__main__":
    main()
