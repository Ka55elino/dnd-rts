"""Neutral outposts (captured by standing next to them): gold mine, ore mine, sawmill. Footprint 8 × 8."""
from art.props._style import *  # noqa: F401,F403

ROCK = (0.46, 0.43, 0.40)
ROCK_D = (0.36, 0.34, 0.32)


def mine(ore, ore_kind="metal"):
    out = rocks(0, -1.2, r=3.2, n=5, color=ROCK)
    out += box((0, 0, 1.6), (2.6, 2.8, 0.6), color=DARK)                           # the shaft
    for x in (-1.3, 1.3):
        out += rod((x, 0, 2.0), (x * 0.9, 3.0, 2.0), 0.16, seg=4, color=WOOD)
    out += rod((-1.6, 3.0, 2.0), (1.6, 3.0, 2.0), 0.16, seg=4, color=WOOD)
    for x in (-0.45, 0.45):
        out += box((x, 0, 3.2), (0.1, 0.1, 2.6), color=IRON, kind="metal", detail=2)
    out += box((1.9, 0, 3.0), (1.1, 0.8, 1.4), color=WOOD_D)                        # a cart full of ore
    for dx, dz in ((-0.2, -0.3), (0.2, 0.2), (0.0, 0.4)):
        out += sphere((1.9 + dx, 0.95, 3.0 + dz), 0.3, seg=5, rings=2, color=ore, kind=ore_kind, detail=1)
    for x, z in ((-2.6, 2.8), (-3.0, 1.8)):
        out += sphere((x, 0.2, z), 0.45, seg=5, rings=2, color=ore, kind=ore_kind, detail=1)
    out += flag((3.0, 0, 1.0), h=3.6)
    out += box((-2.2, 0, 2.4), (1.2, 0.9, 0.8), color=WOOD_D, detail=1)             # a crate
    return out


def sawmill():
    out = hall(5.0, 3.6, 3.0, WOOD, (0.40, 0.27, 0.16), at=(-0.6, 0, -1.6), roof_h=1.8)
    out += box((-0.6, 0, 0.3), (5.0, 3.0, 0.1), color=DARK)                         # open front
    for x in (-2.9, 1.7):
        out += rod((x, 0, 0.35), (x, 3.0, 0.35), 0.14, seg=4, color=WOOD_D)
    out += box((-0.6, 0, 1.4), (3.0, 1.0, 1.0), color=WOOD_D)                       # the saw bench
    out += rod((-0.6, 1.4, 1.3), (-0.6, 1.4, 1.5), 0.9, seg=12, color=IRON, kind="metal")   # saw blade
    out += rod((3.0, 0.9, -2.6), (3.0, 0.9, -1.6), 1.0, seg=10, color=WOOD)          # water wheel
    out += logs(1.4, 3.0, n=3, length=3.4)
    out += logs(-2.6, 3.0, n=2, length=2.4, color=(0.55, 0.42, 0.26))
    out += flag((3.2, 0, 1.4), h=3.6)
    return out


PROPS = [
    Prop("gold_mine", mine(GOLD), "Gold mine: a timbered shaft in a rocky mound, a cart of gold"),
    Prop("ore_mine", mine((0.30, 0.30, 0.34)), "Ore mine: a timbered shaft in a rocky mound, a cart of iron ore"),
    Prop("sawmill", sawmill(), "Sawmill: an open timber shed with a saw blade, a water wheel, logs"),
]
