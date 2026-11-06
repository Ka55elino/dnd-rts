"""Cavalry clips (art/skeletons/cavalry.py): the horse's legs move by phase formulas, the rider sits
(fixed leg angles) and holds the reins and the weapon with hand IK (targets ride with the rider's
chest). The weapon's business end points along `aim` (as in the humanoid stances).

Stances (dicts): weapon_wrist / weapon_elbow / weapon_aim for the right hand, rein_wrist / rein_elbow for
the left; rest-space points of the standing humanoid lifted by RIDER.
"""
import math
import numpy as np
from modelc import Cycle, Motion, Reverse, aim, point_on, place_hand
from modelc.math3d import cubic_bezier
from art.skeletons.cavalry import lift  # lift(p, sk): onto that skeleton's saddle

TAU = 2 * math.pi

SEAT = {
    "thigh.L": {"rx": -60, "ry": -22}, "thigh.R": {"rx": -60, "ry": 22},
    "shin.L": {"rx": 70}, "shin.R": {"rx": 70},
    "foot.L": {"rx": -10}, "foot.R": {"rx": -10},
}

# sword held up at the side, reins in the left hand in front of the saddle
SWORD = dict(weapon_wrist=(-0.24, -0.14, 1.12), weapon_elbow=(0.5, 1.0, -0.4), weapon_aim=(0.0, -0.25, 1.0),
             rein_wrist=(0.10, -0.32, 1.02), rein_elbow=(1.0, 0.4, -0.6))
# lance upright (resting on the stirrup side); couched: level under the arm, the tip forward
LANCE = dict(SWORD, weapon_wrist=(-0.22, -0.10, 1.08), weapon_aim=(0.0, -0.05, 1.0))
LANCE_COUCHED = dict(LANCE, weapon_wrist=(-0.18, -0.12, 1.22), weapon_elbow=(0.6, 0.8, -0.3), weapon_aim=(0.12, -1.0, -0.03))


def mix(a, b, u):
    return {k: tuple(x + (y - x) * u for x, y in zip(a[k], b[k])) for k in a}


def rider_hands(sk, pose, g, lean=0.0):
    w = point_on(sk, pose, "chest", lift(g["rein_wrist"], sk))
    place_hand(sk, pose, "L", w, g["rein_elbow"])
    w = point_on(sk, pose, "chest", lift(g["weapon_wrist"], sk))
    place_hand(sk, pose, "R", w, g["weapon_elbow"], hand_rot=aim((0, 0, -1), g["weapon_aim"], (1, 0, 0), (0, 0, 1)))
    return pose


def _pose(body=None, legs=None, rider=None):
    pose = {k: dict(v) for k, v in SEAT.items()}
    for d in (body or {}), (legs or {}), (rider or {}):
        for k, v in d.items():
            pose.setdefault(k, {}).update(v)
    return pose


def _legs(p, amp_up, amp_lo, phases):
    """Leg angles at phase p: each leg swings forward/back (sin), the lower leg folds while it swings
    forward (the front knee back, the hind hock forward)."""
    out = {}
    for name, ph in phases.items():
        s = math.sin(TAU * (p + ph))
        fold = max(0.0, math.cos(TAU * (p + ph)))
        front = name.startswith("f")
        up, lo = ("fleg", "fshin") if front else ("hleg", "hshin")
        side = name[-1]
        out[f"{up}.{side}"] = {"rx": -amp_up * s}
        out[f"{lo}.{side}"] = {"rx": (amp_lo if front else -amp_lo) * fold}
    return out


def idle(stance=SWORD, duration=4.0):
    def pose(p, sk):
        a = TAU * p
        P = _pose(body={"body": {"dz": 0.006 * math.sin(2 * a)}, "neck_h": {"rx": 4 * math.sin(a)},
                        "head_h": {"rx": 6 * math.sin(a + 1.0)}, "tail": {"rz": 8 * math.sin(2 * a)},
                        "chest": {"rx": 1.0 * math.sin(2 * a)}})
        return rider_hands(sk, P, stance)
    return Cycle(duration, pose)


def gait(duration, amp_up, amp_lo, phases, bob, pitch, ground_speed, stance=SWORD, lean=0.0):
    def pose(p, sk):
        a = TAU * p
        P = _pose(body={"body": {"dz": -bob * (0.5 + 0.5 * math.cos(2 * a)), "rx": pitch * math.sin(a)},
                        "neck_h": {"rx": -pitch * 1.5 * math.sin(a)}, "tail": {"rx": -10 - 8 * math.sin(a)},
                        "spine": {"rx": lean}, "chest": {"rx": lean * 0.5 - pitch * 0.6 * math.sin(a)}},
                  legs=_legs(p, amp_up, amp_lo, phases))
        return rider_hands(sk, P, stance)
    return Cycle(duration, pose, ground_speed=ground_speed)


def walk(stance=SWORD, scale=1.0):
    """Four-beat walk: hind left, fore left, hind right, fore right. scale: the mount's size (a smaller
    mount steps faster and covers less ground)."""
    return gait(1.1 * scale ** 0.5, 18, 45, {"hleg.L": 0.0, "fleg.L": 0.25, "hleg.R": 0.5, "fleg.R": 0.75},
                0.015 * scale, 1.5, 1.9 * scale ** 0.5, stance)


def run(stance=SWORD, scale=1.0):
    """Gallop: the hind legs together, then the forelegs; the body rocks."""
    return gait(0.55 * scale ** 0.5, 32, 75, {"hleg.L": 0.0, "hleg.R": 0.08, "fleg.L": 0.45, "fleg.R": 0.53},
                0.05 * scale, 6, 7.0 * scale ** 0.5, stance, lean=8)


def fly(stance=SWORD, duration=0.8, flap=45, ground_speed=None):
    """Flying (griffons): the wings beat, the legs are tucked, the body rises and falls with each beat."""
    def pose(p, sk):
        a = TAU * p
        beat = math.sin(a)
        P = _pose(body={"body": {"dz": 0.06 * beat, "rx": -3 * beat}, "neck_h": {"rx": 3 * beat},
                        "wing.L": {"ry": -flap * beat}, "wing.R": {"ry": flap * beat}, "tail": {"rx": -20},
                        "chest": {"rx": 2 * beat}},
                  legs={"fleg.L": {"rx": -35}, "fshin.L": {"rx": 70}, "fleg.R": {"rx": -35}, "fshin.R": {"rx": 70},
                        "hleg.L": {"rx": 40}, "hshin.L": {"rx": -50}, "hleg.R": {"rx": 40}, "hshin.R": {"rx": -50}})
        return rider_hands(sk, P, stance)
    return Cycle(duration, pose, ground_speed=ground_speed)


def beast_attack(duration=0.8):
    """A mount without a rider (a war beast): rears a little and slams down with the head and forelegs."""
    def pose(t, sk):
        u = t / duration
        up = math.sin(math.pi * min(u / 0.45, 1.0)) if u < 0.45 else 0.0
        slam = math.sin(math.pi * min(max((u - 0.4) / 0.4, 0.0), 1.0))
        return _pose(body={"body": {"rx": -18 * up + 10 * slam, "dz": 0.08 * up},
                           "neck_h": {"rx": 25 * slam - 10 * up}, "head_h": {"rx": 20 * slam}},
                     legs={"fleg.L": {"rx": -50 * up + 20 * slam}, "fshin.L": {"rx": 60 * up},
                           "fleg.R": {"rx": -40 * up + 15 * slam}, "fshin.R": {"rx": 50 * up}})
    return Motion(duration, pose)


def attack_sword(duration=0.8):
    """A cut from the saddle: the sword up and back, then down and forward at the right side."""
    windup = dict(SWORD, weapon_wrist=(-0.30, 0.05, 1.62), weapon_elbow=(1.0, 0.3, 0.5), weapon_aim=(-0.3, 0.6, 1.0))
    strike = dict(SWORD, weapon_wrist=(-0.42, -0.35, 1.05), weapon_elbow=(0.3, 1.0, 0.0), weapon_aim=(-0.4, -1.0, -0.6))
    return _action(duration, [(0.0, SWORD, {}), (0.32, windup, {"chest": {"rz": 25}}),
                              (0.47, strike, {"chest": {"rz": -20, "rx": 10}}), (0.8, SWORD, {})])


def attack_lance(duration=0.7):
    """A lance thrust from the saddle: couched, driven forward, back."""
    thrust = dict(LANCE_COUCHED, weapon_wrist=(-0.16, -0.40, 1.22))
    return _action(duration, [(0.0, LANCE, {}), (0.25, LANCE_COUCHED, {"chest": {"rz": 10}}),
                              (0.38, thrust, {"chest": {"rx": 12}}), (0.7, LANCE, {})])


def brace_lance(duration=0.6):
    """Wedge: the lance comes down level and the rider leans in (one-shot; the game holds the end)."""
    return _action(duration, [(0.0, LANCE, {}), (0.6, LANCE_COUCHED, {"spine": {"rx": 10}, "chest": {"rx": 5}})])


def _action(duration, keys):
    ease = cubic_bezier(0.42, 0, 0.58, 1)
    T = keys[-1][0]

    def body_mix(a, b, u):
        out = {}
        for bone in set(a) | set(b):
            out[bone] = {k: a.get(bone, {}).get(k, 0) + (b.get(bone, {}).get(k, 0) - a.get(bone, {}).get(k, 0)) * u
                         for k in set(a.get(bone, {})) | set(b.get(bone, {}))}
        return out

    def pose(t, sk):
        t = t / duration * T
        for k in range(len(keys) - 1):
            (t0, a, ba), (t1, b, bb) = keys[k], keys[k + 1]
            if t <= t1 or k == len(keys) - 2:
                u = ease(min(max((t - t0) / max(t1 - t0, 1e-6), 0.0), 1.0))
                return rider_hands(sk, _pose(rider=body_mix(ba, bb, u)), mix(a, b, u))
    return Motion(duration, pose)


def die(duration=1.4, stance=SWORD):
    """The horse's legs give way and it rolls onto its side, the rider with it."""
    def pose(t, sk):
        u = cubic_bezier(0.5, 0, 0.8, 1)(min(t / duration, 1.0))
        P = _pose(body={"body": {"dz": -0.75 * u, "rx": 8 * math.sin(math.pi * u), "ry": 80 * u},
                        "neck_h": {"rx": 25 * u}, "head_h": {"rx": 20 * u}},
                  legs={"fleg.L": {"rx": -30 * u}, "fshin.L": {"rx": 60 * u}, "fleg.R": {"rx": -20 * u},
                        "fshin.R": {"rx": 50 * u}, "hleg.L": {"rx": 25 * u}, "hshin.L": {"rx": -40 * u},
                        "hleg.R": {"rx": 20 * u}, "hshin.R": {"rx": -30 * u}},
                  rider={"spine": {"rx": -20 * u}, "head": {"rx": -20 * u}})
        return rider_hands(sk, P, stance)
    return Motion(duration, pose)
