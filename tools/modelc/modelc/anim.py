"""Animations as formulas and key poses.

A pose is {bone: {"rx": deg, "ry": deg, "rz": deg, "dx": m, "dy": m, "dz": m}}, rotations around
the bone's head in its parent's frame (authoring axes: Z up, front -Y).
For a limb hanging down, rx > 0 swings it backward, rx < 0 forward.
"""
import math
import numpy as np
from .math3d import euler, mat_to_quat, quat_to_mat, slerp, EASE


class Clip:
    def __init__(self, duration, loop):
        self.duration, self.loop = duration, loop
        self.ground_speed = None  # m/s at 1x for locomotion cycles

    def pose(self, t, sk):
        raise NotImplementedError


class Cycle(Clip):
    """fn(p, sk) -> pose, p = phase 0..1. ground_speed: m/s (locomotion)."""
    def __init__(self, duration, fn, ground_speed=None):
        super().__init__(duration, True)
        self.fn, self.ground_speed = fn, ground_speed

    def pose(self, t, sk):
        return self.fn((t / self.duration) % 1.0, sk)


class Action(Clip):
    """keys: [(time s, pose, ease)]; between keys every bone is interpolated (slerp) with the
    ease of the later key. Bones missing from a key are at rest in that key."""
    def __init__(self, duration, keys):
        super().__init__(duration, False)
        self.keys = sorted(keys, key=lambda k: k[0])

    def pose(self, t, sk):
        ks = self.keys
        if t <= ks[0][0]:
            return ks[0][1]
        for (t0, p0, _), (t1, p1, ease) in zip(ks, ks[1:]):
            if t <= t1:
                u = EASE[ease]((t - t0) / max(t1 - t0, 1e-9))
                return blend(p0, p1, u)
        return ks[-1][1]


def blend(p0, p1, u):
    out = {}
    for b in set(p0) | set(p1):
        a, c = p0.get(b, {}), p1.get(b, {})
        q = slerp(mat_to_quat(euler(a.get("rx", 0), a.get("ry", 0), a.get("rz", 0))),
                  mat_to_quat(euler(c.get("rx", 0), c.get("ry", 0), c.get("rz", 0))), u)
        out[b] = {"R": quat_to_mat(q)}
        for k in ("dx", "dy", "dz"):
            out[b][k] = a.get(k, 0) + (c.get(k, 0) - a.get(k, 0)) * u
    return out


def merge(*poses):
    """Combine poses (later ones add their rotations / offsets on top)."""
    out = {}
    for p in poses:
        for b, v in p.items():
            o = out.setdefault(b, {})
            for k, x in v.items():
                o[k] = o.get(k, 0) + x if k != "R" else x
    return out


def to_matrices(pose):
    out = {}
    for b, v in pose.items():
        R = v["R"] if "R" in v else euler(v.get("rx", 0), v.get("ry", 0), v.get("rz", 0))
        out[b] = (R, np.array([v.get("dx", 0), v.get("dy", 0), v.get("dz", 0)]))
    return out


def leg_ik(sk, side, ankle, hips_drop=0.0):
    """Two-bone IK in the side plane: rotations of thigh, shin (and foot kept level) so the ankle
    reaches `ankle` = (forward m, height m) relative to its rest position. forward > 0 = to the front."""
    thigh, shin, foot = f"thigh.{side}", f"shin.{side}", f"foot.{side}"
    L1 = np.linalg.norm(sk[shin].head - sk[thigh].head)
    L2 = np.linalg.norm(sk[foot].head - sk[shin].head)
    hip = sk[thigh].head.copy(); hip[2] -= hips_drop
    rest_ankle = sk[foot].head
    target = np.array([rest_ankle[0], rest_ankle[1] - ankle[0], rest_ankle[2] + ankle[1]])
    D = target - hip
    dy, dz = D[1], D[2]
    d = min(math.hypot(dy, dz), L1 + L2 - 1e-4)
    a = math.atan2(dy, -dz)                       # > 0: the target is behind the hip
    c = math.acos(max(-1, min(1, (L1 * L1 + d * d - L2 * L2) / (2 * L1 * d))))
    k = math.acos(max(-1, min(1, (L1 * L1 + L2 * L2 - d * d) / (2 * L1 * L2))))
    th = math.degrees(a - c)                       # thigh: knee goes forward
    kn = math.degrees(math.pi - k)                 # shin bends back
    return {thigh: {"rx": th}, shin: {"rx": kn}, foot: {"rx": -(th + kn)}}
