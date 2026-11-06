"""Kit for stand-in models: plain shapes, a humanoid body and weapons on the humanoid skeleton.

Rest pose (A-pose, authoring axes Z up, front -Y, left +X): the right hand's grip is at G, the left
hand's at L. A weapon's business end (blade, spear tip, staff head, hammer head) points along -Z from
the grip — the stances aim it with `sword_aim`.
"""
import math
from modelc import Loft, Extrude

G = (-0.668, -0.02, 0.975)   # right-hand grip
L = (0.625, 0.0, 1.02)       # left-hand grip

SKIN = (0.80, 0.64, 0.52)
STEEL = (0.70, 0.72, 0.76)
DARKSTEEL = (0.38, 0.40, 0.44)
CLOTH = (0.30, 0.26, 0.22)
LEATHER = (0.32, 0.21, 0.12)
WOOD = (0.45, 0.32, 0.18)
GOLD = (0.82, 0.66, 0.28)
WHITE = (0.88, 0.86, 0.80)


# --- shapes ---------------------------------------------------------------------------------------
def cyl(id, a, b, r, seg=6, **kw):
    """A cylinder from point a to point b (closed ends); r may be a curve of t."""
    return Loft(id, path=[a, b], r=r, segments=seg, rings=2, **kw)


def cone(id, a, b, r0, r1=0.004, seg=6, **kw):
    """A cone (or a frustum) from a (radius r0) to b (radius r1)."""
    return Loft(id, path=[a, b], r=lambda t: r0 + (r1 - r0) * t, segments=seg, rings=2, **kw)


def box(id, a, b, rx, ry, **kw):
    """A box from a to b with half sizes rx (across) and ry (the other way across)."""
    return Loft(id, path=[a, b], rx=rx, ry=ry, n=8, segments=4, rings=2, **kw)


def ball(id, c, r, ry=None, **kw):
    """A low-poly sphere (or an ellipsoid with ry) of radius r centred at c."""
    x, y, z = c
    prof = lambda t: math.sin(math.pi * t) ** 0.7 + 0.02
    return Loft(id, path=[(x, y, z - r), (x, y, z + r)], rx=lambda t: r * prof(t), ry=lambda t: (ry or r) * prof(t),
                segments=6, rings=4, **kw)


def off(p, dx=0.0, dy=0.0, dz=0.0):
    return (p[0] + dx, p[1] + dy, p[2] + dz)


# --- body -----------------------------------------------------------------------------------------
def humanoid(torso=STEEL, legs=CLOTH, shins=None, arms=None, forearms=CLOTH, hands=LEATHER, feet=LEATHER,
             head="helmet", head_color=STEEL, tabard=True, robe=None, sash=False, build=1.0, skin=SKIN):
    """A body of plain shapes. head: "helmet" | "horned" | "hood" | "hat" | "bare" | "crown". tabard: a
    team-colored tabard over the torso; robe: a long robe (color) to the ankles instead of trousers;
    sash: a team-colored sash at the waist (robed units). build: torso width factor (heavy armour > 1)."""
    arms = arms or torso
    shins = shins or legs
    w = 0.16 * build
    parts = [
        box("torso", (0, 0, 0.95), (0, 0, 1.52), w, 0.10 * build, bones=["hips", "spine", "chest"], color=torso),
        ball("head", (0, -0.01, 1.70), 0.10, bone="head", color=skin),
        cyl("arm", (0.17, 0, 1.46), (0.40, 0, 1.24), 0.05 * build, bone="upperarm.L", color=arms, mirror="x"),
        cyl("forearm", (0.40, 0, 1.24), (0.58, 0, 1.06), 0.045, bone="forearm.L", color=forearms, mirror="x"),
        ball("hand", (0.61, 0, 1.03), 0.042, bone="hand.L", color=hands, mirror="x"),
        box("foot", (0.115, 0.05, 0.035), (0.12, -0.16, 0.035), 0.05, 0.035, bone="foot.L", color=feet, mirror="x"),
    ]
    if robe:
        parts.append(Loft("robe", path=[(0, 0, 1.08), (0, -0.01, 0.60), (0, -0.02, 0.10)],
                          rx=lambda t: 0.17 + 0.13 * t, ry=lambda t: 0.12 + 0.12 * t, segments=8, rings=3,
                          caps=(False, False), bones=["hips", "thigh.L", "thigh.R"], color=robe))
        parts.append(cyl("shin", (0.11, 0, 0.52), (0.115, 0.01, 0.09), 0.05, bone="shin.L", color=robe, mirror="x"))
    else:
        parts += [
            cyl("thigh", (0.10, 0, 0.96), (0.11, 0, 0.52), 0.075, bone="thigh.L", color=legs, mirror="x"),
            cyl("shin", (0.11, 0, 0.52), (0.115, 0.01, 0.09), 0.06, bone="shin.L", color=shins, mirror="x"),
        ]
    if tabard:
        parts.append(box("tabard", (0, -0.005, 0.70), (0, -0.005, 1.25), w + 0.006, 0.106 * build,
                         bones=["hips", "spine"], team=True))
    if sash:
        parts.append(box("sash", (0, 0, 1.00), (0, 0, 1.10), 0.18, 0.13, bone="hips", team=True))
    if head == "helmet":
        parts.append(ball("helmet", (0, -0.005, 1.75), 0.115, bone="head", color=head_color))
    elif head == "horned":
        parts.append(ball("helmet", (0, -0.005, 1.75), 0.115, bone="head", color=head_color))
        parts.append(cone("horn", (0.09, 0, 1.80), (0.22, 0.02, 1.98), 0.03, bone="head", color=WHITE, mirror="x"))
    elif head == "hood":
        parts.append(ball("hood", (0, 0.01, 1.73), 0.125, bone="head", color=head_color))
    elif head == "hat":
        parts.append(cyl("brim", (0, 0, 1.77), (0, 0, 1.79), 0.20, seg=8, bone="head", color=head_color))
        parts.append(cone("hat", (0, 0, 1.78), (0, 0.06, 2.12), 0.11, bone="head", color=head_color))
    elif head == "crown":
        parts.append(cyl("crown", (0, -0.01, 1.76), (0, -0.01, 1.82), 0.10, seg=8, bone="head", color=GOLD))
    return parts


# --- weapons (right hand at G, business end along -Z) ----------------------------------------------
def sword(length=0.78):
    return [
        box("blade", off(G, dz=-0.07), off(G, dz=-0.07 - length), 0.025, 0.006, bone="hand.R", color=STEEL),
        box("guard", off(G, dx=-0.09, dz=-0.06), off(G, dx=0.09, dz=-0.06), 0.014, 0.014, bone="hand.R", color=STEEL),
        cyl("grip", off(G, dz=-0.06), off(G, dz=0.08), 0.016, bone="hand.R", color=LEATHER),
    ]


def spear(length=2.2):
    return [
        cyl("shaft", off(G, dz=0.55), off(G, dz=0.55 - length), 0.018, bone="hand.R", color=WOOD),
        cone("spearhead", off(G, dz=0.55 - length), off(G, dz=0.30 - length), 0.035, seg=4, bone="hand.R", color=STEEL),
    ]


def staff(head_color, length=1.75, head_r=0.065):
    return [
        cyl("staff", off(G, dz=0.55), off(G, dz=0.55 - length), 0.02, bone="hand.R", color=WOOD),
        ball("orb", off(G, dz=0.55 - length - head_r), head_r, bone="hand.R", color=head_color),
    ]


def hammer():
    return [
        cyl("handle", off(G, dz=0.08), off(G, dz=-0.45), 0.017, bone="hand.R", color=WOOD),
        box("hammerhead", off(G, dx=-0.08, dz=-0.47), off(G, dx=0.08, dz=-0.47), 0.04, 0.04, bone="hand.R", color=DARKSTEEL),
    ]


def mace():
    return [
        cyl("handle", off(G, dz=0.08), off(G, dz=-0.48), 0.018, bone="hand.R", color=WOOD),
        ball("macehead", off(G, dz=-0.55), 0.075, bone="hand.R", color=STEEL),
    ]


def shield(color=None):
    """A heater shield on the left hand, its face team-colored (or `color`)."""
    kw = {"color": color} if color else {"team": True}
    return [Extrude("shield", outline=[(-0.24, 0.36), (0.24, 0.36), (0.24, -0.10), (0.0, -0.40), (-0.24, -0.10)],
                    depth=0.04, origin=(0.50, -0.09, 1.10), bone="hand.L", **kw)]


def bow():
    """A bow in the left hand (limbs bending back toward the archer) and a quiver on the back."""
    x, y, z = L
    return [
        Loft("bow", path=[(x, 0.10, z + 0.62), (x, -0.04, z + 0.30), (x, -0.06, z), (x, -0.04, z - 0.30), (x, 0.10, z - 0.62)],
             r=lambda t: 0.012 + 0.01 * math.sin(math.pi * t), segments=4, rings=6, bone="hand.L", color=WOOD),
        cyl("string", (x, 0.10, z + 0.62), (x, 0.10, z - 0.62), 0.003, seg=3, bone="hand.L", color=WHITE),
        cyl("quiver", (0.08, 0.14, 1.05), (-0.08, 0.16, 1.58), 0.045, bone="chest", color=LEATHER),
    ]


def lance(length=2.9):
    return [
        cone("lance", off(G, dz=0.6), off(G, dz=0.6 - length), 0.035, 0.008, bone="hand.R", color=WOOD),
        cone("vamplate", off(G, dz=-0.10), off(G, dz=0.08), 0.07, 0.02, bone="hand.R", color=STEEL),
    ]


# --- cavalry ----------------------------------------------------------------------------------------
def shift(parts, d):
    """Move parts made for the standing humanoid onto the saddle (art/skeletons/cavalry.py RIDER)."""
    import numpy as np
    for p in parts:
        if isinstance(p, Loft):
            p.path = [(x + d[0], y + d[1], z + d[2]) for x, y, z in p.path]
        else:
            p.origin = p.origin + np.array(d, dtype=float)
    return parts


def horse(coat=(0.40, 0.26, 0.16), mane=(0.12, 0.09, 0.07), barding=True, scale=1.0, beak=None):
    """A horse of plain shapes on the cavalry skeleton (scale: the mount skeleton's size — a wolf, a
    beast); barding: a team-colored caparison; beak: a griffon's beak color."""
    HOOF = (0.12, 0.11, 0.10)
    parts = [
        box("hbody", (0, 0.66, 1.20), (0, -0.62, 1.24), 0.25, 0.30, bone="body", color=coat),
        cyl("hneck", (0, -0.45, 1.30), (0, -0.80, 1.76), 0.14, bone="neck_h", color=coat),
        box("mane", (0, -0.40, 1.40), (0, -0.74, 1.86), 0.03, 0.05, bone="neck_h", color=mane),
        box("hhead", (0, -0.74, 1.80), (0, -1.14, 1.56), 0.08, 0.11, bone="head_h", color=coat),
        cone("ear", (0.04, -0.80, 1.86), (0.05, -0.79, 1.96), 0.022, bone="head_h", color=coat, mirror="x"),
        cyl("htail", (0, 0.58, 1.32), (0, 0.76, 0.92), 0.04, bone="tail", color=mane),
        cyl("fleg", (0.14, -0.42, 1.04), (0.14, -0.44, 0.58), 0.085, bone="fleg.L", color=coat, mirror="x"),
        cyl("fshin", (0.14, -0.44, 0.58), (0.14, -0.44, 0.08), 0.052, bone="fshin.L", color=coat, mirror="x"),
        box("fhoof", (0.14, -0.43, 0.0), (0.14, -0.45, 0.09), 0.05, 0.06, bone="fshin.L", color=HOOF, mirror="x"),
        cyl("hleg", (0.14, 0.40, 1.08), (0.14, 0.48, 0.60), 0.10, bone="hleg.L", color=coat, mirror="x"),
        cyl("hshin", (0.14, 0.48, 0.60), (0.14, 0.44, 0.08), 0.052, bone="hshin.L", color=coat, mirror="x"),
        box("hhoof", (0.14, 0.43, 0.0), (0.14, 0.45, 0.09), 0.05, 0.06, bone="hshin.L", color=HOOF, mirror="x"),
        box("saddle", (0, 0.20, 1.50), (0, -0.12, 1.52), 0.18, 0.03, bone="body", color=LEATHER),
    ]
    if barding:
        parts.append(box("barding", (0, 0.68, 1.02), (0, -0.64, 1.04), 0.26, 0.24, bone="body", team=True))
    if beak:
        parts.append(cone("beak", (0, -1.08, 1.60), (0, -1.24, 1.52), 0.06, seg=4, bone="head_h", color=beak))
    return scaled(parts, scale) if scale != 1.0 else parts


def axe(color=DARKSTEEL):
    return [
        cyl("haft", off(G, dz=0.10), off(G, dz=-0.55), 0.017, bone="hand.R", color=WOOD),
        box("axehead", off(G, dx=0.0, dz=-0.50), off(G, dx=0.13, dz=-0.50), 0.010, 0.07, bone="hand.R", color=color),
    ]


def scaled(parts, s, about=(0.0, 0.0, 0.0)):
    """Scale parts (positions and thickness) by s around a point: a mount of another size."""
    import numpy as np
    ax, ay, az = about
    for p in parts:
        if isinstance(p, Loft):
            p.path = [(ax + (x - ax) * s, ay + (y - ay) * s, az + (z - az) * s) for x, y, z in p.path]
            fx, fy = p.rx, p.ry
            p.rx = lambda t, f=fx: f(t) * s
            p.ry = lambda t, f=fy: f(t) * s
        else:
            p.outline = [(u * s, v * s) for u, v in p.outline]
            p.depth *= s
            p.origin = np.array(about) + (p.origin - np.array(about)) * s
    return parts


def mount_rider(parts, sk):
    """Humanoid parts (+ weapon) moved onto the saddle of a mount skeleton (art/skeletons/cavalry.py)."""
    return shift(parts, sk.rider)


def wings(color, span=0.95):
    """Flat wings on the wing bones of a mount (griffon)."""
    return [Extrude("wing", outline=[(0.0, 0.0), (span * 0.45, 0.10), (span, 0.06), (span * 0.9, -0.12), (span * 0.6, -0.22),
                                     (span * 0.3, -0.26), (0.0, -0.20)],
                    depth=0.02, origin=(0.20, -0.20, 1.44), u_axis=(1, 0, 0), v_axis=(0, -1, 0),
                    bone="wing.L", color=color, mirror="x")]


def arm_wings(color):
    """Harpy wings along the arms (upper arm + forearm), feathers hanging back."""
    return [
        Extrude("wing", outline=[(0.0, 0.0), (0.30, -0.22), (0.30, -0.40), (0.0, -0.20)],
                depth=0.02, origin=(0.17, 0.02, 1.46), u_axis=(0.707, 0, -0.707), v_axis=(0, 0.6, -0.8),
                bone="upperarm.L", color=color, mirror="x"),
        Extrude("wingtip", outline=[(0.0, 0.0), (0.28, -0.05), (0.40, -0.30), (0.0, -0.40)],
                depth=0.02, origin=(0.40, 0.02, 1.24), u_axis=(0.707, 0, -0.707), v_axis=(0, 0.6, -0.8),
                bone="forearm.L", color=color, mirror="x"),
    ]


def elemental(core, glow, shards=False):
    """An elemental of plain shapes on the humanoid skeleton: a tapering body without legs (a column
    of fire / ice / storm down to the ground), big shoulders and fists."""
    return [
        cone("column", (0, 0, 1.15), (0, 0, 0.05), 0.20, 0.06, seg=7, bones=["hips", "thigh.L", "thigh.R"], color=glow),
        box("core", (0, 0, 0.95), (0, 0, 1.55), 0.17, 0.12, bones=["hips", "spine", "chest"], color=core),
        ball("ehead", (0, -0.02, 1.72), 0.12, bone="head", color=glow),
        ball("shoulder", (0.22, 0, 1.47), 0.11, bone="upperarm.L", color=core, mirror="x"),
        cyl("earm", (0.22, 0, 1.40), (0.40, 0, 1.24), 0.06, bone="upperarm.L", color=glow, mirror="x"),
        cyl("eforearm", (0.40, 0, 1.24), (0.58, 0, 1.06), 0.07, bone="forearm.L", color=core, mirror="x"),
        ball("fist", (0.62, 0, 1.02), 0.09, bone="hand.L", color=glow, mirror="x"),
    ] + ([cone("shard", (0.06, 0.05, 1.55), (0.12, 0.10, 1.85), 0.05, seg=4, bone="chest", color=glow, mirror="x")]
         if shards else [])


# --- more weapons ---------------------------------------------------------------------------------
def dagger():
    return sword(0.35)


def left_sword(length=0.7):
    """A second sword in the left hand (dual blades), pointing down from the left grip."""
    x, y, z = L
    return [
        box("blade2", (x, y, z - 0.07), (x, y, z - 0.07 - length), 0.022, 0.006, bone="hand.L", color=STEEL),
        box("guard2", (x - 0.08, y, z - 0.06), (x + 0.08, y, z - 0.06), 0.012, 0.012, bone="hand.L", color=STEEL),
    ]


def gun(length=1.1, color=DARKSTEEL):
    """A long gun held in the right hand, the muzzle (business end) along -Z, the stock above the grip."""
    return [
        box("stock", off(G, dz=0.30), off(G, dz=-0.10), 0.025, 0.05, bone="hand.R", color=WOOD),
        cyl("barrel", off(G, dz=-0.10), off(G, dz=-0.10 - length * 0.7), 0.022, bone="hand.R", color=color),
    ]


def crossbow(left=False):
    """A crossbow: the stock along -Z from the grip, a bow across its front end."""
    if left:
        x, y, z = L
        # the left hand keeps its world orientation in the stances: author it pointing forward (-Y)
        return [box("cbstock2", (x, y + 0.05, z), (x, y - 0.50, z), 0.025, 0.03, bone="hand.L", color=WOOD),
                box("cbbow2", (x - 0.25, y - 0.45, z), (x + 0.25, y - 0.45, z), 0.012, 0.012, bone="hand.L", color=DARKSTEEL)]
    return [box("cbstock", off(G, dz=0.0), off(G, dz=-0.55), 0.025, 0.03, bone="hand.R", color=WOOD),
            box("cbbow", off(G, dx=-0.25, dz=-0.50), off(G, dx=0.25, dz=-0.50), 0.012, 0.012, bone="hand.R", color=DARKSTEEL)]


def club(color=WOOD, length=1.0, r=0.10):
    """A big club (trolls): a thick head at the business end."""
    return [cone("club", off(G, dz=0.10), off(G, dz=-length), 0.03, r, bone="hand.R", color=color)]


def greataxe(color=DARKSTEEL):
    """A two-handed axe (axe dwarves): a long haft, a wide blade."""
    return [
        cyl("haft", off(G, dz=0.35), off(G, dz=-0.75), 0.02, bone="hand.R", color=WOOD),
        box("axehead", off(G, dz=-0.70), off(G, dx=0.22, dz=-0.70), 0.012, 0.13, bone="hand.R", color=color),
    ]


# --- head extras ------------------------------------------------------------------------------------
def beard(color):
    return [cone("beard", (0, -0.08, 1.67), (0, -0.10, 1.40), 0.09, 0.02, bone="head", color=color)]


def ears(color=SKIN):
    return [cone("ear", (0.09, 0.0, 1.72), (0.17, 0.03, 1.80), 0.025, seg=4, bone="head", color=color, mirror="x")]


def antlers(color=WHITE):
    """Antlers on a mount's head (a stag)."""
    return [cone("antler", (0.05, -0.80, 1.92), (0.30, -0.70, 2.30), 0.03, bone="head_h", color=color, mirror="x"),
            cone("tine", (0.15, -0.76, 2.08), (0.20, -0.95, 2.25), 0.02, bone="head_h", color=color, mirror="x")]


def floating(robe, head_color=None, crown=False):
    """A robed shape without legs (wraiths): a column of cloth from the chest to the ground, a hood."""
    parts = [
        cone("shroud", (0, 0, 1.40), (0, 0, 0.10), 0.18, 0.28, seg=7, bones=["hips", "spine", "thigh.L", "thigh.R"], color=robe),
        box("torso", (0, 0, 1.05), (0, 0, 1.52), 0.15, 0.10, bones=["spine", "chest"], color=robe),
        ball("hood", (0, 0.0, 1.72), 0.12, bone="head", color=robe),
        ball("face", (0, -0.06, 1.70), 0.07, bone="head", color=head_color or (0.55, 0.85, 0.75)),
        cyl("arm", (0.17, 0, 1.46), (0.40, 0, 1.24), 0.05, bone="upperarm.L", color=robe, mirror="x"),
        cyl("forearm", (0.40, 0, 1.24), (0.58, 0, 1.06), 0.045, bone="forearm.L", color=robe, mirror="x"),
        ball("hand", (0.61, 0, 1.03), 0.04, bone="hand.L", color=head_color or (0.55, 0.85, 0.75), mirror="x"),
    ]
    if crown:
        parts.append(cyl("crown", (0, 0, 1.80), (0, 0, 1.86), 0.10, seg=8, bone="head", color=GOLD))
    return parts


# --- creatures on the mount skeleton ----------------------------------------------------------------
def spider_body(color, scale):
    """A spider on the mount skeleton: a round abdomen at the back, a small head, eight legs splayed out
    (two per leg bone)."""
    parts = [
        ball("abdomen", (0, 0.40, 1.25), 0.42, ry=0.36, bone="body", color=color),
        ball("thorax", (0, -0.15, 1.15), 0.26, bone="body", color=color),
        ball("shead", (0, -0.55, 1.12), 0.16, bone="neck_h", color=color),
    ]
    for k, (bone, shin, y0) in enumerate([("fleg.L", "fshin.L", -0.40), ("fleg.L", "fshin.L", -0.15),
                                          ("hleg.L", "hshin.L", 0.10), ("hleg.L", "hshin.L", 0.35)]):
        parts += [cyl(f"sleg{k}", (0.15, y0, 1.10), (0.55, y0 + 0.05 * (k - 1.5), 1.30), 0.04, bone=bone, color=color, mirror="x"),
                  cyl(f"sshin{k}", (0.55, y0 + 0.05 * (k - 1.5), 1.30), (0.75, y0 + 0.12 * (k - 1.5), 0.02), 0.03,
                      bone=shin, color=color, mirror="x")]
    return scaled(parts, scale)


def dragon_parts(color, belly):
    """A dragon on the mount skeleton (scaled up by the model): a long body and neck, horns, a tail,
    big wings."""
    return [
        box("dbody", (0, 0.70, 1.20), (0, -0.62, 1.26), 0.27, 0.30, bone="body", color=color),
        box("dbelly", (0, 0.55, 1.02), (0, -0.50, 1.04), 0.22, 0.16, bone="body", color=belly),
        cyl("dneck", (0, -0.45, 1.30), (0, -0.85, 1.85), 0.13, bone="neck_h", color=color),
        box("dhead", (0, -0.80, 1.88), (0, -1.25, 1.75), 0.09, 0.09, bone="head_h", color=color),
        cone("horn", (0.05, -0.85, 1.95), (0.12, -0.70, 2.15), 0.03, bone="head_h", color=belly, mirror="x"),
        cone("dtail", (0, 0.62, 1.30), (0, 1.60, 0.90), 0.12, 0.02, bone="tail", color=color),
        cyl("fleg", (0.14, -0.42, 1.04), (0.14, -0.44, 0.58), 0.08, bone="fleg.L", color=color, mirror="x"),
        cyl("fshin", (0.14, -0.44, 0.58), (0.14, -0.44, 0.06), 0.06, bone="fshin.L", color=color, mirror="x"),
        cyl("hleg", (0.14, 0.40, 1.08), (0.14, 0.48, 0.60), 0.10, bone="hleg.L", color=color, mirror="x"),
        cyl("hshin", (0.14, 0.48, 0.60), (0.14, 0.44, 0.06), 0.06, bone="hshin.L", color=color, mirror="x"),
    ] + wings(color, span=1.6)
