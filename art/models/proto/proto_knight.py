"""Shape study after the BFME2 look (a Gondor-style lord): slender human proportions, a narrow
cuirass, layered lames down the upper arm, long flared vambraces, a long black surcoat split at the
front over a mail skirt, greaves over dark boots, bare head with long hair and a crown, a long
straight sword and a scabbard at the left hip. No texture yet: flat part colors only.

Not a game unit: it appears in Model Overview (id proto_knight).
Build:  python3 tools/modelc/modelc build proto_knight
"""
import math
from modelc import Model, Loft, Extrude, curve
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim

SKIN = (0.76, 0.60, 0.50)
HAIR = (0.13, 0.09, 0.07)
STEEL = (0.74, 0.76, 0.80)
DARKSTEEL = (0.40, 0.42, 0.46)
MAIL = (0.30, 0.31, 0.34)
BLACK = (0.08, 0.08, 0.09)       # surcoat
LEATHER = (0.13, 0.10, 0.09)     # gloves, boots, belt
GOLD = (0.80, 0.65, 0.30)

body = [
    # under the clothes: fillers that keep the openings closed
    Loft("torso", path=[(0, 0.0, 0.95), (0, 0, 1.25), (0, 0, 1.56)],
         rx=curve([(0, 0.14), (0.5, 0.15), (1, 0.08)]), ry=curve([(0, 0.10), (0.5, 0.11), (1, 0.07)]),
         n=2.3, segments=6, rings=3, bones=["hips", "spine", "chest", "neck"], color=BLACK, uv_scale=0.1),
    # head and face
    Loft("head", path=[(0, -0.01, 1.58), (0, -0.012, 1.70), (0, -0.005, 1.83)],
         rx=curve([(0, 0.05), (0.2, 0.072), (0.6, 0.082), (1, 0.03)]),
         ry=curve([(0, 0.06), (0.2, 0.09), (0.6, 0.098), (1, 0.035)]),
         offset=lambda t: (0, -0.012 * math.sin(math.pi * t), 0),
         segments=8, rings=5, bone="head", color=SKIN, material="skin"),
    # sleeves (mail) from the shoulder into the vambrace
    Loft("arm", path=[(0.17, 0, 1.46), (0.40, 0, 1.24), (0.47, 0, 1.17)], r=curve([(0, 0.05), (0.7, 0.043), (1, 0.04)]),
         segments=8, rings=4, caps=(False, True), bones=["upperarm.L", "forearm.L"], color=MAIL, material=(0.35, 0.3),
         mirror="x"),
    # legs: mail chausses from the hip into the greave
    Loft("leg", path=[(0.095, 0, 0.98), (0.105, 0, 0.52), (0.108, 0.002, 0.44)],
         r=curve([(0, 0.085), (0.45, 0.072), (0.85, 0.056), (1, 0.052)]),
         segments=8, rings=4, caps=(False, True), bones=["thigh.L", "shin.L"], color=MAIL, material=(0.35, 0.3), mirror="x"),
]

head = [
    # long hair: from the top of the head down the back to the shoulders (the crown is painted)
    Loft("hair", path=[(0, 0.015, 1.84), (0, 0.045, 1.72), (0, 0.06, 1.60), (0, 0.075, 1.50)],
         rx=curve([(0, 0.05), (0.2, 0.092), (0.6, 0.095), (1, 0.08)]),
         ry=curve([(0, 0.05), (0.2, 0.08), (0.6, 0.06), (1, 0.035)]),
         segments=8, rings=4, bones=["head", "neck"], color=HAIR, material=(0.12, 0.3)),

]

armor = [
    # gorget and a narrow cuirass: chest a little forward, waist close to the body
    Loft("gorget", path=[(0, 0.0, 1.49), (0, -0.004, 1.59)], rx=curve([(0, 0.115), (0.5, 0.07), (1, 0.055)]), ry=curve([(0, 0.095), (0.5, 0.065), (1, 0.055)]),
         n=2.2, segments=10, rings=3, shell=0.006, shell_inner=False, caps=(False, False), bones=["chest", "neck"],
         color=STEEL, material="metal"),
    Loft("cuirass", path=[(0, 0, 1.02), (0, 0, 1.18), (0, 0, 1.36), (0, 0, 1.52)],
         rx=curve([(0, 0.150), (0.25, 0.145), (0.6, 0.178), (0.85, 0.18), (1, 0.13)]),
         ry=curve([(0, 0.112), (0.25, 0.108), (0.6, 0.128), (0.85, 0.12), (1, 0.09)]),
         offset=lambda t: (0, -0.015 * math.sin(math.pi * min(max((t - 0.3) / 0.7, 0), 1)), 0),
         n=2.3, segments=12, rings=6, shell=0.006, shell_inner=False, caps=(False, False), bones=["spine", "chest"],
         color=STEEL, material="metal"),
    # belt
    Loft("belt", path=[(0, 0, 0.985), (0, 0, 1.035)], rx=0.158, ry=0.12, n=2.3, segments=12, rings=2,
         shell=0.006, shell_inner=False, caps=(False, False), bone="hips", color=LEATHER, material="leather"),
    # mail skirt under the surcoat (shows in the split at the front), to the knee
    Loft("mail_skirt", path=[(0, 0, 1.0), (0, -0.005, 0.72), (0, -0.01, 0.46)],
         rx=curve([(0, 0.148), (0.5, 0.158), (1, 0.168)]), ry=curve([(0, 0.112), (0.5, 0.118), (1, 0.125)]),
         n=2.0, segments=12, rings=4, shell=0.006, shell_inner=False, caps=(False, False),
         bones=["hips", "thigh.L", "thigh.R"], color=MAIL, material=(0.35, 0.3)),
    # surcoat: two long black panels from the belt to the shins, wrapped around the body and open at
    # the front (each from 22° off the front, around the side, to the middle of the back); each panel
    # follows its own leg
    Extrude("surcoat", outline=[(-0.235, 0.02), (0.235, 0.02), (0.25, -0.40), (0.26, -0.66), (-0.21, -0.66), (-0.22, -0.40)],
            depth=0.01, origin=(0, 0, 1.0), wrap=(0.172, 0.136, 0.45, 101), bones=["hips", "thigh.L"],
            color=BLACK, material="cloth", mirror="x"),

    # --- arms -------------------------------------------------------------------------------------
    # pauldron: one smooth plate over the shoulder and down the upper arm (its lames are painted)
    Loft("pauldron", path=[(0.12, 0, 1.555), (0.22, 0, 1.535), (0.29, 0, 1.43), (0.325, 0, 1.33)],
         rx=curve([(0, 0.04), (0.3, 0.072), (0.7, 0.068), (1, 0.072)]),
         ry=curve([(0, 0.05), (0.3, 0.085), (0.7, 0.078), (1, 0.08)]),
         n=2.2, segments=8, rings=5, shell=0.006, caps=(True, False), bone="upperarm.L", color=STEEL, material="metal",
         mirror="x"),
    # long vambrace, flared at the elbow
    Loft("vambrace", path=[(0.405, 0, 1.235), (0.575, 0, 1.065)], r=curve([(0, 0.064), (0.25, 0.055), (1, 0.046)]),
         segments=8, rings=3, shell=0.006, shell_inner=False, caps=(False, False), bone="forearm.L",
         color=DARKSTEEL, material="metal", mirror="x"),
    # dark leather glove
    Loft("hand", path=[(0.57, 0, 1.07), (0.625, 0, 1.015), (0.67, 0, 0.97)], rx=curve([(0, 0.05), (0.35, 0.046), (1, 0.03)]),
         ry=curve([(0, 0.045), (0.35, 0.032), (1, 0.024)]), n=2.5, segments=6, rings=3,
         bone="hand.L", color=LEATHER, material="leather", mirror="x"),

    # --- legs -------------------------------------------------------------------------------------
    Loft("poleyn", path=[(0.106, -0.03, 0.59), (0.108, -0.045, 0.53), (0.109, -0.03, 0.47)],
         rx=curve([(0, 0.045), (0.45, 0.07), (1, 0.045)]), ry=curve([(0, 0.03), (0.45, 0.048), (1, 0.03)]),
         n=2.2, segments=8, rings=4, bone="shin.L", color=STEEL, material="metal", mirror="x"),
    Loft("greave", path=[(0.108, -0.004, 0.50), (0.11, 0.0, 0.30), (0.112, 0.004, 0.16)],
         r=curve([(0, 0.066), (0.25, 0.072), (0.7, 0.058), (1, 0.056)]),
         segments=8, rings=4, shell=0.006, shell_inner=False, caps=(False, False), bone="shin.L",
         color=STEEL, material="metal", mirror="x"),
    # dark boot from the ankle to a rounded toe, flat underneath
    Loft("foot", path=[(0.112, 0.06, 0.0), (0.114, 0.0, 0.0), (0.116, -0.09, 0.0), (0.118, -0.17, 0.0)],
         rx=curve([(0, 0.042), (0.3, 0.05), (0.75, 0.048), (1, 0.03)]),
         ry=curve([(0, 0.075), (0.3, 0.085), (0.6, 0.045), (0.9, 0.03), (1, 0.02)]),
         offset=lambda t: (0, 0, curve([(0, 0.075), (0.3, 0.085), (0.6, 0.045), (0.9, 0.03), (1, 0.02)])(t) * 0.92),
         n=2.5, segments=8, rings=4, bone="foot.L", color=LEATHER, material="leather", mirror="x"),
]

G = (-0.668, -0.02, 0.975)  # sword grip, in the right hand (rest pose)
weapons = [
    # long straight blade
    Loft("blade", path=[(G[0], G[1], G[2] - 0.07), (G[0], G[1], G[2] - 0.92), (G[0], G[1], G[2] - 1.02)],
         rx=curve([(0, 0.022), (0.85, 0.017), (1, 0.002)]), ry=curve([(0, 0.006), (1, 0.003)]), n=1.3, segments=4, rings=4,
         bone="hand.R", color=STEEL, material="metal"),
    Extrude("guard", outline=[(-0.09, -0.008), (0.09, -0.008), (0.10, 0.008), (-0.10, 0.008)],
            depth=0.022, origin=(G[0], G[1], G[2] - 0.065), bone="hand.R", color=STEEL, material="metal"),
    Loft("grip", path=[(G[0], G[1], G[2] - 0.065), (G[0], G[1], G[2] + 0.10)], r=0.015,
         segments=6, rings=2, caps=(False, False), bone="hand.R", color=LEATHER, material="leather"),
    Loft("pommel", path=[(G[0], G[1], G[2] + 0.10), (G[0], G[1], G[2] + 0.14)], r=curve([(0, 0.02), (0.5, 0.026), (1, 0.01)]),
         segments=6, rings=3, bone="hand.R", color=STEEL, material="metal"),
    # empty scabbard at the left hip, hanging back and down
    Loft("scabbard", path=[(0.17, -0.06, 0.98), (0.24, 0.10, 0.62), (0.29, 0.22, 0.30)],
         rx=curve([(0, 0.026), (1, 0.016)]), ry=curve([(0, 0.012), (1, 0.008)]), segments=6, rings=3,
         bone="hips", color=BLACK, material=(0.2, 0.4)),
]

MODEL = Model(
    id="proto_knight",
    skeleton=HUMANOID,
    parts=body + head + armor + weapons,
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
    texture="proto/proto_knight_v3",   # the lord: AI-painted, fitted with `modelc fit`
    texture_size=512,
    budget_tris=2700,
    fps=30,
    team_tint=False,
)
