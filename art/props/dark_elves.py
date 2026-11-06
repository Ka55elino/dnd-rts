"""Dark Elves: black and purple, spikes, obsidian, lilac lights, webs."""
import math
from art.props._style import *  # noqa: F401,F403

OBS = (0.13, 0.11, 0.15)        # obsidian
OBS_L = (0.22, 0.19, 0.25)
PURPLE = (0.36, 0.17, 0.46)
LILAC = (0.78, 0.55, 1.0)       # glow
WEB = (0.85, 0.85, 0.88)
ROCK = (0.25, 0.23, 0.26)


def needle(x, z, r, h, y=0.0, seg=6, color=OBS):
    """A spiky obsidian spire: a tall thin cone."""
    return cone((x, y, z), r, h, seg=seg, color=color)


def spikes_ring(x, z, r, n, h=1.4, y=0.0, out_tilt=0.5, color=OBS_L, detail=1):
    out = []
    for k in range(n):
        a = 2 * math.pi * k / n
        cx, cz = x + r * math.cos(a), z + r * math.sin(a)
        out += spike((cx, y, cz), (cx + out_tilt * math.cos(a), y + h, cz + out_tilt * math.sin(a)), 0.15, color=color, detail=detail)
    return out


def webs(points, color=WEB, detail=2):
    """Thin strands between points (x, y, z)."""
    out = []
    for a, b in zip(points, points[1:]):
        out += rod(a, b, 0.025, seg=3, color=color, detail=detail)
    return out


def keep_dark_elves():
    out = prism((0, 0, 0), [(6.6, 0), (3.3, 5.7), (-3.3, 5.7), (-6.6, 0), (-3.3, -5.7), (3.3, -5.7)], 0.6, color=OBS_L)
    out += prism((0, 0.6, 0), [(5.6, 0), (2.8, 4.85), (-2.8, 4.85), (-5.6, 0), (-2.8, -4.85), (2.8, -4.85)], 7.0, color=OBS)
    out += cyl((0, 7.6, 0), 3.4, 6.0, r_top=2.6, seg=6, color=OBS_L)
    out += needle(0, 0, 2.6, 12.0, y=13.6, color=OBS)
    out += needle(0, 0, 1.2, 4.0, y=23.0, color=PURPLE)
    for k in range(6):
        a = 2 * math.pi * k / 6
        x, z = 5.6 * math.cos(a), 5.6 * math.sin(a)
        out += needle(x, z, 1.0, 11.0 + 2.0 * (k % 2))
        out += sphere((x, 9.0, z), 0.35, seg=6, rings=3, color=LILAC, kind="glow", detail=1)
    out += spikes_ring(0, 0, 3.3, 10, h=2.0, y=7.6)
    out += door(0, 4.9, w=2.2, h=2.8, y=0.6, color=PURPLE)
    out += windows([-2.0, 2.0], 4.0, 4.9, w=0.5, h=1.6, color=LILAC, kind="glow")
    out += windows([0], 10.0, 2.85, w=0.6, h=1.8, color=LILAC, kind="glow")
    for x in (-3.6, 3.6):
        out += banner((x, 6.4, 3.9), w=1.0, h=3.0, pole=False, ry=(-30 if x < 0 else 30))
    out += webs([(-5.6, 6.0, 0.0), (-4.5, 9.0, 3.0), (-2.8, 6.5, 4.85)])
    return out


def fighting_pits():
    out = box((0, 0, 0), (9.6, 0.2, 9.6), color=SAND)
    pts = circle(12, 4.4)
    for (x0, z0), (x1, z1) in zip(pts, pts[1:]):
        if z0 > 3.6 and abs(x0) < 1.5:
            continue
        mx, mz = (x0 + x1) / 2, (z0 + z1) / 2
        out += box((mx, 0, mz), (math.hypot(x1 - x0, z1 - z0) + 0.1, 2.2, 0.6), ry=-math.degrees(math.atan2(z1 - z0, x1 - x0)), color=OBS)
    out += spikes_ring(0, 0, 4.4, 12, h=1.4, y=2.2, out_tilt=0.4)
    for x in (-1.6, 1.6):
        out += needle(x, 4.4, 0.6, 5.0)
    out += banner((0, 4.0, 4.7), w=1.6, h=1.6)
    out += rod((-1.0, 0.2, -1.0), (-1.0, 2.0, -1.0), 0.1, seg=4, color=WOOD_D, detail=1)    # a training post
    out += rod((1.4, 0.2, 0.4), (2.4, 1.2, 0.4), 0.05, seg=4, color=IRON, kind="metal", detail=2)  # blades on a rack
    out += needle(-4.2, -3.0, 0.5, 4.0)
    out += needle(4.2, -3.0, 0.5, 4.0)
    return out


def shooting_range():
    out = hall(8.0, 3.0, 3.2, OBS, PURPLE, roof="hip", at=(0, 0, -3.0), roof_h=2.2)
    out += needle(-4.0, -3.0, 0.5, 6.0) + needle(4.0, -3.0, 0.5, 6.0)
    out += door(0, -1.48, w=1.4, h=2.0, color=PURPLE)
    out += windows([-2.6, 2.6], 1.8, -1.48, w=0.5, h=1.0, color=LILAC, kind="glow")
    for x in (-2.8, 0.0, 2.8):
        out += target(x, 3.0, ry=180)
    out += spikes_ring(0, 1.0, 4.6, 14, h=1.0, out_tilt=0.0)
    out += banner((0, 3.0, -1.5), w=1.0, h=1.6, pole=False)
    return out


def scout_post():
    out = []
    for x in (-1.4, 1.4):
        for z in (-1.4, 1.4):
            out += rod((x * 1.2, 0, z * 1.2), (x, 4.2, z), 0.14, seg=5, color=OBS_L)
    out += box((0, 4.2, 0), (3.4, 2.4, 3.4), color=OBS)
    out += hip((0, 6.6, 0), 4.2, 4.2, 3.2, color=PURPLE)
    out += needle(0, 0, 0.4, 2.0, y=9.6)
    out += windows([0], 5.0, 1.72, w=0.8, h=0.6, color=LILAC, kind="glow")
    out += rod((0, 0, 3.2), (0, 4.2, 1.72), 0.07, seg=4, color=OBS_L, detail=1)   # ladder rail
    out += sphere((-2.6, 2.2, 2.6), 0.3, seg=6, rings=3, color=LILAC, kind="glow", detail=1)
    out += rod((-2.6, 0, 2.6), (-2.6, 2.0, 2.6), 0.06, seg=4, color=OBS, detail=1)
    out += banner((0, 6.2, 1.73), w=0.8, h=1.2, pole=False)
    return out


def spider_caves():
    out = rocks(0, -1.0, r=4.2, n=5, color=ROCK)
    out += box((0, 0, 2.6), (2.6, 2.4, 0.4), color=DARK)                          # the cave mouth
    out += rod((0, 2.4, 2.6), (0, 2.4, 2.9), 1.3, seg=10, color=DARK)
    out += webs([(-1.4, 0.2, 2.9), (-0.6, 2.0, 2.95), (0.8, 3.2, 2.95), (1.5, 0.4, 2.9)], detail=1)
    out += webs([(-1.4, 1.6, 2.95), (1.4, 1.8, 2.95)], detail=1)
    for x, z in ((-3.4, 2.2), (3.6, 1.6), (2.4, -3.6)):
        out += needle(x, z, 0.5, 3.6)
        out += sphere((x, 3.8, z), 0.25, seg=6, rings=3, color=LILAC, kind="glow", detail=1)
    out += sphere((-2.0, 0.4, 3.4), 0.45, seg=6, rings=3, color=WEB, detail=1)          # egg sacs
    out += sphere((2.2, 0.4, 3.2), 0.35, seg=6, rings=3, color=WEB, detail=1)
    out += banner((0, 3.8, 3.0), w=1.0, h=1.0, pole=False)
    return out


def dark_church():
    out = box((0, 0, 0), (12.4, 0.4, 12.4), color=OBS_L)
    out += hall(5.6, 9.0, 7.0, OBS, PURPLE, at=(0, 0.4, 0.4), roof_h=5.0)        # nave, steep roof
    out += box((0, 0.4, -4.4), (3.4, 15.0, 3.4), color=OBS)                       # bell tower
    out += needle(0, -4.4, 2.6, 8.0, y=15.4, seg=4)
    out += needle(0, -4.4, 0.6, 2.2, y=23.2, color=PURPLE)
    for x in (-3.4, 3.4):                                                         # buttresses
        for z in (-2.0, 1.0, 4.0):
            out += rod((x * 1.3, 0.4, z), (x, 6.6, z), 0.3, seg=4, color=OBS_L, detail=1)
    out += rod((0, 7.0, 4.95), (0, 7.0, 5.15), 1.4, seg=10, color=LILAC, kind="glow")  # rose window
    out += door(0, 4.95, w=2.0, h=3.4, y=0.4, color=PURPLE)
    out += windows([-2.8, 2.8], 3.0, 4.95, w=0.5, h=2.0, color=LILAC, kind="glow")
    out += banner((-1.8, 6.6, 4.96), w=0.8, h=2.2, pole=False)
    out += banner((1.8, 6.6, 4.96), w=0.8, h=2.2, pole=False)
    return out


def dark_dragon_lair():
    out = rocks(0, -0.8, r=5.4, n=6, color=ROCK)
    out += box((0, 0, 3.6), (4.6, 3.6, 0.6), color=DARK)                          # a huge cave mouth
    out += rod((0, 3.6, 3.6), (0, 3.6, 3.95), 2.3, seg=10, color=DARK)
    for k in range(7):                                                            # obsidian shards
        a = 0.3 + k * 0.85
        x, z = 5.6 * math.cos(a), -0.8 + 5.0 * math.sin(a)
        out += spike((x, 0, z), (x * 1.15, 6.0 + (k % 3) * 1.5, z * 1.1), 0.7, color=OBS)
    for k in range(5):                                                            # ribs of a dead beast
        x = -3.0 + k * 1.5
        out += rod((x, 0, 5.4), (x * 0.8, 2.6, 6.2), 0.12, seg=4, color=BONE, detail=1)
    out += box((0, 0.02, 4.4), (3.6, 0.1, 1.2), color=(0.9, 0.35, 0.9), kind="glow", detail=1)   # lilac embers
    out += banner((-3.2, 4.0, 4.1), w=1.2, h=2.0, ry=-20)
    return out


def alchemy_workshop():
    out = hall(7.0, 4.6, 4.0, OBS, PURPLE, roof="hip", at=(0, 0, -2.2), roof_h=2.4)
    for x, col in ((-3.0, (0.5, 0.95, 0.4)), (-1.0, LILAC), (1.2, (0.5, 0.95, 0.4))):  # vats
        out += cyl((x, 0, 2.4), 0.75, 1.4, seg=8, color=IRON, kind="metal")
        out += cyl((x, 1.4, 2.4), 0.62, 0.06, seg=8, color=col, kind="glow")
    out += cyl((2.8, 0, -3.4), 0.6, 8.0, r_top=0.45, seg=6, color=OBS_L)          # chimneys
    out += cyl((-2.6, 0, -3.6), 0.45, 7.0, r_top=0.35, seg=6, color=OBS_L)
    out += sphere((2.8, 8.2, -3.4), 0.4, seg=6, rings=3, color=(0.5, 0.95, 0.4), kind="glow", detail=1)
    out += rod((-3.0, 1.4, 2.4), (-2.0, 3.6, 0.1), 0.08, seg=4, color=IRON, kind="metal", detail=2)   # pipes
    out += rod((1.2, 1.4, 2.4), (1.2, 3.4, 0.1), 0.08, seg=4, color=IRON, kind="metal", detail=2)
    out += door(0.4, 0.12, w=1.4, h=2.0, color=PURPLE)
    out += windows([-2.4, 2.6], 2.2, 0.12, w=0.5, h=0.8, color=LILAC, kind="glow")
    out += banner((2.6, 3.6, 0.13), w=0.8, h=1.4, pole=False)
    return out


def spider_farm():
    out = []
    posts = [(-3.0, -3.0), (3.0, -3.2), (3.2, 2.8), (-3.2, 3.0), (0.0, 0.0)]
    for x, z in posts[:4]:
        out += rod((x, 0, z), (x * 0.9, 4.0, z * 0.9), 0.12, seg=5, color=OBS_L)
    out += rod((0, 0, 0), (0, 5.0, 0), 0.16, seg=5, color=OBS_L)
    for x, z in posts[:4]:                                                        # the web roof
        out += webs([(x * 0.9, 4.0, z * 0.9), (0, 5.0, 0)], detail=0)
    for (x0, z0), (x1, z1) in zip(posts[:4], posts[1:4] + posts[:1]):
        out += webs([(x0 * 0.9, 4.0, z0 * 0.9), ((x0 + x1) * 0.35, 3.4, (z0 + z1) * 0.35), (x1 * 0.9, 4.0, z1 * 0.9)], detail=1)
    for x, z, r in ((-1.4, 1.2, 0.5), (1.2, 1.6, 0.4), (1.6, -1.2, 0.55), (-1.6, -1.4, 0.45), (0.2, 2.6, 0.35)):  # egg sacs
        out += sphere((x, r, z), r, seg=7, rings=4, ry_scale=1.2, color=WEB)
    out += box((0, 0, 0), (7.4, 0.08, 7.4), color=(0.30, 0.26, 0.24))
    out += sphere((0, 5.2, 0), 0.3, seg=6, rings=3, color=LILAC, kind="glow", detail=1)
    out += banner((0, 3.8, 3.3), w=0.8, h=1.0, pole=False)
    return out


def crossbow_tower():
    """Shared by the Dark Elves and the Alkars: a dark, spiked stone tower with black metal."""
    out = box((0, 0, 0), (5.6, 1.0, 5.6), color=ROCK)
    out += cyl((0, 1.0, 0), 2.6, 9.0, r_top=2.2, seg=6, color=OBS)
    out += cyl((0, 10.0, 0), 2.9, 0.6, seg=6, color=OBS_L)
    out += spikes_ring(0, 0, 2.7, 6, h=1.6, y=10.6, out_tilt=0.6)
    out += needle(0, 0, 0.5, 3.0, y=10.6, color=IRON)
    out += windows([0], 6.0, 2.3, w=0.35, h=1.2, color=LILAC, kind="glow")
    out += banner((0, 9.2, 2.35), w=0.9, h=2.2, pole=False)
    return out


PROPS = [
    Prop("keep_dark_elves", keep_dark_elves(), "Dark Elf keep: hexagonal obsidian keep, needle spires, lilac lights"),
    Prop("fighting_pits", fighting_pits(), "Fighting pits: a sand ring behind a spiked black wall"),
    Prop("shooting_range", shooting_range(), "Shooting range: black hall, spiked fence, targets"),
    Prop("scout_post", scout_post(), "Scout post: a black hut on stilts with a lantern"),
    Prop("spider_caves", spider_caves(), "Spider caves: a rocky mound, a webbed cave mouth"),
    Prop("dark_church", dark_church(), "Dark church: steep black nave, needle bell tower, lilac rose window"),
    Prop("dark_dragon_lair", dark_dragon_lair(), "Dark dragon lair: a huge cave among obsidian shards and bones"),
    Prop("alchemy_workshop", alchemy_workshop(), "Alchemy workshop: black workshop, glowing vats, chimneys"),
    Prop("spider_farm", spider_farm(), "Spider farm: a web roof on posts over egg sacs"),
    Prop("crossbow_tower", crossbow_tower(), "Crossbow tower (Dark Elves, Alkars): dark spiked tower"),
]
