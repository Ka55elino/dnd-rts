"""Humanoid clips as formulas (cycles) and key poses (actions). Shared by every humanoid model;
parameters change the character of the motion (a heavy soldier vs a light elf).

Pose conventions (see tools/modelc/modelc/anim.py): degrees around the bone's head, in the parent's
frame; for a hanging limb rx > 0 = backward, rx < 0 = forward. Phase p = 0..1 over a cycle.
"""
import math
from modelc import Cycle, Action, leg_ik, merge

TAU = 2 * math.pi


def idle(duration=2.4, breath=1.5):
    def pose(p, sk):
        b = math.sin(TAU * p)
        return {
            "chest": {"rx": -breath * b},
            "spine": {"rx": 0.6 * breath * b},
            "upperarm.L": {"rz": 2 * b}, "upperarm.R": {"rz": -2 * b},
            "hips": {"dz": -0.004 * (1 - math.cos(TAU * p))},
        }
    return Cycle(duration, pose)


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
