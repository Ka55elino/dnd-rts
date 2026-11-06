"""Clips for rigid machines (art/skeletons/machines.py)."""
import math
from modelc import Cycle, Motion

TAU = 2 * math.pi


def airship_hover(duration=2.0, spin=4):
    """Hovering / flying: the hull bobs and rolls a little, the propeller spins (`spin` turns per cycle)."""
    def pose(p, sk):
        return {"hull": {"dz": 0.08 * math.sin(TAU * p), "ry": 2 * math.sin(TAU * p)}, "prop": {"ry": 360 * spin * p}}
    return Cycle(duration, pose)


def airship_fire(duration=0.8):
    def pose(t, sk):
        k = math.sin(math.pi * min(t / 0.3, 1.0)) if t < 0.3 else 0.0
        return {"hull": {"rx": -4 * k, "dy": 0.10 * k}, "prop": {"ry": 360 * 3 * t / duration}}
    return Motion(duration, pose)


def airship_crash(duration=1.6):
    def pose(t, sk):
        u = min(t / duration, 1.0)
        return {"hull": {"dz": -1.0 * u * u, "rx": 25 * u, "ry": 30 * u}}
    return Motion(duration, pose)


def mortar_idle(duration=3.0):
    return Cycle(duration, lambda p, sk: {"barrel": {"rx": 2 * math.sin(TAU * p)}})


def mortar_move(duration=0.8, ground_speed=1.5):
    """Rolled forward: the carriage rocks."""
    return Cycle(duration, lambda p, sk: {"base": {"rx": 2 * math.sin(2 * TAU * p), "dz": 0.01 * math.sin(2 * TAU * p)}},
                 ground_speed=ground_speed)


def mortar_fire(duration=1.0):
    def pose(t, sk):
        u = t / duration
        kick = math.exp(-12 * u) if u > 0.1 else u / 0.1
        return {"barrel": {"rx": -25 * kick}, "base": {"dy": 0.08 * kick}}
    return Motion(duration, pose)


def mortar_wreck(duration=1.2):
    def pose(t, sk):
        u = min(t / duration, 1.0)
        return {"base": {"ry": 35 * u, "dz": -0.15 * u}, "barrel": {"rx": 40 * u}}
    return Motion(duration, pose)


def trap_idle(duration=3.0, glow=0.0):
    """A trap waiting: still, the lid breathes a little (rune traps pulse higher with glow)."""
    return Cycle(duration, lambda p, sk: {"lid": {"dz": (0.004 + glow) * (0.5 + 0.5 * math.sin(TAU * p))}})


def trap_spring(duration=0.6):
    """Going off: the lid jumps up and spins, the base squashes."""
    def pose(t, sk):
        u = min(t / duration, 1.0)
        return {"lid": {"dz": 0.6 * math.sin(math.pi * u), "rz": 360 * u}, "base": {"dz": -0.02 * u}}
    return Motion(duration, pose)
