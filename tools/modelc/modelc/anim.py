"""Animations as formulas and key poses.

A pose is {bone: {"rx": deg, "ry": deg, "rz": deg, "dx": m, "dy": m, "dz": m}}, rotations around
the bone's head in its parent's frame (authoring axes: Z up, front -Y).
For a limb hanging down, rx > 0 swings it backward, rx < 0 forward.
"""
import math
import numpy as np
from .math3d import euler, mat_to_quat, quat_to_mat, slerp, EASE, ease_fn


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


class Motion(Clip):
    """A one-shot clip from a function: fn(t, sk) -> pose, t = 0..duration seconds (e.g. a stance
    transition that interpolates IK targets instead of bone angles)."""
    def __init__(self, duration, fn):
        super().__init__(duration, False)
        self.fn = fn

    def pose(self, t, sk):
        return self.fn(min(max(t, 0.0), self.duration), sk)


class Reverse(Clip):
    """Another one-shot clip played backwards (unbrace = brace reversed)."""
    def __init__(self, clip):
        super().__init__(clip.duration, False)
        self.clip = clip

    def pose(self, t, sk):
        return self.clip.pose(self.duration - t, sk)


class Action(Clip):
    """keys: [(time s, pose, ease)]; between keys every bone is interpolated (slerp) with the
    ease of the later key: a name ("in_out") or CSS-like cubic-bezier points (x1, y1, x2, y2). Bones missing from a key are at rest in that key."""
    def __init__(self, duration, keys):
        super().__init__(duration, False)
        self.keys = sorted(keys, key=lambda k: k[0])

    def pose(self, t, sk):
        ks = self.keys
        if t <= ks[0][0]:
            return ks[0][1]
        for (t0, p0, _), (t1, p1, ease) in zip(ks, ks[1:]):
            if t <= t1:
                u = ease_fn(ease)((t - t0) / max(t1 - t0, 1e-9))
                return blend(p0, p1, u)
        return ks[-1][1]


def _rot(v):
    return v["R"] if "R" in v else euler(v.get("rx", 0), v.get("ry", 0), v.get("rz", 0))


def blend(p0, p1, u):
    out = {}
    for b in set(p0) | set(p1):
        a, c = p0.get(b, {}), p1.get(b, {})
        q = slerp(mat_to_quat(_rot(a)), mat_to_quat(_rot(c)), u)
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
        R = _rot(v)
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


# --- world-space IK ---------------------------------------------------------------------------------
# Poses can also be built from targets in the world (authoring axes: Z up, front -Y, left +X):
# "the left ankle here, the toe turned 10° out", "the sword hand at the waist, the blade aimed at the
# enemy". The solvers write rotation matrices {"R": 3x3} into the pose for the bones they place;
# set the torso first, since the limbs are solved from where the torso puts their roots.

def _unit(v):
    v = np.asarray(v, dtype=float)
    return v / max(np.linalg.norm(v), 1e-9)


def _basis(d, n):
    """Orthonormal frame [d, n', d x n'] (n made perpendicular to d)."""
    d = _unit(d)
    n = _unit(np.asarray(n, dtype=float) - d * (np.dot(n, d)))
    return np.column_stack([d, n, np.cross(d, n)])


def aim(axis, direction, up_axis, up):
    """World rotation that turns the bone's rest `axis` to `direction` and its rest `up_axis` toward `up`."""
    return _basis(direction, up) @ _basis(axis, up_axis).T


def world_of(sk, pose):
    return sk.world(to_matrices(pose))


def point_on(sk, pose, bone, p):
    """Where the rest-space point p ends up when it rides with `bone` (e.g. a hand target in front of
    the chest that follows the chest's breathing and the body's turn)."""
    W = world_of(sk, pose)[bone]
    return (W @ np.append(np.asarray(p, dtype=float) - sk[bone].head, 1.0))[:3]


def limb_ik(sk, pose, root, mid, end, target, pole, rest_pole, end_rot=None):
    """Two-bone IK in world space: rotate `root` and `mid` so the head of `end` reaches `target`,
    the middle joint bending toward `pole` (a world direction: forward for a knee, back/out for an
    elbow). rest_pole: the same direction in the rest pose (defines the twist-free bend plane).
    end_rot: optional world rotation of `end` (a level foot, a hand holding a weapon a certain way).
    Changes and returns `pose`."""
    W = world_of(sk, pose)
    parent = sk[root].parent
    Pw = W[parent][:3, :3] if parent else np.eye(3)
    H = W[root][:3, 3]
    r0, m0, e0 = sk[root].head, sk[mid].head, sk[end].head
    L1, L2 = np.linalg.norm(m0 - r0), np.linalg.norm(e0 - m0)
    D = np.asarray(target, dtype=float) - H
    d = min(max(np.linalg.norm(D), abs(L1 - L2) + 1e-4), L1 + L2 - 1e-4)
    dirD = _unit(D)
    pp = _unit(np.asarray(pole, dtype=float) - dirD * np.dot(pole, dirD))
    a = (L1 * L1 - L2 * L2 + d * d) / (2 * d)
    K = H + dirD * a + pp * np.sqrt(max(L1 * L1 - a * a, 0.0))
    E = H + dirD * d
    n = np.cross(pp, dirD)
    rest_dir = _unit(e0 - r0)
    rp = _unit(np.asarray(rest_pole, dtype=float) - rest_dir * np.dot(rest_pole, rest_dir))
    n0 = np.cross(rp, rest_dir)
    Rr = _basis(K - H, n) @ _basis(m0 - r0, n0).T
    Rm = _basis(E - K, n) @ _basis(e0 - m0, n0).T
    pose.setdefault(root, {})["R"] = Pw.T @ Rr
    pose.setdefault(mid, {})["R"] = Rr.T @ Rm
    if end_rot is not None:
        pose.setdefault(end, {})["R"] = Rm.T @ end_rot
    return pose


def plant_foot(sk, pose, side, ankle, yaw=0.0, knee_out=0.15):
    """Put the ankle of leg `side` ("L"/"R") at the world point `ankle`, the foot flat on the ground and
    turned by `yaw` degrees around Z (+ = toe to the character's left). The knee points over the toe."""
    from .math3d import rot_z
    R = rot_z(yaw)
    toe = R @ np.array([0.0, -1.0, 0.0])
    out = np.array([1.0 if side == "L" else -1.0, 0.0, 0.0])
    return limb_ik(sk, pose, f"thigh.{side}", f"shin.{side}", f"foot.{side}", ankle,
                   pole=toe + knee_out * out, rest_pole=(0, -1, 0), end_rot=R)


def place_hand(sk, pose, side, wrist, elbow=(0, 1, -0.3), hand_rot=None):
    """Put the wrist of arm `side` at the world point `wrist`; the elbow bends toward `elbow` (world
    direction; x is mirrored for the right arm, so (0.5, 0.5, -1) = out, back and down on both sides)."""
    e = np.array(elbow, dtype=float)
    if side == "R":
        e[0] = -e[0]
    return limb_ik(sk, pose, f"upperarm.{side}", f"forearm.{side}", f"hand.{side}", wrist,
                   pole=e, rest_pole=(0, 1, 0), end_rot=hand_rot)
