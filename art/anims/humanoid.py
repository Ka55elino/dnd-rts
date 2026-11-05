"""Humanoid clips as formulas (cycles) and key poses (actions). Shared by every humanoid model;
parameters change the character of the motion (a heavy soldier vs a light elf).

Pose conventions (see tools/modelc/modelc/anim.py): degrees around the bone's head, in the parent's
frame; for a hanging limb rx > 0 = backward, rx < 0 = forward. Phase p = 0..1 over a cycle.
"""
import math
import numpy as np
from modelc import Cycle, Action, Motion, Reverse, leg_ik, merge, aim, point_on, plant_foot, place_hand
from modelc.math3d import rot_z, rot_x, cubic_bezier

TAU = 2 * math.pi


# --- combat stance (sword and shield) ------------------------------------------------------------
# Words -> numbers. Authoring axes: Z up, the enemy is in front (-Y), the character's left is +X.
GUARD = dict(
    # legs: feet a bit wider than the shoulders, the shield-side (left) foot in front
    foot_l=(0.17, -0.10), foot_r=(-0.16, 0.14),   # ankle (x, y) on the ground, m
    toe_l=12, toe_r=-40,                         # toe turned out, degrees
    drop=0.07,                                   # knees bent: the hips go down, m
    # body: turned side-on (right shoulder back), leaning a little forward; the head looks at the enemy
    turn=-25, lean=5,
    # shield arm: shield up in front of the body, covering chin to hip (wrist, rest-space point riding with the chest);
    # the elbow goes out and forward so the forearm lies along the back of the shield
    shield_wrist=(0.17, -0.30, 1.16), shield_elbow=(1.0, -0.3, -0.6), shield_yaw=-8,
    shield_tilt=10,                              # the shield top leans back toward the body, degrees
    # sword arm (variant B): hand at the waist, the point aimed at the enemy past the shield
    sword_wrist=(-0.22, -0.20, 1.04), sword_elbow=(0.6, 0.8, -0.6),
    sword_aim=(-0.10, -1.0, 0.28),               # blade direction in the world
)


def guard_pose(sk, g=GUARD, breath=0.0, shift=0.0, tip=(0.0, 0.0)):
    """The combat stance. breath: -1..1 (chest), shift: -1..1 (weight from the back to the front foot),
    tip: small offset of the sword point (x, z)."""
    turn, lean = g["turn"], g["lean"]
    pose = {
        "hips": {"dz": -g["drop"], "dx": 0.025 * shift, "dy": -0.02 * shift, "rz": turn * 0.6, "rx": lean * 0.3},
        "spine": {"rz": turn * 0.2, "rx": lean * 0.35 + 0.6 * breath},
        "chest": {"rz": turn * 0.2, "rx": lean * 0.35 - 1.2 * breath},
        "neck": {"rz": -turn * 0.3, "rx": -lean * 0.3},
        "head": {"rz": -turn * 0.7, "rx": -lean * 0.4 + 0.5 * breath},
    }
    # legs: ankles stay planted whatever the hips do
    ankle_z = sk["foot.L"].head[2]
    plant_foot(sk, pose, "L", (*g["foot_l"], ankle_z + g.get("lift_l", 0.0)), g["toe_l"])
    plant_foot(sk, pose, "R", (*g["foot_r"], ankle_z + g.get("lift_r", 0.0)), g["toe_r"])
    # shield: the hand rides with the chest; the shield (bound to hand.L) is upright, facing the enemy
    lift = np.array([0, 0, 0.012 * breath])
    w = point_on(sk, pose, "chest", np.array(g["shield_wrist"]) + lift)
    place_hand(sk, pose, "L", w, g["shield_elbow"], hand_rot=rot_z(g["shield_yaw"]) @ rot_x(-g.get("shield_tilt", 0.0)))
    # sword: the blade (rest: down from the hand, flat facing front) aimed forward, edge up
    w = point_on(sk, pose, "chest", np.array(g["sword_wrist"]) + lift)
    d = np.array(g["sword_aim"]) + np.array([tip[0], 0, tip[1]])
    place_hand(sk, pose, "R", w, g["sword_elbow"], hand_rot=aim((0, 0, -1), d, (1, 0, 0), (0, 0, 1)))
    return pose


# Relaxed stance (idle): the same guard, loosened — upright, knees almost straight, the body turned
# less; the shield lowered to the side and turned outward, the sword low with the point forward-down.
RELAXED = dict(GUARD,
    foot_l=(0.15, -0.06), foot_r=(-0.15, 0.10), toe_l=10, toe_r=-30,
    drop=0.035, turn=-15, lean=2,
    shield_wrist=(0.31, -0.34, 1.04), shield_elbow=(1.0, -0.1, -0.9), shield_yaw=40,
    sword_wrist=(-0.24, -0.13, 1.00), sword_elbow=(0.5, 1.0, -0.6),
    sword_aim=(-0.05, -1.0, -0.40),
)

# Brace (the "raise shields" ability): the guard with the sword raised over the shield — the hand at
# the shoulder, the elbow out to the side, the blade level and aimed at the enemy over the shield's top;
# the front (shield-side) foot steps forward, the body bent forward behind the shield held high.
BRACE = dict(GUARD,
    foot_l=(0.19, -0.36), toe_l=14, foot_r=(-0.16, 0.12), toe_r=-45, drop=0.11,
    lean=32,
    shield_wrist=(0.47, -1.0, 1.10),
    sword_wrist=(-0.26, -0.10, 1.62), sword_elbow=(1.0, 0.35, 0.5),
    sword_aim=(0.30, -1.0, 0.06),
)


def mix_stance(a, b, u):
    """Stance parameters between a (u = 0) and b (u = 1); numbers and tuples are interpolated."""
    out = {}
    for k in set(a) | set(b):
        x, y = a.get(k, 0.0), b.get(k, 0.0)
        out[k] = tuple(p + (q - p) * u for p, q in zip(x, y)) if isinstance(x, tuple) else x + (y - x) * u
    return out


def idle(duration=4.0):
    """Relaxed stance with life: breathing (2 breaths per cycle), a slow weight shift between the feet,
    a small circle of the sword point."""
    def pose(p, sk):
        a = TAU * p
        return guard_pose(sk, RELAXED, breath=math.sin(2 * a), shift=math.sin(a),
                          tip=(0.03 * math.cos(a), 0.03 * math.sin(2 * a)))
    return Cycle(duration, pose)


def brace(duration=0.7, ease=(0.25, 0.1, 0.25, 1)):
    """From the relaxed stance into the brace (one-shot; the game holds the last pose). Everything moves
    together on one easing curve (CSS-like cubic-bezier): the shield comes up (a little higher than its
    end position on the way), the body drops and turns behind it, the front foot steps forward (lifted
    off the ground in the middle of the step), the sword rises over the shield."""
    curve = cubic_bezier(*ease)

    def pose(t, sk):
        s = t / duration
        u = curve(s)
        g = mix_stance(RELAXED, BRACE, u)
        bump = math.sin(math.pi * min(max(u, 0.0), 1.0))
        g["lift_l"] = 0.07 * bump
        sw = g["shield_wrist"]
        g["shield_wrist"] = (sw[0], sw[1], sw[2] + 0.05 * bump)
        return guard_pose(sk, g, breath=0.8 * bump)
    return Motion(duration, pose)


def unbrace(duration=0.7):
    return Reverse(brace(duration))


def locomotion(duration, stride, lift, swing, bob, lean):
    """Walk / run: feet follow a formula path and are placed with IK, so the ground speed is exact:
    a planted foot slides back by `stride` during half a cycle -> speed = 2 * stride / duration."""
    def foot(p):
        if p < 0.5:                       # stance: planted, slides from front to back
            return (stride / 2 - stride * (p / 0.5), 0.0)
        q = (p - 0.5) / 0.5               # swing: forward with a lift
        s = q * q * (3 - 2 * q)
        return (-stride / 2 + stride * s, lift * math.sin(math.pi * q))

    def pose(p, sk):
        drop = bob * (0.5 + 0.5 * math.cos(2 * TAU * p))        # lowest at the middle of each step
        legs = merge(leg_ik(sk, "L", foot(p), drop), leg_ik(sk, "R", foot((p + 0.5) % 1), drop))
        arm = swing * math.cos(TAU * p)
        return merge(legs, {
            "hips": {"dz": -drop, "rz": 6 * math.sin(TAU * p)},
            "spine": {"rx": lean * 0.5, "rz": -4 * math.sin(TAU * p)},
            "chest": {"rx": lean * 0.5},
            "upperarm.L": {"rx": arm}, "forearm.L": {"rx": -15 - 10 * max(0, -math.cos(TAU * p))},
            "upperarm.R": {"rx": -arm}, "forearm.R": {"rx": -15 - 10 * max(0, math.cos(TAU * p))},
        })
    c = Cycle(duration, pose, ground_speed=2 * stride / duration)
    return c


def walk(duration=1.0, stride=0.62, lift=0.09, swing=22, bob=0.03, lean=3):
    return locomotion(duration, stride, lift, swing, bob, lean)


def run(duration=0.66, stride=0.95, lift=0.18, swing=40, bob=0.05, lean=12):
    return locomotion(duration, stride, lift, swing, bob, lean)


# --- actions ------------------------------------------------------------------------------------
READY = {"upperarm.R": {"rx": -25, "rz": 10}, "forearm.R": {"rx": -45},
         "upperarm.L": {"rx": -30, "rz": -15}, "forearm.L": {"rx": -50}}


def attack_sword(duration=0.8):
    windup = {"upperarm.R": {"rx": -150, "rz": 25}, "forearm.R": {"rx": -70},
              "chest": {"rz": 25}, "spine": {"rz": 10}, "upperarm.L": {"rx": -45, "rz": -20}, "forearm.L": {"rx": -60}}
    strike = {"upperarm.R": {"rx": -35, "rz": -35}, "forearm.R": {"rx": -10},
              "chest": {"rz": -30, "rx": 10}, "spine": {"rz": -10}, "upperarm.L": {"rx": -25, "rz": -10}, "forearm.L": {"rx": -40},
              "thigh.L": {"rx": -15}, "shin.L": {"rx": 15}}
    return Action(duration, [
        (0.00, READY, "linear"),
        (0.32, windup, "in_out"),
        (0.47, strike, "in"),
        (0.80, READY, "in_out"),
    ])


def die(duration=1.3):
    hit = {"chest": {"rx": -20}, "spine": {"rx": -10}, "head": {"rx": -25},
           "upperarm.L": {"rx": -40, "rz": 20}, "upperarm.R": {"rx": -40, "rz": -20}}
    knees = {"hips": {"dz": -0.35, "rx": 10}, "thigh.L": {"rx": -70}, "shin.L": {"rx": 100}, "thigh.R": {"rx": -60},
             "shin.R": {"rx": 110}, "chest": {"rx": 15}, "head": {"rx": 10}}
    down = {"hips": {"dz": -0.80, "rx": 88, "dy": -0.35}, "thigh.L": {"rx": -5}, "shin.L": {"rx": 10},
            "thigh.R": {"rx": -10}, "shin.R": {"rx": 15}, "upperarm.L": {"rx": -60, "rz": 30},
            "upperarm.R": {"rx": -80, "rz": -20}, "head": {"ry": 40}}
    return Action(duration, [
        (0.00, {}, "linear"),
        (0.20, hit, "out"),
        (0.65, knees, "in"),
        (1.10, down, "in"),
        (1.30, down, "linear"),
    ])
