#!/usr/bin/env python3
"""bot_ace.py -- reference strategy for free-for-all: attacks AND evades.

Designed for Starfighter 0.4.0 (2 to 5 ships, everyone against everyone,
ADR-0013). Every tick:

1. **Plan an attack.** Keep a target (switching only for a clearly better
   one). While the gun reloads, orbit the target at a preferred distance so
   the ship is never a still target; when the gun is about to be ready,
   turn to the intercept point.
2. **Check the plan against every threat.** Simulate the next 0.6 s with
   the engine's own dynamics for the planned maneuver and for the other
   eight (thrust forward/off/brake x turn left/none/right), against every
   enemy bullet in flight, a virtual bullet from every rival currently
   aiming at this ship, and the arena walls. Keep the plan if it is safe;
   otherwise take the safe maneuver closest to it, or the one with the most
   clearance. Raise the shield only when no maneuver avoids the impact.
3. **Shoot only when the shot connects.** Simulate this ship's bullet
   against every rival moving at constant velocity and fire only if it
   passes within the hull of one of them, inside an effective range.

Each copy derives a small deterministic personality (preferred distance,
orbit side) from its slot id, so five copies do not fly identically.

Engine facts it relies on (agentrix_engine src/lib.rs, src/physics.rs):
thrust adds ~22.2 u/s per tick along the nose (800000 N on 600 kg); speed
is capped at 500 u/s; "OFF" applies drag, "BRAKE" decelerates at the same
rate; the turn rate changes 0.5 rad/s per tick up to 4 rad/s and decays by
0.5 rad/s per tick when not turning; bullets leave the nose (+24 u) at
1500 u/s added to the ship velocity, live 90 ticks and are 3 u in radius;
the gun reloads in 40 ticks; the arena is 2000 x 1000 centered at the
origin. Asteroids are NOT in the perception, so they cannot be avoided.

Speaks the real Agentrix agent protocol (ATD-007) over stdin/stdout, one
JSON object per line -- no dependency beyond the Python standard library.
"""
import json
import math
import sys
import zlib

PROTOCOL_VERSION = "1.0"

DT = 1.0 / 60.0
THRUST_DV = 800000.0 / 600.0 * DT  # u/s gained per tick of thrust
MAX_SPEED = 500.0
DRAG_COEF = 0.02
DRAG_EXP = 1.5
TURN_STEP = 0.5  # rad/s per tick
MAX_TURN = 4.0
BULLET_SPEED = 1500.0
BULLET_OFFSET = 24.0
BULLET_LIFETIME = 90 * DT
RELOAD_TICKS = 40
SHOOT_COST = 15.0
ARENA_HALF_W = 1000.0
ARENA_HALF_H = 500.0

HORIZON_TICKS = 36  # 0.6 s of look-ahead for maneuvers
HIT_RADIUS = 22.0  # bullet center closer than this to the hull center hits
SAFE_CLEARANCE = 38.0  # a maneuver is "safe" above this clearance
WALL_MARGIN = 40.0
SHOT_RADIUS = 14.0  # our shot must pass this close to count as a hit
EFFECTIVE_RANGE = 720.0
AIM_THREAT_ANGLE = 0.14  # a rival whose nose is this close to us may fire
AIM_THREAT_RANGE = 700.0
ENERGY_RESERVE = 10.0  # kept for the shield

MANEUVERS = [(t, r) for t in ("FORWARD", "OFF", "BRAKE") for r in ("LEFT", "NONE", "RIGHT")]


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


def vec(d: dict):
    return d["x"], d["y"]


class Ace:
    def __init__(self, slot_id: str) -> None:
        # crc32, not hash(): Python's str hash is randomized per process.
        h = zlib.crc32(slot_id.encode("utf-8"))
        self.preferred_distance = 480.0 + (h % 5) * 40.0  # 480..640
        self.orbit_side = 1.0 if (h >> 3) & 1 else -1.0
        self.prev_angle = None
        self.omega = 0.0
        self.target_id = None

    # -- dynamics -------------------------------------------------------------

    def simulate(self, x, y, vx, vy, angle, omega, thrust, turn, ticks):
        """Yield the ship position for each of the next ticks, holding one
        maneuver, with the engine's own per-tick update order."""
        for _ in range(ticks):
            if turn == "LEFT":
                omega = min(MAX_TURN, omega + TURN_STEP)
            elif turn == "RIGHT":
                omega = max(-MAX_TURN, omega - TURN_STEP)
            elif omega != 0.0:
                omega -= math.copysign(min(TURN_STEP, abs(omega)), omega)
            speed = math.hypot(vx, vy)
            if thrust == "FORWARD":
                vx += -math.sin(angle) * THRUST_DV
                vy += math.cos(angle) * THRUST_DV
            elif thrust == "OFF":
                factor = 1.0 - DRAG_COEF * (speed / MAX_SPEED) ** DRAG_EXP
                vx, vy = vx * factor, vy * factor
            elif speed > 1.0:
                cut = min(THRUST_DV, speed)
                vx -= vx / speed * cut
                vy -= vy / speed * cut
            speed = math.hypot(vx, vy)
            if speed > MAX_SPEED:
                vx, vy = vx / speed * MAX_SPEED, vy / speed * MAX_SPEED
            angle += omega * DT
            x += vx * DT
            y += vy * DT
            yield x, y

    # -- threats --------------------------------------------------------------

    @staticmethod
    def threats(perception, x, y, vx, vy):
        """World-frame (x, y, vx, vy) of every enemy bullet in flight, and
        separately a virtual bullet leaving the nose of each rival that is
        aiming at us now (it may or may not fire)."""
        me = perception["player_id"]
        real, virtual = [], []
        for b in perception["bullets"]:
            if b["player_id"] == me:
                continue
            px, py = vec(b["relative_position"])
            rvx, rvy = vec(b["relative_velocity"])
            real.append((x + px, y + py, vx + rvx, vy + rvy))
        out = virtual
        for r in perception["rivals"]:
            px, py = vec(r["relative_position"])
            distance = math.hypot(px, py)
            if distance < 1.0 or distance > AIM_THREAT_RANGE:
                continue
            fx, fy = vec(r["facing"])
            # Angle between the rival's nose and the line from it to us.
            if math.acos(max(-1.0, min(1.0, (-px * fx - py * fy) / distance))) > AIM_THREAT_ANGLE:
                continue
            rvx, rvy = vec(r["relative_velocity"])
            out.append((
                x + px + fx * BULLET_OFFSET,
                y + py + fy * BULLET_OFFSET,
                vx + rvx + fx * BULLET_SPEED,
                vy + rvy + fy * BULLET_SPEED,
            ))
        return real, virtual

    def clearance(self, state, maneuver, threats):
        """Smallest distance to any threat over the horizon, penalized near
        the walls. Returns (clearance, first tick at which it happens)."""
        x, y, vx, vy, angle, omega = state
        worst, worst_tick = 1e9, HORIZON_TICKS
        for tick, (sx, sy) in enumerate(self.simulate(x, y, vx, vy, angle, omega, *maneuver, HORIZON_TICKS), 1):
            t = tick * DT
            for bx, by, bvx, bvy in threats:
                if t > BULLET_LIFETIME:
                    continue
                d = math.hypot(sx - (bx + bvx * t), sy - (by + bvy * t))
                if d < worst:
                    worst, worst_tick = d, tick
            wall = min(ARENA_HALF_W - abs(sx), ARENA_HALF_H - abs(sy))
            if wall < WALL_MARGIN:
                # Hitting a wall reflects the ship and kills its speed.
                worst = min(worst, SAFE_CLEARANCE - 1.0 + max(wall, 0.0) / WALL_MARGIN)
        return worst, worst_tick

    # -- targeting ------------------------------------------------------------

    def pick_target(self, rivals):
        def score(r):
            distance = math.hypot(*vec(r["relative_position"]))
            return distance - 1.5 * (100.0 - r["health"]) + (60.0 if r["shield_active"] else 0.0)

        best = min(rivals, key=score)
        current = next((r for r in rivals if r["player_id"] == self.target_id), None)
        if current is not None and score(current) < score(best) + 120.0:
            best = current  # hysteresis: do not hop between targets
        self.target_id = best["player_id"]
        return best

    @staticmethod
    def intercept(px, py, rvx, rvy):
        a = rvx * rvx + rvy * rvy - BULLET_SPEED * BULLET_SPEED
        b = 2 * (px * rvx + py * rvy)
        c = px * px + py * py
        disc = b * b - 4 * a * c
        if abs(a) < 1e-6 or disc < 0:
            return px, py
        root = math.sqrt(disc)
        times = [t for t in ((-b - root) / (2 * a), (-b + root) / (2 * a)) if t > 0]
        if not times:
            return px, py
        t = min(times)
        return px + rvx * t, py + rvy * t

    @staticmethod
    def shot_connects(perception, fx, fy):
        """Would a bullet fired now pass within SHOT_RADIUS of a rival that
        keeps its current velocity? Evaluated relative to this ship."""
        for r in perception["rivals"]:
            px, py = vec(r["relative_position"])
            if math.hypot(px, py) > EFFECTIVE_RANGE:
                continue
            rvx, rvy = vec(r["relative_velocity"])
            # Relative gap: target - bullet = (P - f*24) + (V - f*1500) t
            gx, gy = px - fx * BULLET_OFFSET, py - fy * BULLET_OFFSET
            wx, wy = rvx - fx * BULLET_SPEED, rvy - fy * BULLET_SPEED
            ww = wx * wx + wy * wy
            t = max(0.0, min(BULLET_LIFETIME, -(gx * wx + gy * wy) / ww)) if ww > 1e-9 else 0.0
            if math.hypot(gx + wx * t, gy + wy * t) < SHOT_RADIUS:
                return True
        return False

    def steer(self, dx, dy, angle, tolerance):
        """Turn toward a world direction, coasting early when the current
        spin already carries the nose there (no overshoot)."""
        diff = normalize_angle(math.atan2(dy, dx) - (angle + math.pi / 2))
        stopping = self.omega * self.omega / (2 * TURN_STEP / DT)
        if diff > tolerance:
            return "NONE" if self.omega > 0 and stopping >= diff else "LEFT"
        if diff < -tolerance:
            return "NONE" if self.omega < 0 and stopping >= -diff else "RIGHT"
        if self.omega > 1.0:
            return "RIGHT"
        if self.omega < -1.0:
            return "LEFT"
        return "NONE"

    @staticmethod
    def turn_ticks(error):
        """Ticks needed to rotate the nose by `error` radians from rest
        (accelerate at 30 rad/s^2 up to 4 rad/s, then brake)."""
        error = abs(error)
        accel = TURN_STEP / DT
        if error < MAX_TURN * MAX_TURN / accel:
            seconds = 2 * math.sqrt(error / accel)
        else:
            seconds = error / MAX_TURN + MAX_TURN / accel
        return seconds / DT

    def plan(self, perception, x, y, vx, vy, angle, fx, fy):
        """Returns ((thrust, turn), mode) with mode "aim" or "move"."""
        myself = perception["myself"]
        rivals = perception["rivals"]
        if not rivals:
            # Patrol toward the center; the radar is omnidirectional.
            far = math.hypot(x, y) > 220.0
            return ("FORWARD" if far else "OFF", self.steer(-x, -y, angle, 0.3)), "move"

        target = self.pick_target(rivals)
        px, py = vec(target["relative_position"])
        rvx, rvy = vec(target["relative_velocity"])
        distance = max(math.hypot(px, py), 1.0)
        ux, uy = px / distance, py / distance

        reload_left = myself["remaining_bullet_cooldown"]
        ax, ay = self.intercept(px, py, rvx, rvy)
        aim_error = normalize_angle(math.atan2(ay, ax) - (angle + math.pi / 2))
        # Start turning back early enough to be on target when the gun is.
        if reload_left <= self.turn_ticks(aim_error) + 6 or distance > EFFECTIVE_RANGE:
            turn = self.steer(ax, ay, angle, 0.03)
            facing_target = (fx * ux + fy * uy) > 0.8
            closing = -(rvx * ux + rvy * uy)
            if distance > self.preferred_distance + 90.0 and facing_target:
                thrust = "FORWARD"
            elif distance < self.preferred_distance - 110.0 and closing > 60.0:
                thrust = "BRAKE"
            else:
                thrust = "OFF"
            return (thrust, turn), "aim"

        # Reloading: orbit the target (tangential speed + distance keeping).
        side = self.orbit_side
        tx, ty = -uy * side, ux * side
        # Turn the orbit around if it runs into a wall.
        if abs(x + tx * 150.0) > ARENA_HALF_W - 120.0 or abs(y + ty * 150.0) > ARENA_HALF_H - 120.0:
            self.orbit_side = -self.orbit_side
            tx, ty = -tx, -ty
        radial = max(-1.0, min(1.0, (distance - self.preferred_distance) / 150.0))
        dvx = tx * 280.0 + ux * radial * 200.0 - vx
        dvy = ty * 280.0 + uy * radial * 200.0 - vy
        if math.hypot(dvx, dvy) < 70.0:
            # Already moving as wanted: pre-aim at the target.
            return ("OFF", self.steer(ax, ay, angle, 0.05)), "move"
        turn = self.steer(dvx, dvy, angle, 0.2)
        aligned = (fx * dvx + fy * dvy) / math.hypot(dvx, dvy) > 0.7
        return (("FORWARD" if aligned else "OFF"), turn), "move"

    # -- decision -------------------------------------------------------------

    def choose_action(self, perception: dict) -> dict:
        myself = perception["myself"]
        x, y = vec(myself["position"])
        vx, vy = vec(myself["velocity"])
        fx, fy = vec(myself["facing"])
        # Engine convention: nose at (-sin θ, cos θ).
        angle = math.atan2(-fx, fy)
        if self.prev_angle is not None:
            self.omega = normalize_angle(angle - self.prev_angle) / DT
        self.prev_angle = angle

        planned, mode = self.plan(perception, x, y, vx, vy, angle, fx, fy)
        real, virtual = self.threats(perception, x, y, vx, vy)
        state = (x, y, vx, vy, angle, self.omega)
        chosen, shield = planned, False
        if mode == "aim":
            # Aiming has priority: only real bullets may change the plan, and
            # only its thrust, so the nose stays on target. If the impact is
            # unavoidable anyway, shield and keep shooting.
            planned_clear, planned_tick = self.clearance(state, planned, real)
            if planned_clear < SAFE_CLEARANCE:
                options = [(m, *self.clearance(state, m, real)) for m in MANEUVERS if m[1] == planned[1]]
                best = max(options, key=lambda o: o[1])
                if best[1] >= SAFE_CLEARANCE:
                    chosen = best[0]
                else:
                    shield = best[1] < HIT_RADIUS and best[2] <= 12 and myself["energy"] > 3.0
        else:
            # Moving while reloading: avoid real bullets and the line of fire
            # of every rival aiming at us.
            threats = real + virtual
            planned_clear, _ = self.clearance(state, planned, threats)
            if planned_clear < SAFE_CLEARANCE:
                scored = [(m, *self.clearance(state, m, threats)) for m in MANEUVERS]
                safe = [o for o in scored if o[1] >= SAFE_CLEARANCE]
                if safe:
                    # Closest safe maneuver to the plan: same turn first.
                    safe.sort(key=lambda o: (o[0][1] != planned[1], o[0][0] != planned[0], -o[1]))
                    chosen = safe[0][0]
                else:
                    best = max(scored, key=lambda o: o[1])
                    chosen = best[0]
                    real_clear, real_tick = self.clearance(state, chosen, real)
                    shield = real_clear < HIT_RADIUS and real_tick <= 12 and myself["energy"] > 3.0

        can_shoot = (
            myself["remaining_bullet_cooldown"] <= 0
            and myself["energy"] >= SHOOT_COST + ENERGY_RESERVE
            and self.shot_connects(perception, fx, fy)
        )
        return {"thrust": chosen[0], "turn": chosen[1], "shoot": can_shoot, "shield": shield}


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
