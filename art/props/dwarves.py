"""Dwarves: stone, bronze, runes, machinery, steam. Low, massive, square shapes."""
import math
from art.props._style import *  # noqa: F401,F403

GRANITE = (0.55, 0.52, 0.48)
GRANITE_D = (0.42, 0.40, 0.37)
GRANITE_L = (0.64, 0.61, 0.56)
BRONZE = (0.72, 0.50, 0.25)
RUNE = (0.40, 0.82, 1.0)        # glow
TURF = (0.36, 0.50, 0.26)
STEAM = (0.85, 0.85, 0.85)


def stepped(w, d, h, steps=3, at=(0, 0, 0), color=GRANITE, shrink=0.8):
    """A stepped block (a mountain hall): each level smaller than the one below."""
    x, y, z = at
    out = []
    lh = h / steps
    for k in range(steps):
        f = shrink ** k
        out += box((x, y + k * lh, z), (w * f, lh, d * f), color=color if k % 2 == 0 else GRANITE_D)
    return out


def chimney(x, z, h, y=0.0, w=0.9):
    return (box((x, y, z), (w, h, w), color=GRANITE_D) + box((x, y + h, z), (w + 0.2, 0.3, w + 0.2), color=BRONZE, kind="metal", detail=1) +
            sphere((x, y + h + 0.9, z), 0.6, seg=6, rings=3, color=STEAM, detail=2))


def gear(x, y, z, r, ry=0, detail=1):
    """A bronze cog wheel standing upright, facing +z (turned ry°)."""
    out = rod((x, y, z - 0.12), (x, y, z + 0.12), r, seg=10, color=BRONZE, kind="metal", detail=detail)
    for k in range(8):
        a = 2 * math.pi * k / 8
        out += box((x + r * math.cos(a), y + r * math.sin(a) - 0.12, z), (0.25, 0.25, 0.24), color=BRONZE, kind="metal", detail=max(detail, 2))
    return turn(out, ry, (x, 0, z)) if ry else out


def rune_door(x, z, w=2.4, h=2.8, y=0.0):
    """A heavy bronze door with a glowing rune above."""
    return (box((x, y, z), (w + 0.6, h + 0.5, 0.4), color=GRANITE_L) + box((x, y, z + 0.1), (w, h, 0.3), color=BRONZE, kind="metal") +
            box((x, y + h + 0.6, z + 0.2), (0.6, 0.5, 0.08), color=RUNE, kind="glow", detail=1))


def keep_dwarves():
    out = box((0, 0, 0), (13.8, 1.0, 13.8), color=GRANITE_D)
    out += stepped(13.0, 13.0, 12.0, steps=3, at=(0, 1.0, 0))
    out += box((0, 13.0, 0), (6.6, 2.4, 6.6), color=GRANITE_L)
    out += hip((0, 15.4, 0), 7.2, 7.2, 2.4, color=BRONZE, kind="metal")
    for x in (-5.4, 5.4):
        for z in (-5.4, 5.4):
            out += box((x, 1.0, z), (2.6, 9.0, 2.6), color=GRANITE)
            out += box((x, 10.0, z), (3.0, 0.5, 3.0), color=BRONZE, kind="metal")
            out += battlements((x, 10.5, z), 3.0, 3.0, size=0.5, gap=0.5, color=GRANITE_L)
    out += chimney(-2.4, -4.0, 6.0, y=9.0) + chimney(3.0, -3.6, 5.0, y=9.0)
    out += rune_door(0, 6.6, w=3.2, h=3.4, y=1.0)                                # the hall gate (and tunnel entrance)
    for x in (-3.0, 3.0):                                                        # guardian statues
        out += box((x, 1.0, 7.2), (1.2, 1.0, 1.2), color=GRANITE_L, detail=1)
        out += box((x, 2.0, 7.2), (0.9, 2.2, 0.7), color=GRANITE, detail=1)
        out += box((x, 4.2, 7.2), (0.6, 0.7, 0.6), color=GRANITE, detail=1)
    out += banner((-1.8, 7.8, 6.62), w=1.0, h=2.6, pole=False)
    out += banner((1.8, 7.8, 6.62), w=1.0, h=2.6, pole=False)
    out += windows([-4.0, 4.0], 5.0, 6.52, w=0.6, h=0.9, color=FIRE, kind="glow")
    return out


def dwarf_barracks():
    out = box((0, 0, -1.2), (8.6, 5.0, 6.4), color=GRANITE)
    out += box((0, 5.0, -1.2), (9.0, 0.4, 6.8), color=GRANITE_D)
    out += battlements((0, 5.4, -1.2), 9.0, 6.8, size=0.6, gap=0.6, color=GRANITE_L)
    out += rune_door(0, 2.02, w=2.0, h=2.4)
    for x in (-3.0, 3.0):                                                          # round bronze shields
        out += rod((x, 3.0, 2.0), (x, 3.0, 2.15), 0.7, seg=10, color=BRONZE, kind="metal", detail=1)
    out += banner((-3.0, 4.6, 2.03), w=0.8, h=1.0, pole=False)
    out += banner((3.0, 4.6, 2.03), w=0.8, h=1.0, pole=False)
    for x in (1.5, 2.3, 3.1):                                                      # axe rack
        out += rod((x, 0, 3.4), (x, 1.6, 3.4), 0.05, seg=4, color=WOOD_D, detail=2)
        out += box((x + 0.15, 1.3, 3.4), (0.3, 0.35, 0.06), color=IRON, kind="metal", detail=2)
    out += anvil(-2.4, 3.4)
    return out


def dwarf_workshop():
    out = box((0, 0, -2.0), (11.0, 5.6, 7.0), color=GRANITE)
    out += box((0, 5.6, -2.0), (11.4, 0.5, 7.4), color=BRONZE, kind="metal")
    out += box((0, 0, 1.5), (5.0, 4.0, 0.2), color=DARK)                             # big doors
    out += chimney(-4.0, -4.2, 5.0, y=5.6) + chimney(-2.6, -4.2, 4.0, y=5.6)
    out += gear(3.0, 3.6, 1.6, 1.4)
    out += gear(4.5, 2.0, 1.6, 0.9, detail=2)
    out += rod((4.6, 0, 3.6), (4.6, 7.0, 3.6), 0.2, seg=5, color=WOOD_D)              # crane
    out += rod((4.6, 6.8, 3.6), (1.4, 6.8, 5.4), 0.14, seg=5, color=BRONZE, kind="metal")
    out += rod((1.4, 6.8, 5.4), (1.4, 3.6, 5.4), 0.03, seg=3, color=IRON, detail=1)
    out += rod((-5.0, 4.6, 1.6), (5.0, 4.6, 1.6), 0.16, seg=6, color=BRONZE, kind="metal", detail=1)  # steam pipe
    out += box((-3.4, 0, 4.0), (2.4, 1.4, 1.6), color=WOOD, detail=1)              # crates
    out += box((-3.0, 1.4, 4.0), (1.2, 1.0, 1.2), color=WOOD_D, detail=2)
    out += banner((-1.0, 5.2, 1.52), w=1.2, h=1.8, pole=False)
    return out


def bear_grounds():
    out = box((0, 0, 0), (9.6, 0.1, 9.6), color=DIRT)
    for (x0, z0), (x1, z1) in (((-4.6, -4.6), (4.6, -4.6)), ((4.6, -4.6), (4.6, 4.6)), ((-4.6, -4.6), (-4.6, 4.6)),
                               ((-4.6, 4.6), (-1.6, 4.6)), ((1.6, 4.6), (4.6, 4.6))):
        cx, cz = (x0 + x1) / 2, (z0 + z1) / 2
        out += box((cx, 0, cz), (max(abs(x1 - x0), 0.6), 1.6, max(abs(z1 - z0), 0.6)), color=GRANITE)
    out += hall(5.0, 3.0, 2.4, WOOD, TURF, at=(0, 0, -2.8), roof_h=1.6)          # log shelter with a turf roof
    out += box((-1.6, 1.6, 4.6), (0.7, 1.0, 0.7), color=GRANITE_L, detail=1)
    out += box((1.6, 1.6, 4.6), (0.7, 1.0, 0.7), color=GRANITE_L, detail=1)
    out += dome((2.6, 0, 1.2), 1.0, h=0.8, seg=7, rings=3, color=STRAW, detail=1)
    out += cyl((-2.4, 0, 1.6), 0.6, 0.6, seg=8, color=WOOD_D, detail=1)            # a trough
    out += banner((0, 2.2, -1.28), w=0.9, h=1.2, pole=False)
    return out


def golem_workshop():
    out = box((0, 0, -2.4), (11.0, 6.4, 7.0), color=GRANITE)
    out += hip((0, 6.4, -2.4), 11.6, 7.6, 2.4, ridge=4.0, color=BRONZE, kind="metal")
    out += chimney(3.6, -5.0, 4.0, y=6.4, w=1.2)
    out += box((0, 0, 1.12), (5.0, 5.0, 0.1), color=DARK)
    out += box((-2.0, 0, 1.2), (1.6, 1.4, 0.1), color=RUNE, kind="glow")            # the rune forge
    # a golem standing in front, half built
    g = box((2.0, 0, 3.6), (0.7, 1.6, 0.7), color=GRANITE_L) + box((3.2, 0, 3.6), (0.7, 1.6, 0.7), color=GRANITE_L)
    g += box((2.6, 1.6, 3.6), (2.4, 2.2, 1.4), color=GRANITE_L)
    g += box((2.6, 3.8, 3.6), (0.9, 0.8, 0.9), color=GRANITE_L)
    g += box((1.1, 1.8, 3.6), (0.6, 2.0, 0.6), color=GRANITE_L) + box((4.1, 1.8, 3.6), (0.6, 2.0, 0.6), color=GRANITE_L)
    g += box((2.6, 2.6, 4.32), (0.6, 0.4, 0.06), color=RUNE, kind="glow", detail=1)
    out += g
    out += rod((0.6, 0, 5.0), (0.6, 5.0, 4.6), 0.12, seg=4, color=WOOD_D, detail=1)    # scaffold
    out += rod((4.8, 0, 5.0), (4.8, 5.0, 4.6), 0.12, seg=4, color=WOOD_D, detail=1)
    out += rod((0.6, 4.6, 4.6), (4.8, 4.6, 4.6), 0.1, seg=4, color=WOOD_D, detail=1)
    out += banner((-3.6, 5.6, 1.13), w=1.0, h=1.8, pole=False)
    return out


def dwarf_forge():
    out = box((0, 0, -1.6), (7.6, 4.4, 6.0), color=GRANITE)
    out += box((0, 4.4, -1.6), (8.0, 0.4, 6.4), color=GRANITE_D)
    out += box((2.6, 0, -3.4), (2.2, 10.0, 2.2), color=GRANITE_D)                   # the great chimney
    out += box((2.6, 10.0, -3.4), (2.6, 0.5, 2.6), color=BRONZE, kind="metal")
    out += sphere((2.6, 11.4, -3.4), 0.9, seg=6, rings=3, color=STEAM, detail=2)
    out += box((-1.0, 0, 1.42), (3.0, 2.6, 0.1), color=DARK)
    out += box((-1.0, 0, 1.5), (2.4, 1.4, 0.1), color=FIRE, kind="glow")
    out += anvil(1.4, 3.0, ry=20) + anvil(-2.2, 3.2)
    out += cyl((3.2, 0, 2.6), 0.5, 0.8, seg=8, color=BRONZE, kind="metal", detail=1)
    out += banner((1.6, 3.8, 1.42), w=0.8, h=1.4, pole=False)
    return out


def rune_hall():
    out = box((0, 0, 0), (9.6, 0.6, 9.6), color=GRANITE_D)
    out += box((0, 0.6, -1.4), (7.0, 5.0, 5.6), color=GRANITE)
    out += dome((0, 5.6, -1.4), 3.0, h=2.2, seg=10, rings=3, color=BRONZE, kind="metal")
    for x in (-3.0, -1.0, 1.0, 3.0):                                               # columns
        out += box((x, 0.6, 2.0), (0.7, 5.0, 0.7), color=GRANITE_L)
    out += box((0, 5.6, 2.0), (7.4, 0.6, 1.0), color=GRANITE_L)
    out += rune_door(0, 1.42, w=1.8, h=2.6, y=0.6)
    for x, z in ((-3.6, 3.8), (3.6, 3.8), (0, 4.2)):                              # rune stones
        out += box((x, 0.6, z), (0.9, 2.0, 0.5), color=GRANITE_L, detail=1)
        out += box((x, 1.4, z + 0.26), (0.4, 0.6, 0.04), color=RUNE, kind="glow", detail=1)
    out += banner((-2.0, 5.4, 2.36), w=0.8, h=1.6, pole=False)
    out += banner((2.0, 5.4, 2.36), w=0.8, h=1.6, pole=False)
    return out


def tunnel(undermine=False):
    """A tunnel entrance: a stone (or, for Undermine, timber-framed) arch into a mound, mine-cart rails."""
    rock = (0.40, 0.36, 0.32) if undermine else GRANITE_D
    out = rocks(0, -1.4, r=3.0, n=4, color=rock)
    frame = WOOD if undermine else GRANITE_L
    out += box((0, 0, 1.4), (3.2, 3.2, 0.8), color=frame)
    out += box((0, 0, 1.8), (2.2, 2.4, 0.1), color=DARK)
    if not undermine:
        out += box((0, 3.2, 1.4), (3.6, 0.5, 1.0), color=BRONZE, kind="metal")
        out += box((0, 2.6, 1.86), (0.5, 0.4, 0.06), color=RUNE, kind="glow", detail=1)
        out += banner((-2.2, 3.0, 1.9), w=0.7, h=1.2, pole=False)
    else:
        out += rod((-1.4, 0, 1.9), (-1.2, 3.0, 1.9), 0.15, seg=4, color=WOOD_D)
        out += rod((1.4, 0, 1.9), (1.2, 3.0, 1.9), 0.15, seg=4, color=WOOD_D)
        out += rod((-1.6, 3.0, 1.9), (1.6, 3.0, 1.9), 0.15, seg=4, color=WOOD_D)
        out += box((0, 0, 2.8), (3.6, 0.5, 1.2), color=DIRT)
    for x in (-0.5, 0.5):                                                          # rails
        out += box((x, 0, 3.2), (0.1, 0.1, 3.0), color=IRON, kind="metal", detail=2)
    out += box((1.8, 0, 3.0), (1.0, 0.8, 1.4), color=WOOD_D, detail=1)             # a mine cart
    return out


def dwarf_farm():
    out = hall(4.0, 3.4, 2.2, GRANITE, TURF, at=(-1.8, 0, -1.8), roof_h=1.4)
    out += door(-1.8, -0.08, w=1.0, h=1.4, color=BRONZE)
    out += chimney(-3.0, -2.8, 2.2, y=1.2, w=0.6)
    for k, col in enumerate([(0.70, 0.62, 0.30), (0.55, 0.45, 0.35), (0.72, 0.64, 0.32)]):
        out += box((2.2, 0, -2.4 + k * 2.2), (3.0, 0.3, 1.8), color=col)
    for x, z in ((-2.6, 2.2), (-1.0, 2.6), (0.2, 1.8)):                            # big mushrooms
        out += cyl((x, 0, z), 0.12, 0.7, seg=5, color=BONE, detail=1)
        out += dome((x, 0.7, z), 0.5, h=0.3, seg=6, rings=2, color=(0.60, 0.42, 0.28), detail=1)
    out += box((3.8, 0, 0), (0.4, 1.0, 7.6), color=GRANITE_D, detail=1)            # stone wall
    out += banner((-1.8, 2.0, -0.08), w=0.5, h=0.7, pole=False)
    return out


def squat_tower(top_h=7.5):
    out = box((0, 0, 0), (5.8, 1.2, 5.8), color=GRANITE_D)
    out += box((0, 1.2, 0), (5.0, top_h - 1.2, 5.0), color=GRANITE)
    out += box((0, top_h, 0), (5.6, 0.5, 5.6), color=BRONZE, kind="metal")
    out += windows([-1.2, 1.2], top_h * 0.6, 2.52, w=0.6, h=0.25)
    out += banner((0, top_h - 0.4, 2.52), w=1.0, h=1.6, pole=False)
    return out


def mortar_tower():
    out = squat_tower(7.0)
    out += battlements((0, 7.5, 0), 5.6, 5.6, size=0.6, gap=0.7, color=GRANITE_L)
    out += cyl((0, 7.5, 0), 1.2, 0.6, seg=8, color=IRON, kind="metal")
    out += rod((0, 8.0, 0), (0, 9.6, 0.9), 0.55, r_b=0.48, seg=8, color=IRON, kind="metal")   # the mortar barrel
    return out


def gunner_tower():
    out = squat_tower(8.5)
    for k in range(4):
        a = math.pi / 4 + k * math.pi / 2
        out += rod((2.4 * math.cos(a), 9.0, 2.4 * math.sin(a)), (2.4 * math.cos(a), 10.6, 2.4 * math.sin(a)), 0.18, seg=5, color=BRONZE, kind="metal")
    out += hip((0, 10.6, 0), 6.0, 6.0, 1.8, color=BRONZE, kind="metal")
    out += box((0, 9.0, 2.3), (1.6, 0.25, 0.25), color=IRON, kind="metal", detail=1)   # a gun barrel poking out
    return out


def rune_tower():
    out = squat_tower(8.0)
    out += battlements((0, 8.5, 0), 5.6, 5.6, size=0.6, gap=0.7, color=GRANITE_L)
    out += box((0, 8.5, 0), (1.4, 3.0, 1.4), color=GRANITE_L)                       # a rune pillar
    out += box((0, 11.5, 0), (1.0, 1.0, 1.0), ry=45, color=RUNE, kind="glow")
    out += box((0, 9.4, 0.72), (0.6, 0.8, 0.06), color=RUNE, kind="glow", detail=1)
    return out


PROPS = [
    Prop("keep_dwarves", keep_dwarves(), "Dwarf keep: a stepped granite hall, corner towers, bronze roof, rune gate"),
    Prop("dwarf_barracks", dwarf_barracks(), "Dwarf barracks: a battlemented stone block, bronze shields"),
    Prop("dwarf_workshop", dwarf_workshop(), "Dwarf workshop: stone hall, gears, chimneys, crane, steam pipe"),
    Prop("bear_grounds", bear_grounds(), "Bear grounds: stone enclosure, a turf-roofed shelter"),
    Prop("golem_workshop", golem_workshop(), "Golem workshop: bronze-roofed hall, rune forge, a golem in scaffolding"),
    Prop("dwarf_forge", dwarf_forge(), "Dwarf forge: stone forge with a great chimney, anvils"),
    Prop("rune_hall", rune_hall(), "Rune hall: a columned hall under a bronze dome, rune stones"),
    Prop("tunnel", tunnel(), "Tunnel: a stone arch into a mound, rails, a mine cart"),
    Prop("undermine_tunnel", tunnel(undermine=True), "Undermine passage: a timber-framed hole in a dirt mound"),
    Prop("dwarf_farm", dwarf_farm(), "Dwarf farm: a turf-roofed stone cottage, fields, mushrooms"),
    Prop("mortar_tower", mortar_tower(), "Mortar tower: squat tower with a mortar on top"),
    Prop("gunner_tower", gunner_tower(), "Gunner tower: squat tower with a bronze roof on posts"),
    Prop("rune_tower", rune_tower(), "Rune tower: squat tower with a glowing rune crystal"),
]
