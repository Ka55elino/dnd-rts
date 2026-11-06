"""Empire (and the shared base pieces): steel, blue and gold, stone walls and towers, banners.
Also the siege machines (ballista, trebuchet) with their moving parts."""
from art.props._style import *  # noqa: F401,F403

STONE = (0.63, 0.61, 0.57)
STONE_D = (0.49, 0.47, 0.44)
STONE_L = (0.72, 0.70, 0.66)
ROOF = (0.22, 0.32, 0.56)
SLATE = (0.30, 0.31, 0.35)


# ------------------------------------------------------------------------------------ machines
def ballista_parts(prefix="ballista", wheels=True):
    """A ballista on a wheeled carriage (≈2.5 m), the bow and rail pointing +z. Moving parts: wheels
    (<prefix>_wheel_*) and the bow (<prefix>_crossbow, kicks back when it shoots)."""
    out = box((0, 0.35, 0), (1.5, 0.25, 2.2), color=WOOD_D)                       # carriage
    out += box((0, 0.6, -0.2), (0.4, 0.55, 0.4), color=WOOD)                       # mount
    if wheels:
        for i, (x, z) in enumerate([(-0.85, 0.75), (0.85, 0.75), (-0.85, -0.75), (0.85, -0.75)]):
            out += rod((x - 0.08, 0.42, z), (x + 0.08, 0.42, z), 0.42, seg=10, color=WOOD,
                       node=f"{prefix}_wheel_{i}", pivot=(x, 0.42, z))
    bow = box((0, 1.15, 0.2), (0.3, 0.22, 2.4), color=WOOD)                         # rail
    bow += rod((0, 1.3, 1.1), (-1.2, 1.3, 0.75), 0.07, seg=5, color=WOOD_D)         # bow arms
    bow += rod((0, 1.3, 1.1), (1.2, 1.3, 0.75), 0.07, seg=5, color=WOOD_D)
    bow += rod((-1.2, 1.3, 0.75), (0, 1.3, -0.5), 0.02, seg=3, color=BONE, detail=1)  # string
    bow += rod((1.2, 1.3, 0.75), (0, 1.3, -0.5), 0.02, seg=3, color=BONE, detail=1)
    bow += spike((0, 1.32, 0.2), (0, 1.32, 1.6), 0.06, color=IRON, kind="metal", detail=1)  # bolt
    bow += box((0, 1.0, -0.9), (0.7, 0.25, 0.3), kind="team", detail=1)
    return out + tagged(bow, node=f"{prefix}_crossbow", pivot=(0, 1.15, 0.2))


def trebuchet_parts(prefix="trebuchet", wheels=True):
    """A counterweight trebuchet (≈3.3 × 6.7 m, 7.5 m with the arm up). The arm swings about its
    axle (<prefix>_arm); the counterweight and the sling follow it; wheels roll."""
    import math
    out = box((0, 0.5, 0), (2.6, 0.35, 6.2), color=WOOD_D)                         # base frame
    for x in (-1.15, 1.15):
        out += box((x, 0.5, 0), (0.3, 0.35, 6.4), color=WOOD)                      # side beams
        out += rod((x, 0.8, -1.8), (x, 4.6, 0), 0.16, seg=5, color=WOOD)           # A-frame
        out += rod((x, 0.8, 1.8), (x, 4.6, 0), 0.16, seg=5, color=WOOD)
        out += rod((x, 0.8, 0), (x, 4.6, 0), 0.12, seg=5, color=WOOD_D, detail=1)
    out += rod((-1.35, 4.6, 0), (1.35, 4.6, 0), 0.12, seg=6, color=IRON, kind="metal")  # axle
    out += box((0, 0.85, -2.6), (1.6, 0.4, 0.6), kind="team", detail=1)
    if wheels:
        for i, (x, z) in enumerate([(-1.45, 2.3), (1.45, 2.3), (-1.45, -2.3), (1.45, -2.3)]):
            out += rod((x - 0.1, 0.55, z), (x + 0.1, 0.55, z), 0.55, seg=10, color=WOOD,
                       node=f"{prefix}_wheel_{i}", pivot=(x, 0.55, z))
    # Arm at rest: the long end down and back (it swings up and over the front when it shoots).
    pivot = (0.0, 4.6, 0.0)
    a = -1.9  # angle from +z towards +y
    d = (0.0, math.sin(a), math.cos(a))
    long_end = tuple(pivot[i] + 4.4 * d[i] for i in range(3))
    short_end = tuple(pivot[i] - 1.4 * d[i] for i in range(3))
    arm = rod(short_end, long_end, 0.13, r_b=0.08, seg=6, color=WOOD)
    out += tagged(arm, node=f"{prefix}_arm", pivot=pivot)
    cw = box((short_end[0], short_end[1] - 1.5, short_end[2]), (1.3, 1.1, 1.1), color=STONE_D)
    cw += rod(short_end, (short_end[0], short_end[1] - 0.4, short_end[2]), 0.08, seg=4, color=IRON, kind="metal")
    out += tagged(cw, node=f"{prefix}_counterweight", pivot=short_end)
    sling = rod(long_end, (long_end[0], long_end[1] - 0.15, long_end[2] - 0.9), 0.03, seg=3, color=BONE, detail=1)
    sling += sphere((long_end[0], long_end[1] - 0.25, long_end[2] - 1.0), 0.28, seg=6, rings=3, color=STONE_D)
    out += tagged(sling, node=f"{prefix}_sling", pivot=long_end)
    return out


# ----------------------------------------------------------------------------------- base walls
def gatehouse():
    """The gate segment of a base wall (stretched by the game over the gate's footprint, 18.6 × 7):
    two towers, the walkway block over a 5 m passage, leaves (gate_left / gate_right) and a
    portcullis that rises."""
    out = []
    for s in (-1, 1):
        x = s * 5.5
        out += box((x, 0, 0), (6.0, 8.5, 7.0), color=STONE)
        out += box((x, 0, 0.0), (6.4, 1.0, 7.4), color=STONE_D)
        out += battlements((x, 8.5, 0), 6.0, 7.0, color=STONE_L)
        out += windows([x], 5.2, 3.56, w=0.4, h=1.2)
        out += banner((x, 7.6, 3.56), w=1.4, h=2.8)
    out += box((0, 5.6, 0), (5.2, 3.0, 7.0), color=STONE)                          # over the passage
    out += box((0, 5.3, 3.3), (5.2, 0.5, 0.5), color=STONE_L)                      # lintel
    out += battlements((0, 8.6, 0), 5.2, 7.0, sides="ns", color=STONE_L)
    for s, name in ((-1, "gate_left"), (1, "gate_right")):
        hinge = (s * 2.5, 0.0, 0.2)
        leaf = box((s * 1.25, 0, 0.2), (2.45, 5.0, 0.3), color=DOOR)
        for y in (1.0, 3.8):
            leaf += box((s * 1.25, y, 0.37), (2.3, 0.18, 0.06), color=IRON, kind="metal", detail=1)
        out += tagged(leaf, node=name, pivot=hinge)
    grid = []
    for k in range(6):
        x = -2.2 + k * 0.88
        grid += box((x, 0, 1.5), (0.12, 5.0, 0.12), color=IRON, kind="metal")
    for y in (0.8, 2.2, 3.6, 4.8):
        grid += box((0, y, 1.5), (4.8, 0.12, 0.12), color=IRON, kind="metal")
    out += tagged(grid, node="portcullis", pivot=(0, 0, 1.5))
    return out


def keep_empire():
    out = box((0, 0, 0), (13.2, 0.8, 13.2), color=STONE_D)
    out += hall(9.4, 9.4, 12, STONE, ROOF, roof="none", at=(0, 0.8, 0))
    out += battlements((0, 12.8, 0), 9.4, 9.4, color=STONE_L)
    out += box((0, 12.8, -0.5), (5.2, 6.0, 5.2), color=STONE_L)                    # central tower
    out += hip((0, 18.8, -0.5), 6.0, 6.0, 4.2, color=ROOF)
    out += flag((0, 23.0, -0.5), h=2.5)
    for x in (-5.0, 5.0):
        for z in (-5.0, 5.0):
            out += round_tower(x, z, 1.8, 15.0, STONE, ROOF, roof_h=4.0, merlons=STONE_L, y=0.8)
    out += door(0, 4.75, w=2.6, h=3.2, y=0.8)
    out += box((0, 4.9, 4.8), (3.6, 0.35, 0.3), color=GOLD, kind="metal", detail=1)
    for x in (-2.8, 2.8):
        out += banner((x, 11.2, 4.78), w=1.5, h=4.0)
    out += windows([-2.8, 0, 2.8], 7.0, 4.75, w=0.6, h=1.3)
    out += windows([-2.8, 2.8], 7.0, -4.75, w=0.6, h=1.3)
    out += windows([0], 15.0, 2.15, w=0.7, h=1.4)
    return out


# ------------------------------------------------------------------------------------- towers
def tower_base(w=5.0, h=10.0):
    out = box((0, 0, 0), (w + 0.6, 1.0, w + 0.6), color=STONE_D)
    out += box((0, 0, 0), (w, h, w), color=STONE)
    out += box((0, h - 0.4, 0), (w + 0.5, 0.4, w + 0.5), color=STONE_L)
    out += battlements((0, h, 0), w + 0.5, w + 0.5, size=0.55, gap=0.55, color=STONE_L)
    out += windows([0], h * 0.55, w / 2, w=0.35, h=1.1)
    out += banner((0, h - 1.0, w / 2 + 0.02), w=1.1, h=2.4)
    return out


def archer_tower():
    out = tower_base(4.8, 11.0)
    for x in (-2.2, 2.2):
        for z in (-2.2, 2.2):
            out += rod((x, 11.0, z), (x, 13.0, z), 0.12, seg=4, color=WOOD_D, detail=1)
    out += hip((0, 13.0, 0), 5.8, 5.8, 2.2, color=ROOF)
    return out


def ballista_tower():
    out = tower_base(5.4, 9.5)
    out += move(scaled(ballista_parts("tb", wheels=False), 0.9), 0, 9.2, 0)
    return tagged(out, node=None)


def trebuchet_tower():
    out = round_tower(0, 0, 3.2, 8.0, STONE, merlons=STONE_L, seg=12)
    out += cyl((0, 0, 0), 3.5, 1.0, seg=12, color=STONE_D)
    out += banner((0, 7.0, 3.22), w=1.2, h=2.4)
    out += move(scaled(trebuchet_parts("tt", wheels=False), 0.55), 0, 8.0, 0)
    return tagged(out, node=None)


def watchtower():
    """A wooden watchtower: four posts, a platform at 7.6 m (the garrison stands on it), a ladder."""
    out = []
    for x in (-1.9, 1.9):
        for z in (-2.1, 2.1):
            out += rod((x * 1.08, 0, z * 1.08), (x, 8.7, z), 0.18, seg=5, color=WOOD)
    for y in (2.4, 5.0):
        out += rod((-1.95, y, 2.1), (1.95, y + 1.4, 2.1), 0.08, seg=4, color=WOOD_D, detail=1)
        out += rod((-1.95, y, -2.1), (1.95, y + 1.4, -2.1), 0.08, seg=4, color=WOOD_D, detail=1)
        out += rod((1.9, y, -2.1), (1.9, y + 1.4, 2.1), 0.08, seg=4, color=WOOD_D, detail=1)
        out += rod((-1.9, y, -2.1), (-1.9, y + 1.4, 2.1), 0.08, seg=4, color=WOOD_D, detail=1)
    out += box((0, 7.2, 0), (4.5, 0.4, 5.0), color=WOOD_D)
    for (a, b) in [((-2.2, -2.45), (2.2, -2.45)), ((2.2, -2.45), (2.2, 2.45)), ((2.2, 2.45), (-2.2, 2.45)), ((-2.2, 2.45), (-2.2, -2.45))]:
        out += rod((a[0], 8.6, a[1]), (b[0], 8.6, b[1]), 0.07, seg=4, color=WOOD)
    out += rod((-0.4, 0, -3.2), (-0.4, 7.4, -2.4), 0.06, seg=4, color=WOOD, detail=1)   # ladder
    out += rod((0.4, 0, -3.2), (0.4, 7.4, -2.4), 0.06, seg=4, color=WOOD, detail=1)
    out += banner((0, 8.5, 2.52), w=1.0, h=1.6, pole=False)
    return out


# ---------------------------------------------------------------------------------- buildings
def barracks():
    out = hall(8.5, 5.5, 4.2, STONE, ROOF, at=(0, 0, -1.8))
    out += box((0, 0, -1.8), (9.0, 0.7, 6.0), color=STONE_D)
    out += box((3.4, 0, -1.8), (2.2, 6.5, 2.2), color=STONE_L)                     # stair tower
    out += hip((3.4, 6.5, -1.8), 2.8, 2.8, 2.0, color=ROOF)
    out += door(-1.0, 1.0, w=1.8, h=2.2)
    out += windows([-3.2, 1.3], 2.0, 1.0)
    out += banner((-3.2, 4.0, 1.02), w=1.0, h=1.8)
    out += fence([(-4.6, 4.6), (4.6, 4.6)])
    for x in (2.0, 3.0):                                                            # weapon rack
        out += rod((x, 0, 3.0), (x, 1.8, 3.0), 0.05, seg=4, color=WOOD_D, detail=2)
        out += spike((x, 1.8, 3.0), (x, 2.3, 3.0), 0.09, color=IRON, kind="metal", detail=2)
    out += rod((1.6, 1.2, 3.0), (3.4, 1.2, 3.0), 0.05, seg=4, color=WOOD_D, detail=2)
    out += rod((-2.5, 0, 3.0), (-2.5, 1.6, 3.0), 0.1, seg=4, color=WOOD_D, detail=1)  # training dummy
    out += sphere((-2.5, 1.85, 3.0), 0.25, seg=6, rings=3, color=STRAW, detail=1)
    out += rod((-3.0, 1.35, 3.0), (-2.0, 1.35, 3.0), 0.08, seg=4, color=STRAW, detail=2)
    return out


def archery_range():
    out = hall(8.0, 3.2, 3.4, STONE, ROOF, roof="hip", at=(0, 0, -3.0))
    out += door(0, -1.35, w=1.4, h=2.0)
    out += banner((-2.5, 3.0, -1.37), w=0.9, h=1.6)
    out += fence([(-4.6, -1.0), (-4.6, 4.6), (4.6, 4.6), (4.6, -1.0)])
    for x in (-2.8, 0.0, 2.8):
        out += target(x, 3.3, ry=180)
    out += box((3.3, 0, -0.3), (0.8, 1.0, 0.5), color=WOOD_D, detail=1)            # arrow crate
    return out


def stables():
    out = box((0, 0, -1.5), (10.0, 0.6, 5.6), color=STONE_D)
    out += hall(9.6, 5.2, 3.6, WOOD, (0.42, 0.27, 0.16), at=(0, 0.6, -1.5), roof_h=2.4)
    for k in range(4):
        x = -3.3 + k * 2.2
        out += box((x, 0.6, 1.12), (1.5, 2.0, 0.1), color=WOOD_D)
        out += box((x, 1.7, 1.18), (1.5, 0.1, 0.06), color=WOOD, detail=2)
    out += banner((0, 4.0, 1.12), w=1.0, h=1.4, pole=False)
    out += fence([(-5.2, 1.6), (-5.2, 4.8), (5.2, 4.8), (5.2, 1.6)])
    out += dome((3.6, 0, 3.5), 0.9, h=0.8, seg=8, rings=3, color=STRAW, detail=1)
    out += box((-3.8, 0, 3.4), (1.2, 0.5, 0.6), color=WOOD_D, detail=1)            # trough
    return out


def mage_guild():
    out = box((0, 0, 0), (12.4, 0.5, 12.4), color=STONE_D)
    out += round_tower(2.4, -2.4, 3.0, 13.0, STONE_L, (0.30, 0.22, 0.52), roof_h=6.5, seg=12, y=0.5)
    out += sphere((2.4, 21.0, -2.4), 0.8, seg=8, rings=4, color=(0.6, 0.5, 1.0), kind="glow")
    out += cyl((2.4, 13.2, -2.4), 3.15, 0.3, seg=12, color=GOLD, kind="metal", detail=1)
    out += hall(6.0, 7.0, 5.0, STONE, ROOF, roof="hip", at=(-2.6, 0.5, 1.6))
    out += door(-2.6, 5.12, w=2.0, h=2.8, y=0.5)
    out += windows([2.4], 7.0, 0.62, w=0.5, h=1.6, color=(0.55, 0.5, 1.0), kind="glow")
    out += windows([-4.5, -0.7], 2.4, 5.12, color=(0.55, 0.5, 1.0), kind="glow")
    out += banner((-0.2, 4.8, 5.12), w=0.9, h=1.8)
    out += cyl((3.8, 0.5, 3.8), 0.6, 1.0, seg=8, color=STONE_L, detail=1)          # pedestal and orb
    out += sphere((3.8, 1.9, 3.8), 0.4, seg=6, rings=3, color=(0.5, 0.6, 1.0), kind="glow", detail=1)
    return out


def forge():
    out = hall(7.5, 5.0, 4.0, STONE, SLATE, at=(0, 0, -1.8))
    out += box((2.8, 0, -3.0), (1.6, 8.5, 1.6), color=STONE_D)                     # chimney
    out += box((2.8, 8.5, -3.0), (1.9, 0.4, 1.9), color=DARK)
    out += box((-0.8, 0.0, 0.72), (2.6, 2.4, 0.1), color=DARK)                     # forge mouth
    out += box((-0.8, 0.0, 0.6), (2.0, 1.4, 0.1), color=FIRE, kind="glow")
    out += box((0, 3.2, 2.2), (6.0, 0.2, 2.6), color=WOOD_D)                       # awning
    for x in (-2.8, 2.8):
        out += rod((x, 0, 3.4), (x, 3.2, 3.4), 0.1, seg=4, color=WOOD_D)
    out += anvil(0.6, 2.4)
    out += cyl((-2.6, 0, 2.5), 0.45, 0.7, seg=8, color=WOOD_D, detail=1)           # quench barrel
    out += banner((2.8, 7.0, -2.18), w=0.9, h=1.6, pole=False)
    return out


def workshop():
    out = box((0, 0, -2.8), (11.0, 5.5, 0.6), color=WOOD)                          # back wall
    for x in (-5.3, 5.3):
        out += box((x, 0, -0.5), (0.5, 5.5, 5.2), color=WOOD)
    for x in (-5.3, -1.8, 1.8, 5.3):
        out += rod((x, 0, 2.2), (x, 5.5, 2.2), 0.2, seg=5, color=WOOD_D)
    out += gable((0, 5.5, -0.4), 6.4, 11.8, 3.0, ry=90, color=(0.40, 0.26, 0.15))
    out += box((0, 0, -0.4), (11.0, 0.15, 5.6), color=DIRT)
    out += move(ballista_parts("ws", wheels=False), -2.6, 0, 0.0)                  # a machine being built
    out += rod((3.8, 0, 4.6), (3.8, 6.8, 4.6), 0.18, seg=5, color=WOOD_D)          # crane
    out += rod((3.8, 6.5, 4.6), (1.0, 6.5, 6.0), 0.12, seg=5, color=WOOD_D)
    out += rod((1.0, 6.5, 6.0), (1.0, 3.5, 6.0), 0.03, seg=3, color=BONE, detail=1)
    out += logs(-3.6, 5.6, n=3, length=3.0)
    out += rod((2.0, 0.7, -1.6), (2.2, 0.7, -1.6), 0.7, seg=10, color=WOOD, detail=1)  # spare wheel
    out += banner((0, 5.2, -2.48), w=1.2, h=2.0, pole=False)
    return out


def farm():
    out = hall(4.2, 3.2, 2.6, (0.72, 0.66, 0.55), STRAW, at=(-1.7, 0, -1.6), roof_h=2.0)
    out += door(-1.7, 0.02, w=1.0, h=1.6)
    out += box((-0.2, 0, -2.6), (0.6, 4.6, 0.6), color=STONE_D, detail=1)          # chimney
    for k, col in enumerate([(0.70, 0.62, 0.30), (0.45, 0.58, 0.25), (0.72, 0.64, 0.32)]):
        out += box((2.3, 0, -2.4 + k * 2.2), (3.0, 0.3, 1.8), color=col)
    out += box((-1.6, 0, 2.3), (4.0, 0.25, 1.8), color=(0.45, 0.58, 0.25))
    out += fence([(-3.8, 3.4), (3.8, 3.4), (3.8, -3.4)])
    out += dome((-3.2, 0, 1.0), 0.7, seg=7, rings=3, color=STRAW, detail=1)
    out += banner((-1.7, 2.4, 0.02), w=0.6, h=0.9, pole=False)
    return out


def castle_wall():
    return wall_row(stone=STONE, cap=STONE_L)


PROPS = [
    Prop("ballista", ballista_parts(), "Ballista: wheeled carriage, bow on a rail"),
    Prop("trebuchet", trebuchet_parts(), "Trebuchet: A-frames, swinging arm, counterweight, sling"),
    Prop("castle_wall", castle_wall(), "Base wall row (two back to back per segment), outer face +z"),
    Prop("gatehouse", gatehouse(), "Base gate: two towers, passage, leaves, portcullis"),
    Prop("keep", keep_empire(), "Empire keep: square keep, four round towers with blue roofs, banners"),
    Prop("archer_tower", archer_tower(), "Empire archer tower: square stone tower, roof on posts"),
    Prop("ballista_tower", ballista_tower(), "Empire ballista tower: stone tower with a ballista on top"),
    Prop("trebuchet_tower", trebuchet_tower(), "Empire trebuchet tower: round platform with a trebuchet"),
    Prop("watchtower", watchtower(), "Wooden watchtower with a platform at 7.6 m"),
    Prop("barracks", barracks(), "Empire barracks: stone hall, stair tower, training yard"),
    Prop("archery_range", archery_range(), "Empire archery range: hall and targets"),
    Prop("stables", stables(), "Empire stables: timber barn, stalls, paddock"),
    Prop("mage_guild", mage_guild(), "Empire mage guild: round tower with a glowing orb, annex"),
    Prop("forge", forge(), "Empire forge: stone smithy, chimney, glowing hearth, anvil"),
    Prop("workshop", workshop(), "Empire workshop: open timber shed, a ballista being built, crane"),
    Prop("farm", farm(), "Farm: thatched cottage, fields"),
]
