"""Empire swordsman, models-as-code version (docs/lore/images/empire/units/swordsman/central.png):
a heroic plate soldier — broad layered pauldrons over a narrow waist, a sculpted cuirass with a belt
and an eagle buckle, a Corinthian helmet with a tall red crest sweeping back, flared couters and
vambraces, knee cops over full greaves, armoured sabatons, red tabard strips, a long sword with a
curved guard and a large curved heater shield.

Not a game unit yet: it appears in Model Overview (id proto_knight).
Build:  python3 tools/modelc/modelc build proto_knight
"""
import math
from modelc import Model, Loft, Extrude, curve
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim

CLOTH = (0.12, 0.11, 0.12)     # dark under-tunic, sleeves, trousers
SKIN = (0.78, 0.62, 0.52)
STEEL = (0.74, 0.76, 0.80)
DARK = (0.20, 0.20, 0.23)      # dark steel of the shield field
LEATHER = (0.34, 0.21, 0.12)
GOLD = (0.78, 0.62, 0.28)
RED = (0.55, 0.06, 0.07)


def bump(t, a=0.0, b=1.0):
    """0 at a and b, 1 in the middle: a rounded profile between two points along a path."""
    if t <= a or t >= b:
        return 0.0
    return math.sin(math.pi * (t - a) / (b - a))


body = [
    # Under the armour only the sleeves (upper arm) and the trousers (thigh) show. The torso and the head
    # are minimal fillers: they only keep the openings of the cuirass and the helmet from being see-through.
    Loft("torso", path=[(0, 0.0, 0.95), (0, 0, 1.25), (0, 0, 1.56)],
         rx=curve([(0, 0.15), (0.5, 0.17), (1, 0.08)]),
         ry=curve([(0, 0.11), (0.5, 0.12), (1, 0.07)]),
         n=2.3, segments=6, rings=3, bones=["hips", "spine", "chest", "neck"], color=CLOTH, uv="torso", uv_scale=0.12),
    Loft("head", path=[(0, -0.005, 1.57), (0, -0.005, 1.82)], r=lambda t: 0.10 * math.sin(math.pi * (0.08 + 0.9 * t)) ** 0.6,
         segments=6, rings=3, bone="head", color=SKIN, uv_scale=0.08),
    # sleeve: shoulder to just inside the vambrace (the forearm is covered)
    Loft("arm", path=[(0.17, 0, 1.46), (0.40, 0, 1.24), (0.47, 0, 1.17)],
         r=curve([(0, 0.1), (0.7, 0.047), (1, 0.044)]),
         segments=8, rings=4, caps=(False, True), bones=["upperarm.L", "forearm.L"], color=CLOTH, mirror="x"),
    # trousers: hip to just inside the greave (the shin is covered)
    Loft("leg", path=[(0.10, 0, 0.98), (0.11, 0, 0.52), (0.112, 0.002, 0.44)],
         r=curve([(0, 0.095), (0.45, 0.083), (0.85, 0.064), (1, 0.058)]),
         segments=8, rings=4, caps=(False, True), bones=["thigh.L", "shin.L"], color=CLOTH, mirror="x"),
]

armor = [
    # --- head -------------------------------------------------------------------------------------
    # Corinthian helmet: a tall dome, deeper at the back; the cheek guards flare out at the bottom and
    # come forward to the chin (the T-shaped face opening is painted)
    Loft("helmet", path=[(0, -0.01, 1.53), (0, -0.005, 1.66), (0, 0.0, 1.80), (0, 0.005, 1.90)],
         rx=curve([(0, 0.12), (0.15, 0.116), (0.45, 0.126), (0.75, 0.114), (0.92, 0.07), (1, 0.02)]),
         ry=curve([(0, 0.150), (0.15, 0.142), (0.45, 0.140), (0.75, 0.128), (0.92, 0.08), (1, 0.025)]),
         offset=lambda t: (0, -0.03 * (1 - t) ** 2, 0),
         n=2.3, segments=12, rings=6, shell=0.008, shell_inner=False, caps=(False, True), bone="head", color=STEEL, material="metal"),
    # crest: a tall fan of horsehair from the forehead over the top, sweeping back behind the head
    Extrude("crest", outline=[(-0.12, 1.86), (-0.16, 1.98), (-0.13, 2.09), (-0.04, 2.17), (0.08, 2.19), (0.19, 2.15),
                              (0.28, 2.06), (0.33, 1.94), (0.33, 1.80), (0.26, 1.84), (0.18, 1.90), (0.08, 1.93),
                              (-0.03, 1.92)],
            depth=0.06, u_axis=(0, 1, 0), v_axis=(0, 0, 1), bone="head", color=RED),
    # gorget: a steel collar at the neck
    Loft("gorget", path=[(0, 0.0, 1.50), (0, 0.0, 1.58)], rx=curve([(0, 0.13), (1, 0.085)]), ry=curve([(0, 0.11), (1, 0.08)]),
         n=2.2, segments=10, rings=2, shell=0.008, shell_inner=False, caps=(False, False), bones=["chest", "neck"], color=STEEL, material="metal"),

    # --- torso ------------------------------------------------------------------------------------
    # sculpted cuirass: broad chest, narrow waist, the chest pushed forward, a little flare at the bottom
    Loft("cuirass", path=[(0, 0, 1.00), (0, 0, 1.12), (0, 0, 1.30), (0, 0, 1.44), (0, 0, 1.53)],
         rx=curve([(0, 0.168), (0.2, 0.158), (0.45, 0.19), (0.7, 0.225), (0.88, 0.215), (1, 0.15)]),
         ry=curve([(0, 0.128), (0.2, 0.122), (0.45, 0.14), (0.7, 0.152), (0.88, 0.14), (1, 0.105)]),
         offset=lambda t: (0, -0.022 * bump(t, 0.25, 1.0), 0),
         n=2.4, segments=12, rings=6, shell=0.008, shell_inner=False, caps=(False, False), bones=["spine", "chest"], color=STEEL, material="metal"),
    # belt with an eagle buckle
    Loft("belt", path=[(0, 0, 0.985), (0, 0, 1.055)], rx=0.176, ry=0.136, n=2.4, segments=12, rings=2,
         shell=0.008, shell_inner=False, caps=(False, False), bone="hips", color=LEATHER, material="leather"),
    Extrude("buckle", outline=[(math.cos(a) * 0.055 * (1.25 if k % 2 == 0 else 1.0), 0.02 + math.sin(a) * 0.045)
                               for k, a in enumerate([math.pi * 2 * i / 10 for i in range(10)])],
            depth=0.02, origin=(0, -0.142, 1.0), bone="hips", color=GOLD, material="gold"),
    # red tabard strips hanging in front between the legs, and at the back
    Extrude("strips", outline=[(-0.13, 0.03), (0.13, 0.03), (0.13, -0.30), (0.10, -0.34), (0.078, -0.31), (0.052, -0.37),
                               (0.026, -0.34), (0.0, -0.40), (-0.026, -0.34), (-0.052, -0.37), (-0.078, -0.31), (-0.10, -0.34), (-0.13, -0.30)],
            depth=0.012, origin=(0, 0, 0.99), wrap=(0.172, 0.138, 0.18), bone="hips", color=RED),
    Extrude("strips_back", outline=[(-0.13, 0.03), (0.13, 0.03), (0.13, -0.26), (0.0, -0.33), (-0.13, -0.26)],
            depth=0.012, origin=(0, 0, 0.99), wrap=(0.172, 0.142, 0.18, 180), bone="hips", color=RED, uv="strips"),
    # tassets: a steel plate hanging over the outside-front of each thigh
    Extrude("tasset", outline=[(-0.07, 0.0), (0.07, 0.0), (0.085, -0.16), (0.0, -0.19), (-0.085, -0.16)],
            depth=0.012, origin=(0, 0, 1.0), wrap=(0.19, 0.155, 0.35, 62), bone="thigh.L", color=STEEL, material="metal",
            mirror="x"),

    # --- arms -------------------------------------------------------------------------------------
    # pauldron: a rounded shoulder dome and two lames below it
    Loft("pauldron", path=[(0.15, 0, 1.527), (0.21, 0, 1.527), (0.26, 0, 1.495), (0.285, 0, 1.435)],
         rx=curve([(0, 0.03), (0.25, 0.058), (0.6, 0.062), (1, 0.05)]),
         ry=curve([(0, 0.037), (0.25, 0.068), (0.6, 0.07), (1, 0.06)]),
         n=2.2, segments=10, rings=4, bone="upperarm.L", color=STEEL, material="metal", mirror="x"),
    Loft("lame1", path=[(0.268, 0, 1.427), (0.28, 0, 1.402)], rx=curve([(0, 0.048), (1, 0.052)]), ry=curve([(0, 0.056), (1, 0.06)]),
         n=2.2, segments=8, rings=2, shell=0.008, caps=(False, False), bone="upperarm.L", color=STEEL, material="metal", mirror="x", uv="pauldron"),
    Loft("lame2", path=[(0.28, 0, 1.4), (0.29, 0, 1.377)], rx=curve([(0, 0.041), (1, 0.045)]), ry=curve([(0, 0.048), (1, 0.052)]),
         n=2.2, segments=8, rings=2, shell=0.008, caps=(False, False), bone="upperarm.L", color=STEEL, material="metal", mirror="x", uv="pauldron"),
    # couter: a cup over the elbow with a flared wing on the outside
    Loft("couter", path=[(0.375, 0.01, 1.265), (0.405, 0.02, 1.235), (0.44, 0.015, 1.20)],
         r=curve([(0, 0.035), (0.4, 0.068), (0.75, 0.062), (1, 0.03)]), segments=6, rings=4,
         bone="forearm.L", color=STEEL, material="metal", mirror="x"),
    # vambrace: flared toward the elbow
    Loft("vambrace", path=[(0.43, 0, 1.215), (0.57, 0, 1.07)], r=curve([(0, 0.062), (0.3, 0.056), (1, 0.047)]),
         segments=8, rings=3, shell=0.008, shell_inner=False, caps=(False, False), bone="forearm.L", color=STEEL, material="metal", mirror="x"),
    # gauntlet: the plated hand, flared at the wrist (the cuff detail is painted)
    Loft("hand", path=[(0.565, 0, 1.075), (0.625, 0, 1.015), (0.67, 0, 0.97)], rx=curve([(0, 0.056), (0.35, 0.048), (1, 0.03)]),
         ry=curve([(0, 0.052), (0.35, 0.034), (1, 0.024)]), n=2.5, segments=6, rings=3,
         bone="hand.L", color=STEEL, material="metal", mirror="x"),

    # --- legs -------------------------------------------------------------------------------------
    # knee cop: a domed plate over the knee with side wings
    Loft("poleyn", path=[(0.11, -0.03, 0.62), (0.112, -0.05, 0.53), (0.113, -0.03, 0.45)],
         rx=curve([(0, 0.05), (0.45, 0.088), (1, 0.05)]), ry=curve([(0, 0.035), (0.45, 0.058), (1, 0.035)]),
         n=2.2, segments=8, rings=4, bone="shin.L", color=STEEL, material="metal", mirror="x"),
    # greave: the whole shin, full at the calf, flaring over the foot
    Loft("greave", path=[(0.112, -0.006, 0.50), (0.114, -0.002, 0.30), (0.116, 0.004, 0.10)],
         r=curve([(0, 0.078), (0.22, 0.086), (0.6, 0.072), (0.85, 0.07), (1, 0.082)]),
         offset=lambda t: (0, -0.008 * bump(t), 0),
         segments=8, rings=4, shell=0.008, shell_inner=False, caps=(False, False), bone="shin.L", color=STEEL, material="metal", mirror="x"),
    # sabaton: a rounded plated shoe from the heel to a blunt toe, flat underneath
    Loft("foot", path=[(0.116, 0.075, 0.0), (0.118, 0.0, 0.0), (0.12, -0.10, 0.0), (0.122, -0.19, 0.0)],
         rx=curve([(0, 0.045), (0.3, 0.055), (0.75, 0.056), (1, 0.035)]),
         ry=curve([(0, 0.06), (0.3, 0.075), (0.6, 0.05), (0.9, 0.032), (1, 0.02)]),
         offset=lambda t: (0, 0, curve([(0, 0.06), (0.3, 0.075), (0.6, 0.05), (0.9, 0.032), (1, 0.02)])(t) * 0.92),
         n=2.6, segments=8, rings=4, bone="foot.L", color=STEEL, material="metal", mirror="x"),
]

G = (-0.668, -0.02, 0.975)  # sword grip, in the right hand (rest pose)
weapons = [
    # long blade with a ridge (diamond section), tapering to the point
    Loft("blade", path=[(G[0], G[1], G[2] - 0.08), (G[0], G[1], G[2] - 0.80), (G[0], G[1], G[2] - 0.97)],
         rx=curve([(0, 0.03), (0.8, 0.022), (1, 0.002)]), ry=curve([(0, 0.009), (1, 0.004)]), n=1.3, segments=4, rings=5,
         bone="hand.R", color=STEEL, material="metal"),
    # guard: a bar whose ends curve down toward the blade
    Extrude("guard", outline=[(-0.13, -0.045), (-0.115, -0.02), (-0.07, 0.0), (0.07, 0.0), (0.115, -0.02), (0.13, -0.045),
                              (0.14, -0.035), (0.125, 0.0), (0.08, 0.022), (-0.08, 0.022), (-0.125, 0.0), (-0.14, -0.035)],
            depth=0.028, origin=(G[0], G[1], G[2] - 0.075), bone="hand.R", color=STEEL, material="metal"),
    Loft("grip", path=[(G[0], G[1], G[2] - 0.075), (G[0], G[1], G[2] + 0.085)], r=curve([(0, 0.016), (1, 0.017)]),
         segments=6, rings=3, caps=(False, False), bone="hand.R", color=LEATHER, material="leather"),
    # spiked pommel
    Loft("pommel", path=[(G[0], G[1], G[2] + 0.085), (G[0], G[1], G[2] + 0.12), (G[0], G[1], G[2] + 0.16)],
         r=curve([(0, 0.018), (0.35, 0.032), (1, 0.003)]), segments=6, rings=4, bone="hand.R", color=STEEL, material="metal"),
    # large heater shield, curved around the arm (wrap: a vertical cylinder of radius 0.45 m)
    Extrude("shield", outline=[(-0.30, 0.44), (0.0, 0.47), (0.30, 0.44), (0.30, 0.10), (0.26, -0.15), (0.17, -0.34),
                               (0.0, -0.50), (-0.17, -0.34), (-0.26, -0.15), (-0.30, 0.10)],
            depth=0.035, origin=(0.50, -0.092 + 0.45, 1.10), wrap=(0.45, 0.45, 0.0), bone="hand.L", color=DARK,
            material=(0.55, 0.5)),  # back face against the gauntlet cuff
]

MODEL = Model(
    id="proto_knight",
    skeleton=HUMANOID,
    parts=body + armor + weapons,
    clips={
        "idle": anim.idle(),
        "brace": anim.brace(),
        "unbrace": anim.unbrace(),
        "walk": anim.walk(stride=0.6),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "die": anim.die(),
    },
    skins=["central"],
    texture="proto/proto_knight_v2",   # new shapes, new atlas: repaint (the v1 texture stays in proto/proto_knight)
    texture_size=512,
    budget_tris=2600,
    fps=30,
    team_tint=False,   # the player color is shown by the ring under the unit
)
