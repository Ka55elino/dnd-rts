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


def guard_pose(sk, g=GUARD, breath=0.0, shift=0.0, tip=(0.0, 0.0), legs=True):
    """A stance. breath: -1..1 (chest), shift: -1..1 (weight from the back to the front foot),
    tip: small offset of the weapon point (x, z). legs=False: the caller places the legs (walking)."""
    turn, lean = g["turn"], g["lean"]
    pose = {
        "hips": {"dz": -g["drop"], "dx": 0.025 * shift, "dy": -0.02 * shift, "rz": turn * 0.6, "rx": lean * 0.3},
        "spine": {"rz": turn * 0.2, "rx": lean * 0.35 + 0.6 * breath},
        "chest": {"rz": turn * 0.2, "rx": lean * 0.35 - 1.2 * breath},
        "neck": {"rz": -turn * 0.3, "rx": -lean * 0.3},
        "head": {"rz": -turn * 0.7, "rx": -lean * 0.4 + 0.5 * breath},
    }
    if legs:
        # ankles stay planted whatever the hips do
        ankle_z = sk["foot.L"].head[2]
        plant_foot(sk, pose, "L", (*g["foot_l"], ankle_z + g.get("lift_l", 0.0)), g["toe_l"])
        plant_foot(sk, pose, "R", (*g["foot_r"], ankle_z + g.get("lift_r", 0.0)), g["toe_r"])
    place_hands(sk, pose, g, breath, tip)
    return pose


def place_hands(sk, pose, g, breath=0.0, tip=(0.0, 0.0)):
    """Both hands from the stance: left = the shield / bow / free hand (shield_*), right = the weapon
    hand (sword_*: the weapon's business end — blade, spear tip, staff head — points along sword_aim).
    The targets ride with the chest."""
    lift = np.array([0, 0, 0.012 * breath])
    w = point_on(sk, pose, "chest", np.array(g["shield_wrist"]) + lift)
    place_hand(sk, pose, "L", w, g["shield_elbow"], hand_rot=rot_z(g["shield_yaw"]) @ rot_x(-g.get("shield_tilt", 0.0)))
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


def idle(duration=4.0, stance=None, tip=0.03):
    """A stance with life: breathing (2 breaths per cycle), a slow weight shift between the feet,
    a small circle of the weapon point."""
    stance = stance or RELAXED

    def pose(p, sk):
        a = TAU * p
        return guard_pose(sk, stance, breath=math.sin(2 * a), shift=math.sin(a),
                          tip=(tip * math.cos(a), tip * math.sin(2 * a)))
    return Cycle(duration, pose)


def brace(duration=0.7, ease=(0.25, 0.1, 0.25, 1), rest=None, target=None, step=0.07):
    """From the rest stance into a held stance (one-shot; the game holds the last pose): shield wall,
    spear wall. Everything moves together on one easing curve (CSS-like cubic-bezier); the front foot
    steps (lifted mid-step), the shield rises a little higher on the way."""
    rest, target = rest or RELAXED, target or BRACE
    curve = cubic_bezier(*ease)

    def pose(t, sk):
        u = curve(t / duration)
        g = mix_stance(rest, target, u)
        bump = math.sin(math.pi * min(max(u, 0.0), 1.0))
        g["lift_l"] = step * bump
        sw = g["shield_wrist"]
        g["shield_wrist"] = (sw[0], sw[1], sw[2] + 0.05 * bump)
        return guard_pose(sk, g, breath=0.8 * bump)
    return Motion(duration, pose)


def unbrace(duration=0.7, **kw):
    return Reverse(brace(duration, **kw))


def stance_action(duration, keys, eases=None):
    """A one-shot clip through stances: keys = [(time, stance), ...] (times in seconds, the last one =
    the clip length unless `duration` differs: they are scaled), eases = one cubic-bezier per span."""
    T = keys[-1][0]
    eases = [cubic_bezier(*e) for e in (eases or [(0.42, 0, 0.58, 1)] * (len(keys) - 1))]

    def pose(t, sk):
        t = t / duration * T
        for k in range(len(keys) - 1):
            (t0, a), (t1, b) = keys[k], keys[k + 1]
            if t <= t1 or k == len(keys) - 2:
                u = eases[k](min(max((t - t0) / max(t1 - t0, 1e-6), 0.0), 1.0))
                return guard_pose(sk, mix_stance(a, b, u))
    return Motion(duration, pose)


def locomotion(duration, stride, lift, swing, bob, lean, carry=None):
    """Walk / run: feet follow a formula path and are placed with IK, so the ground speed is exact:
    a planted foot slides back by `stride` during half a cycle -> speed = 2 * stride / duration.
    carry: a stance whose hands hold the weapon while moving (a spear upright, a staff, a bow); without
    it the arms swing."""
    def foot(p):
        if p < 0.5:                       # stance: planted, slides from front to back
            return (stride / 2 - stride * (p / 0.5), 0.0)
        q = (p - 0.5) / 0.5               # swing: forward with a lift
        s = q * q * (3 - 2 * q)
        return (-stride / 2 + stride * s, lift * math.sin(math.pi * q))

    def pose(p, sk):
        drop = bob * (0.5 + 0.5 * math.cos(2 * TAU * p))        # lowest at the middle of each step
        legs = merge(leg_ik(sk, "L", foot(p), drop), leg_ik(sk, "R", foot((p + 0.5) % 1), drop))
        body = {
            "hips": {"dz": -drop, "rz": 6 * math.sin(TAU * p)},
            "spine": {"rx": lean * 0.5, "rz": -4 * math.sin(TAU * p)},
            "chest": {"rx": lean * 0.5},
        }
        if carry is None:
            arm = swing * math.cos(TAU * p)
            return merge(legs, body, {
                "upperarm.L": {"rx": arm}, "forearm.L": {"rx": -15 - 10 * max(0, -math.cos(TAU * p))},
                "upperarm.R": {"rx": -arm}, "forearm.R": {"rx": -15 - 10 * max(0, math.cos(TAU * p))},
            })
        pose = merge(legs, body)
        sway = 0.02 * math.sin(TAU * p)
        g = dict(carry, shield_wrist=_add(carry["shield_wrist"], (0, sway, 0)), sword_wrist=_add(carry["sword_wrist"], (0, -sway, 0)))
        return place_hands(sk, pose, g)
    c = Cycle(duration, pose, ground_speed=2 * stride / duration)
    return c


def _add(a, b):
    return tuple(x + y for x, y in zip(a, b))


def walk(duration=1.0, stride=0.62, lift=0.09, swing=22, bob=0.03, lean=3, carry=None):
    return locomotion(duration, stride, lift, swing, bob, lean, carry)


def run(duration=0.66, stride=0.95, lift=0.18, swing=40, bob=0.05, lean=12, carry=None):
    return locomotion(duration, stride, lift, swing, bob, lean, carry)


# --- actions ------------------------------------------------------------------------------------
def attack_sword(duration=0.8, rest=None):
    """Overhead cut (sword, mace, hammer): up and back over the shoulder while the body winds up, a cut
    forward and down with a short drop and lean, back to the rest stance."""
    rest = rest or RELAXED
    windup = dict(rest, turn=-35, lean=0,
                  sword_wrist=(-0.28, 0.02, 1.62), sword_elbow=(1.0, 0.3, 0.5), sword_aim=(-0.2, 0.5, 1.0))
    strike = dict(rest, turn=-5, lean=10, drop=0.07,
                  sword_wrist=(-0.10, -0.40, 1.10), sword_elbow=(0.6, 0.8, -0.6), sword_aim=(0.35, -1.0, -0.45))
    if "shield_yaw" in rest and rest is RELAXED:
        windup.update(shield_wrist=(0.22, -0.24, 1.12), shield_yaw=12)
        strike.update(shield_wrist=(0.22, -0.26, 1.08), shield_yaw=12)
    return stance_action(duration, [(0.0, rest), (0.32, windup), (0.47, strike), (0.80, rest)],
                         [(0.42, 0, 0.58, 1), (0.55, 0, 1, 0.45), (0.42, 0, 0.58, 1)])


# --- spear and shield ----------------------------------------------------------------------------
# The spear's tip is its business end (sword_aim points it). Relaxed: upright at the right side.
SPEAR_RELAXED = dict(RELAXED, sword_wrist=(-0.24, -0.10, 1.12), sword_elbow=(0.4, 1.0, -0.5), sword_aim=(0.0, -0.08, 1.0))
# spear wall: the spear level over the shield, the butt under the arm, the tip at the enemy's chest
SPEAR_BRACE = dict(BRACE, sword_wrist=(-0.20, -0.18, 1.30), sword_elbow=(0.8, 0.6, -0.2), sword_aim=(0.15, -1.0, 0.05))
SPEAR_CARRY = dict(SPEAR_RELAXED, turn=0)


def attack_spear(duration=0.7):
    """Thrust: draw back, lunge forward with the spear level, return."""
    back = dict(SPEAR_RELAXED, turn=-25, lean=0, sword_wrist=(-0.26, 0.10, 1.25), sword_elbow=(0.6, 0.8, -0.2),
                sword_aim=(0.05, -1.0, 0.1))
    thrust = dict(SPEAR_RELAXED, turn=-5, lean=12, drop=0.08, sword_wrist=(-0.12, -0.45, 1.25),
                  sword_elbow=(0.5, 0.8, -0.4), sword_aim=(0.1, -1.0, 0.0))
    return stance_action(duration, [(0.0, SPEAR_RELAXED), (0.28, back), (0.40, thrust), (0.70, SPEAR_RELAXED)],
                         [(0.42, 0, 0.58, 1), (0.55, 0, 1, 0.45), (0.42, 0, 0.58, 1)])


# --- bow ------------------------------------------------------------------------------------------
# The bow is in the left hand (shield_* = the bow hand), the right hand draws the string.
BOW_RELAXED = dict(RELAXED, turn=-10, lean=2,
                   shield_wrist=(0.27, -0.18, 1.02), shield_elbow=(1.0, 0.3, -1.0), shield_yaw=0, shield_tilt=0,
                   sword_wrist=(-0.24, -0.06, 1.00), sword_elbow=(0.4, 1.0, -0.6), sword_aim=(0.0, -0.3, -1.0))
BOW_CARRY = dict(BOW_RELAXED, turn=0)
BOW_AIM = dict(BOW_RELAXED, turn=-70, lean=4, foot_l=(0.18, -0.10), foot_r=(-0.14, 0.14), toe_l=20, toe_r=-60,
               shield_wrist=(0.10, -0.62, 1.44), shield_elbow=(1.0, 0.2, -0.3), shield_yaw=0, shield_tilt=0,
               sword_wrist=(-0.06, -0.06, 1.44), sword_elbow=(-0.2, 1.0, 0.3), sword_aim=(0.0, -1.0, 0.0))


def attack_bow(duration=1.0):
    """Nock and draw (the bow arm up and out, the string hand to the chin), hold, release, lower."""
    nock = dict(BOW_AIM, sword_wrist=(0.0, -0.40, 1.40))
    released = dict(BOW_AIM, sword_wrist=(-0.12, 0.05, 1.46))
    return stance_action(duration, [(0.0, BOW_RELAXED), (0.25, nock), (0.55, BOW_AIM), (0.68, BOW_AIM),
                                    (0.74, released), (1.0, BOW_RELAXED)])


# --- staff (mages, healers) -----------------------------------------------------------------------
# The staff's head is its business end: upright at the right side; the left hand free.
STAFF_RELAXED = dict(RELAXED, turn=-8, lean=1,
                     shield_wrist=(0.26, -0.06, 1.00), shield_elbow=(1.0, 0.4, -1.0), shield_yaw=0, shield_tilt=0,
                     sword_wrist=(-0.24, -0.14, 1.10), sword_elbow=(0.4, 1.0, -0.5), sword_aim=(0.0, -0.1, 1.0))
STAFF_CARRY = dict(STAFF_RELAXED, turn=0)


def cast_staff(duration=0.9):
    """Cast: the staff raised and thrust toward the target, the free hand pushes forward."""
    gather = dict(STAFF_RELAXED, turn=-20, lean=-2, sword_wrist=(-0.24, 0.0, 1.30), sword_aim=(0.0, 0.2, 1.0),
                  shield_wrist=(0.18, -0.20, 1.30))
    release = dict(STAFF_RELAXED, turn=-5, lean=8, drop=0.05, sword_wrist=(-0.18, -0.40, 1.40), sword_elbow=(0.6, 0.6, -0.5),
                   sword_aim=(0.0, -1.0, 0.6), shield_wrist=(0.12, -0.55, 1.35), shield_elbow=(1.0, 0.0, -0.6))
    return stance_action(duration, [(0.0, STAFF_RELAXED), (0.35, gather), (0.50, release), (0.90, STAFF_RELAXED)],
                         [(0.42, 0, 0.58, 1), (0.55, 0, 1, 0.45), (0.42, 0, 0.58, 1)])


# --- one weapon, free left hand (builder's hammer) ------------------------------------------------
TOOL_RELAXED = dict(RELAXED, turn=-8, lean=2,
                    shield_wrist=(0.26, -0.06, 1.00), shield_elbow=(1.0, 0.4, -1.0), shield_yaw=0, shield_tilt=0)


# --- guns and crossbows (two hands forward) --------------------------------------------------------
GUN_RELAXED = dict(STAFF_RELAXED, turn=-10,
                   sword_wrist=(-0.20, -0.20, 1.05), sword_elbow=(0.5, 1.0, -0.6), sword_aim=(0.1, -1.0, -0.6),
                   shield_wrist=(0.10, -0.40, 1.02), shield_elbow=(1.0, 0.4, -0.8), shield_yaw=-20)
GUN_CARRY = dict(GUN_RELAXED, turn=0, sword_aim=(0.0, -0.2, 1.0), sword_wrist=(-0.24, -0.12, 1.10))
GUN_AIM = dict(GUN_RELAXED, turn=-30, lean=3,
               sword_wrist=(-0.16, -0.18, 1.42), sword_elbow=(0.2, 1.0, -0.4), sword_aim=(0.0, -1.0, 0.03),
               shield_wrist=(0.04, -0.52, 1.40), shield_elbow=(1.0, 0.3, -0.6), shield_yaw=0)


def attack_gun(duration=1.0):
    """Raise and aim, fire (a short kick back), lower."""
    kick = dict(GUN_AIM, sword_wrist=(-0.16, -0.10, 1.46), sword_aim=(0.0, -1.0, 0.25))
    return stance_action(duration, [(0.0, GUN_RELAXED), (0.35, GUN_AIM), (0.55, GUN_AIM), (0.62, kick), (1.0, GUN_RELAXED)])


# --- empty hands (elementals) ----------------------------------------------------------------------
FIST_RELAXED = dict(STAFF_RELAXED, turn=-5, sword_wrist=(-0.30, -0.10, 1.02), sword_elbow=(0.6, 1.0, -0.5),
                    sword_aim=(-0.2, -0.3, -1.0), shield_wrist=(0.30, -0.10, 1.02), shield_elbow=(1.0, 0.6, -0.5))


# --- flying humanoid (harpy: wings along the arms) --------------------------------------------------
def fly(duration=0.7, flap=40, speed=None):
    """Hovering / flying: the arms (wings) beat up and down, the legs hang tucked, the body bobs."""
    def pose(p, sk):
        b = math.sin(TAU * p)
        return {
            "hips": {"dz": 0.05 * b, "rx": 15},
            "spine": {"rx": 5}, "chest": {"rx": -3 * b},
            "upperarm.L": {"ry": -55 - flap * b}, "upperarm.R": {"ry": 55 + flap * b},
            "forearm.L": {"ry": -10 * b}, "forearm.R": {"ry": 10 * b},
            "thigh.L": {"rx": -35}, "thigh.R": {"rx": -30}, "shin.L": {"rx": 70}, "shin.R": {"rx": 65},
            "foot.L": {"rx": 40}, "foot.R": {"rx": 40},
        }
    return Cycle(duration, pose, ground_speed=speed)


def dive_attack(duration=0.8):
    """Claw attack from the air: the wings sweep up, the legs kick forward, back to hovering."""
    def pose(t, sk):
        u = t / duration
        k = math.sin(math.pi * min(max((u - 0.2) / 0.5, 0.0), 1.0))
        return {
            "hips": {"rx": 15 - 40 * k, "dz": -0.10 * k},
            "upperarm.L": {"ry": -100 * k - 40 * (1 - k)}, "upperarm.R": {"ry": 100 * k + 40 * (1 - k)},
            "thigh.L": {"rx": -35 - 60 * k}, "thigh.R": {"rx": -30 - 55 * k},
            "shin.L": {"rx": 70 - 50 * k}, "shin.R": {"rx": 65 - 45 * k},
        }
    return Motion(duration, pose)


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
