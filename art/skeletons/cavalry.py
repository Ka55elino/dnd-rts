"""Mount + rider in one skeleton (cavalry: knights, the emperor, wolf riders, griffon riders; a beast
without a rider uses it too). The mount's body is the root; the rider is the humanoid skeleton lifted
onto the saddle with its hips parented to the mount's body, so the humanoid clips' hand IK and the
kit's shapes work for the rider too. Authoring axes as everywhere: Z up, front -Y, left +X; .L bones get
mirrored .R twins. `mount(scale)` makes a smaller / bigger mount (a wolf ~0.62, a horse 1.0)."""
from modelc import Skeleton
from art.skeletons.humanoid import HUMANOID_BONES

HORSE_BONES = [
    # name          parent       head                    tail
    ("body",        None,        (0, 0.0, 1.20),         (0, -0.50, 1.24)),
    ("neck_h",      "body",      (0, -0.48, 1.32),       (0, -0.78, 1.74)),
    ("head_h",      "neck_h",    (0, -0.78, 1.74),       (0, -1.10, 1.52)),
    ("tail",        "body",      (0, 0.56, 1.32),        (0, 0.74, 0.95)),
    ("fleg.L",      "body",      (0.14, -0.42, 1.04),    (0.14, -0.44, 0.58)),
    ("fshin.L",     "fleg.L",    (0.14, -0.44, 0.58),    (0.14, -0.44, 0.06)),
    ("hleg.L",      "body",      (0.14, 0.42, 1.06),     (0.14, 0.48, 0.60)),
    ("hshin.L",     "hleg.L",    (0.14, 0.48, 0.60),     (0.14, 0.44, 0.06)),
    ("wing.L",      "body",      (0.20, -0.20, 1.42),    (0.95, 0.05, 1.55)),   # griffons only
]


def mount(scale=1.0):
    """The skeleton for a mount of the given size; `.rider` = where the humanoid was moved to (the
    offset from the standing humanoid), used by the cavalry clips and `kit.shift`."""
    s = scale
    rider = (0.0, 0.06 * s, 1.53 * s - 0.98)   # hips on the saddle (the body's top)
    sc = lambda p: (p[0] * s, p[1] * s, p[2] * s)
    lift = lambda p: (p[0] + rider[0], p[1] + rider[1], p[2] + rider[2])
    bones = [(n, par, sc(h), sc(t)) for n, par, h, t in HORSE_BONES]
    bones += [(n, "body" if par is None else par, lift(h), lift(t)) for n, par, h, t in HUMANOID_BONES]
    sk = Skeleton(bones)
    sk.rider = rider
    sk.mount_scale = s
    return sk


CAVALRY = mount(1.0)
RIDER = CAVALRY.rider
WOLF = mount(0.62)


def lift(p, sk=CAVALRY):
    r = sk.rider
    return (p[0] + r[0], p[1] + r[1], p[2] + r[2])
TIGER = mount(0.72)
BEAR = mount(0.85)
SPIDER = mount(0.75)
