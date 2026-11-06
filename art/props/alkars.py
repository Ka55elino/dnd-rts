"""Alkars: dark volcanic stone, black metal, porcelain-white aristocratic details, greenish necrotic
light; crude wood, hides and bones for the orcs, goblins and trolls."""
import math
from art.props._style import *  # noqa: F401,F403

BASALT = (0.20, 0.19, 0.19)
BASALT_L = (0.30, 0.28, 0.27)
BLACKM = (0.14, 0.14, 0.15)     # black metal
PORCELAIN = (0.90, 0.89, 0.86)
NECRO = (0.45, 0.95, 0.55)      # glow
HIDE = (0.55, 0.42, 0.30)
LOG = (0.40, 0.29, 0.18)


def brazier(x, z, y=0.0, detail=1):
    return (cyl((x, y, z), 0.15, 1.4, seg=5, color=BLACKM, detail=detail) +
            cyl((x, y + 1.4, z), 0.5, 0.4, r_top=0.65, seg=6, color=BLACKM, detail=detail) +
            cone((x, y + 1.75, z), 0.45, 0.7, seg=5, color=NECRO, kind="glow", detail=detail))


def skull(x, y, z, r=0.3, detail=2):
    return sphere((x, y, z), r, seg=6, rings=3, color=BONE, detail=detail)


def hut(x, z, r=1.6, h=1.6, roof=HIDE, wall=LOG):
    """A crude round hut: log wall and a hide cone roof."""
    return cyl((x, 0, z), r, h, seg=7, color=wall) + cone((x, h, z), r + 0.3, r * 1.4, seg=7, color=roof)


def tent(x, z, r=1.4, h=2.4):
    return cone((x, 0, z), r, h, seg=6, color=HIDE) + rod((x, h - 0.3, z), (x + 0.2, h + 0.5, z), 0.05, seg=3, color=LOG, detail=2)


def totem(x, z, h=3.0):
    return (rod((x, 0, z), (x, h, z), 0.16, seg=5, color=LOG, detail=1) + skull(x, h + 0.15, z, 0.3, detail=1) +
            rod((x - 0.6, h - 0.6, z), (x + 0.6, h - 0.6, z), 0.08, seg=4, color=BONE, detail=2))


def keep_alkars():
    out = box((0, 0, 0), (13.6, 1.0, 13.6), color=BASALT_L)
    out += box((0, 1.0, 0), (10.4, 9.0, 10.4), color=BASALT)
    out += box((0, 10.0, 0), (11.2, 0.6, 11.2), color=BLACKM)
    out += battlements((0, 10.6, 0), 11.2, 11.2, size=0.7, gap=0.8, h=1.2, color=BASALT_L)
    out += box((0, 10.6, -1.0), (6.0, 6.0, 6.0), color=BASALT)                     # inner keep
    out += hip((0, 16.6, -1.0), 7.0, 7.0, 3.2, color=BLACKM)
    for x in (-5.6, 5.6):
        for z in (-5.6, 5.6):
            out += box((x, 0, z), (3.4, 14.0, 3.4), color=BASALT)
            out += box((x, 14.0, z), (3.9, 0.5, 3.9), color=BLACKM)
            for k in range(4):                                                     # bone spikes on the corners
                a = math.pi / 4 + k * math.pi / 2
                out += spike((x + 1.6 * math.cos(a), 14.5, z + 1.6 * math.sin(a)),
                             (x + 2.4 * math.cos(a), 16.4, z + 2.4 * math.sin(a)), 0.18, color=BONE, detail=1)
            out += brazier(x, z, y=14.5)
    out += door(0, 5.25, w=2.8, h=3.0, y=1.0, color=BLACKM)
    out += box((0, 5.6, 5.3), (4.4, 0.6, 0.3), color=PORCELAIN, detail=1)           # a white lintel
    out += windows([-2.8, 2.8], 6.4, 5.22, w=0.5, h=1.6, color=NECRO, kind="glow")
    out += windows([0], 13.0, 2.02, w=0.6, h=1.6, color=NECRO, kind="glow")
    for x in (-2.8, 2.8):
        out += banner((x, 9.6, 5.24), w=1.4, h=3.4, pole=False)
    out += skull(0, 9.0, 5.3, 0.45, detail=1)
    return out


def orc_pits():
    out = box((0, 0, 0), (9.6, 0.1, 9.6), color=DIRT)
    pts = circle(14, 4.5, start=math.pi / 2 + 0.25)
    out += palisade(pts[:-1], h=2.6, r=0.2)
    out += tent(-2.2, -1.8, 1.5, 2.6) + tent(1.8, -2.4, 1.3, 2.2) + tent(2.4, 1.0, 1.1, 2.0)
    out += cyl((-0.6, 0, 1.2), 1.2, 0.3, seg=8, color=(0.25, 0.20, 0.16))           # the pit
    out += totem(-1.0, 4.4, 3.4) + totem(1.0, 4.4, 3.4)
    out += brazier(-3.0, 2.4, detail=1)
    out += banner((0, 3.6, 4.6), w=1.4, h=1.4)
    return out


def dark_order():
    """The Alkar aristocracy's hall: black stone, porcelain-white pillars, pointed arches, green light."""
    out = box((0, 0, 0), (9.8, 0.6, 9.8), color=BASALT_L)
    out += hall(7.0, 6.0, 6.0, BASALT, BLACKM, roof="hip", at=(0, 0.6, -1.2), roof_h=4.5, ridge=1.0)
    for x in (-2.8, -0.95, 0.95, 2.8):
        out += cyl((x, 0.6, 2.4), 0.28, 5.6, seg=6, color=PORCELAIN)
    out += box((0, 6.2, 2.4), (6.8, 0.5, 0.8), color=PORCELAIN)
    for x in (-3.5, 3.5):                                                          # thin pointed spires
        out += cyl((x, 0.6, -3.8), 0.7, 9.0, seg=6, color=BASALT)
        out += cone((x, 9.6, -3.8), 0.9, 3.6, seg=6, color=BLACKM)
        out += sphere((x, 8.6, -2.95), 0.25, seg=6, rings=3, color=NECRO, kind="glow", detail=1)
    out += door(0, 1.85, w=1.8, h=3.0, y=0.6, color=BLACKM)
    out += windows([-2.0, 2.0], 3.0, 1.85, w=0.5, h=2.0, color=NECRO, kind="glow")
    out += banner((0, 6.0, 2.85), w=1.2, h=2.2, pole=False)
    return out


def dark_order_enchantments():
    out = box((0, 0, 0), (9.6, 0.5, 9.6), color=BASALT_L)
    out += prism((0, 0.5, 0), [(1.8, 1.8), (-1.8, 1.8), (-1.8, -1.8), (1.8, -1.8)], 9.0, color=BASALT)  # obelisk
    out += hip((0, 9.5, 0), 3.6, 3.6, 3.0, color=BLACKM)
    for z in (1.82,):
        for y in (2.5, 4.5, 6.5):
            out += box((0, y, z), (1.2, 0.3, 0.08), color=NECRO, kind="glow", detail=1)   # runes
    for k in range(8):                                                             # ring of bone pillars
        a = 2 * math.pi * k / 8
        x, z = 4.0 * math.cos(a), 4.0 * math.sin(a)
        out += cyl((x, 0.5, z), 0.22, 2.6, seg=5, color=PORCELAIN)
        out += spike((x, 3.1, z), (x * 0.92, 4.0, z * 0.92), 0.2, color=BONE, detail=1)
    out += glow_pool(0, 3.4, 0.8, NECRO, BASALT_L, y=0.5, rim_h=0.5, seg=8)
    out += banner((0, 8.4, 1.84), w=1.0, h=1.4, pole=False)
    return out


def harpy_nest():
    out = rocks(0, 0, r=3.0, n=4, color=BASALT_L)
    out += cone((0, 1.0, 0), 2.6, 11.0, seg=7, color=BASALT)                       # a rock spire
    for y, r, ang in ((5.0, 1.6, 0.4), (8.0, 1.3, 2.4), (10.2, 1.0, 4.4)):         # nests on the spire's sides
        rr = 2.6 * (1 - (y - 1.0) / 11.0) + r * 0.55
        x, z = rr * math.cos(ang), rr * math.sin(ang)
        out += cyl((x, y, z), r, 0.5, r_top=r * 1.2, seg=8, color=(0.40, 0.32, 0.20))
        out += rod((x * 0.3, y - 0.1, z * 0.3), (x, y, z), 0.12, seg=4, color=LOG, detail=1)
    for k in range(4):
        a = k * 1.6
        out += rod((3.0 * math.cos(a), 0, 3.0 * math.sin(a)), (3.6 * math.cos(a), 1.6, 3.6 * math.sin(a)), 0.1, seg=4, color=BONE, detail=1)
    out += banner((0.4, 6.5, 2.2), w=1.0, h=1.8, pole=False, ry=10)
    return out


def wolf_den():
    out = rocks(-0.6, -1.6, r=3.4, n=4, color=BASALT_L)
    out += box((-0.6, 0, 1.6), (2.0, 1.8, 0.4), color=DARK)                        # den mouth
    out += hut(2.8, 2.0, r=1.5, h=1.5)
    out += fence([(-4.4, 4.4), (4.4, 4.4)], color=LOG)
    for x in (-2.6, -1.0):
        out += skull(x, 1.9, 4.4, 0.3, detail=1)
    out += totem(-3.4, 2.2, 2.8)
    out += banner((2.8, 3.4, 3.6), w=0.9, h=1.2)
    return out


def dark_cult():
    out = box((0, 0, 0), (9.4, 0.1, 9.4), color=(0.24, 0.22, 0.20))
    for k in range(7):                                                             # standing stones
        a = 2 * math.pi * k / 7 + 0.2
        x, z = 3.9 * math.cos(a), 3.9 * math.sin(a)
        out += box((x, 0, z), (0.9, 3.4 + (k % 3) * 0.6, 0.6), ry=-math.degrees(a), color=BASALT)
    out += box((0, 0, 0), (2.2, 1.0, 1.4), color=BASALT_L)                          # altar
    out += box((0, 1.0, 0), (1.6, 0.1, 0.9), color=NECRO, kind="glow")
    out += brazier(-1.6, 1.4) + brazier(1.6, 1.4)
    out += hut(0, -2.6, r=1.4, h=2.0, roof=BLACKM, wall=BASALT)
    out += banner((0, 3.6, 4.2), w=1.0, h=1.6)
    return out


def troll_arena():
    out = box((0, 0, 0), (12.6, 0.1, 12.6), color=DIRT)
    pts = circle(18, 6.0, start=math.pi / 2 + 0.2)
    out += palisade(pts[:-1], h=3.6, r=0.3, step=0.6)
    for k in range(4):                                                             # huge ribs over the gate
        a = math.pi / 2 + (k - 1.5) * 0.18
        x, z = 6.0 * math.cos(a), 6.0 * math.sin(a)
        out += rod((x, 0, z), (x * 0.5, 6.0, z * 0.5 + 1.0), 0.2, r_b=0.12, seg=5, color=BONE, detail=1)
    out += box((-2.4, 0, -2.0), (3.0, 3.0, 3.0), color=(0.0, 0.0, 0.0), detail=2)  # a cage (dark box with bars)
    for k in range(5):
        out += rod((-3.8 + k * 0.7, 0, -0.45), (-3.8 + k * 0.7, 3.2, -0.45), 0.07, seg=4, color=BLACKM, kind="metal", detail=1)
    out += box((-2.4, 3.0, -2.0), (3.2, 0.3, 3.2), color=LOG)
    out += brazier(2.6, 2.4) + brazier(-1.0, 3.4)
    out += totem(3.4, -2.6, 4.2)
    out += banner((0, 4.6, 6.2), w=1.6, h=2.0)
    return out


def alkar_workshop():
    out = hall(7.0, 5.0, 4.2, BASALT, BLACKM, roof="gable", at=(0, 0, -2.0), roof_h=2.6)
    out += box((2.8, 0, -3.6), (1.4, 8.0, 1.4), color=BASALT_L)                     # chimney
    out += box((2.8, 8.0, -3.6), (1.7, 0.4, 1.7), color=BLACKM)
    out += box((-0.8, 0, 0.52), (2.6, 2.6, 0.1), color=DARK)
    out += box((-0.8, 0, 0.6), (2.0, 1.2, 0.1), color=NECRO, kind="glow")         # green forge fire
    out += anvil(1.4, 2.0)
    for x in (-3.6, 3.6):
        out += spike((x, 4.2, -2.0), (x * 1.15, 6.2, -2.0), 0.2, color=BLACKM, kind="metal", detail=1)
    out += box((-2.8, 0, 2.6), (1.4, 0.8, 1.0), color=LOG, detail=1)              # sacks of rocks
    out += sphere((-2.8, 1.0, 2.6), 0.5, seg=6, rings=3, color=HIDE, detail=1)
    out += banner((-2.8, 3.8, 0.52), w=0.9, h=1.6, pole=False)
    return out


def goblin_farm():
    out = box((0, 0, 0), (7.6, 0.1, 7.6), color=(0.30, 0.26, 0.20))
    out += hut(-2.0, -2.0, r=1.7, h=1.8, roof=(0.36, 0.30, 0.22))
    for x, z, h, r, cap in ((1.0, -2.4, 1.0, 0.7, (0.55, 0.30, 0.55)), (2.6, -0.8, 0.8, 0.55, (0.45, 0.60, 0.30)),  # mushrooms
                            (0.8, 0.6, 1.3, 0.8, (0.55, 0.30, 0.55)), (2.4, 2.0, 0.7, 0.5, (0.70, 0.55, 0.30)),
                            (-1.2, 1.8, 1.1, 0.7, (0.45, 0.60, 0.30)), (-2.8, 2.4, 0.6, 0.45, (0.55, 0.30, 0.55))):
        out += cyl((x, 0, z), 0.16, h, seg=5, color=PORCELAIN, detail=1)
        out += dome((x, h, z), r, h=r * 0.6, seg=7, rings=2, color=cap)
    out += fence([(-3.6, 3.6), (3.6, 3.6), (3.6, -3.6)], color=LOG)
    out += banner((-2.0, 1.8, -0.3), w=0.6, h=0.8, pole=False)
    return out


def wraith_tower():
    out = box((0, 0, 0), (5.6, 1.0, 5.6), color=BASALT_L)
    out += box((0, 1.0, 0), (4.4, 9.0, 4.4), color=BASALT)
    out += box((0, 10.0, 0), (5.0, 0.5, 5.0), color=BLACKM)
    for k in range(4):
        a = math.pi / 4 + k * math.pi / 2
        out += spike((2.2 * math.cos(a), 10.5, 2.2 * math.sin(a)), (3.0 * math.cos(a), 12.6, 3.0 * math.sin(a)), 0.2, color=BONE)
    out += sphere((0, 11.8, 0), 0.9, seg=8, rings=4, color=NECRO, kind="glow")    # the necrotic eye
    out += cyl((0, 10.5, 0), 0.6, 0.6, seg=6, color=BLACKM)
    out += windows([0], 6.0, 2.22, w=0.4, h=1.4, color=NECRO, kind="glow")
    out += banner((0, 9.4, 2.23), w=0.9, h=2.4, pole=False)
    return out


PROPS = [
    Prop("keep_alkars", keep_alkars(), "Alkar keep: massive basalt block, corner towers with bone spikes, green braziers"),
    Prop("orc_pits", orc_pits(), "Orc pits: a palisade ring, hide tents, a pit, totems"),
    Prop("dark_order", dark_order(), "Dark Order: black hall, porcelain-white colonnade, thin spires"),
    Prop("dark_order_enchantments", dark_order_enchantments(), "Dark Order Enchantments: rune obelisk in a ring of bone pillars"),
    Prop("harpy_nest", harpy_nest(), "Harpy nest: a rock spire with nests"),
    Prop("wolf_den", wolf_den(), "Wolf den: a rock den, a hut, skulls on the fence"),
    Prop("dark_cult", dark_cult(), "Dark cult: standing stones around a glowing altar"),
    Prop("troll_arena", troll_arena(), "Troll arena: a big palisade ring, huge ribs, a cage"),
    Prop("alkar_workshop", alkar_workshop(), "Alkar workshop: black smithy with a green forge fire"),
    Prop("goblin_farm", goblin_farm(), "Goblin farm: a hut among giant mushrooms"),
    Prop("wraith_tower", wraith_tower(), "Wraith tower: basalt tower with bone spikes and a necrotic eye"),
]
