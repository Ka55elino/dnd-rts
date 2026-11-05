"""Test model for the models-as-code pipeline: a humanoid body + plate armor, sword and shield,
all from formulas. Not a game unit yet: it only appears in Model Overview (id proto_knight).

Build:  python3 tools/modelc/modelc build proto_knight
"""
import math
from modelc import Model, Loft, Extrude, curve
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim

CLOTH = (0.13, 0.11, 0.11)
SKIN = (0.78, 0.62, 0.52)
STEEL = (0.72, 0.74, 0.78)
DARK = (0.14, 0.14, 0.18)
LEATHER = (0.36, 0.22, 0.12)
GOLD = (0.78, 0.62, 0.28)

body = [
    # torso: one loft from the hips to the neck, blended over 4 bones
    Loft("torso", path=[(0, 0.0, 0.90), (0, 0.005, 1.10), (0, 0, 1.30), (0, 0, 1.50), (0, 0, 1.58)],
         rx=curve([(0, 0.155), (0.3, 0.14), (0.6, 0.185), (0.85, 0.17), (1, 0.06)]),
         ry=curve([(0, 0.11), (0.3, 0.10), (0.6, 0.125), (0.85, 0.10), (1, 0.055)]),
         n=2.3, segments=10, rings=9, bones=["hips", "spine", "chest", "neck"], color=CLOTH, uv="torso", uv_scale=0.15),  # hidden under the cuirass
    Loft("head", path=[(0, -0.005, 1.57), (0, -0.005, 1.82)], r=lambda t: 0.105 * math.sin(math.pi * (0.08 + 0.9 * t)) ** 0.6,
         segments=8, rings=6, bone="head", color=SKIN, uv_scale=0.1),  # hidden under the helmet
    Loft("arm", path=[(0.17, 0, 1.46), (0.40, 0, 1.24), (0.58, 0, 1.06)],
         r=curve([(0, 0.058), (0.45, 0.046), (0.55, 0.044), (1, 0.034)]),
         segments=8, rings=7, caps=(False, True), bones=["upperarm.L", "forearm.L"], color=CLOTH, mirror="x"),
    Loft("hand", path=[(0.575, 0, 1.065), (0.665, 0, 0.975)], rx=0.04, ry=0.026, n=2.6, segments=6, rings=3,
         bone="hand.L", color=LEATHER, mirror="x"),
    Loft("leg", path=[(0.10, 0, 0.98), (0.11, 0, 0.52), (0.115, 0.01, 0.09)],
         r=curve([(0, 0.09), (0.45, 0.068), (0.55, 0.066), (1, 0.046)]),
         segments=8, rings=8, caps=(False, True), bones=["thigh.L", "shin.L"], color=CLOTH, mirror="x"),
    # foot (sabaton): a side profile — high heel, sloping instep, pointed toe — extruded across the width
    Extrude("foot", outline=[(-0.065, 0.0), (-0.07, 0.10), (-0.02, 0.125), (0.05, 0.10), (0.16, 0.035), (0.185, 0.0)],
            depth=0.095, origin=(0.115, 0.01, 0.0), u_axis=(0, -1, 0), v_axis=(0, 0, 1),
            bone="foot.L", color=STEEL, mirror="x"),
]

armor = [
    Loft("helmet", path=[(0, -0.005, 1.55), (0, -0.005, 1.87)],
         rx=curve([(0, 0.12), (0.3, 0.135), (0.75, 0.125), (1, 0.03)]),
         ry=curve([(0, 0.13), (0.3, 0.148), (0.75, 0.135), (1, 0.035)]),
         n=2.2, segments=10, rings=6, caps=(False, True), bone="head", color=STEEL),
    Extrude("crest", outline=[(-0.12, 1.80), (-0.08, 1.99), (0.02, 2.06), (0.14, 2.01), (0.20, 1.80), (0.12, 1.78), (0.02, 1.88)],
            depth=0.05, u_axis=(0, 1, 0), v_axis=(0, 0, 1), bone="head", team=True),
    Loft("cuirass", path=[(0, 0, 1.02), (0, 0, 1.30), (0, 0, 1.52)],
         rx=curve([(0, 0.18), (0.5, 0.215), (0.8, 0.235), (1, 0.16)]),
         ry=curve([(0, 0.13), (0.5, 0.15), (0.8, 0.145), (1, 0.11)]),
         offset=lambda t: (0, -0.012 * math.sin(math.pi * t), 0),
         n=2.2, segments=10, rings=6, caps=(False, False), bones=["spine", "chest"], color=STEEL),
    Loft("pauldron", path=[(0.14, 0, 1.56), (0.23, 0, 1.535), (0.30, 0, 1.46), (0.34, 0, 1.37)],
         rx=curve([(0, 0.055), (0.3, 0.095), (0.7, 0.09), (1, 0.06)]),
         ry=curve([(0, 0.07), (0.3, 0.11), (0.7, 0.105), (1, 0.08)]),
         segments=8, rings=5, bone="upperarm.L", color=STEEL, mirror="x"),
    Loft("vambrace", path=[(0.42, 0, 1.225), (0.585, 0, 1.055)], r=curve([(0, 0.056), (1, 0.047)]),
         segments=6, rings=3, caps=(False, False), bone="forearm.L", color=STEEL, mirror="x"),
    Loft("greave", path=[(0.11, -0.012, 0.63), (0.115, -0.006, 0.10)],
         r=curve([(0, 0.08), (0.18, 0.094), (0.5, 0.074), (1, 0.06)]),
         segments=8, rings=6, caps=(True, False), bone="shin.L", color=STEEL, mirror="x"),
    Extrude("strips", outline=[(-0.15, 0.03), (0.15, 0.03), (0.15, -0.28), (0.10, -0.32), (0.065, -0.29), (0.033, -0.35),
                               (0.0, -0.38), (-0.033, -0.35), (-0.065, -0.29), (-0.10, -0.32), (-0.15, -0.28)],
            depth=0.014, origin=(0, 0, 1.02), wrap=(0.185, 0.14, 0.25), bone="hips", team=True),
    # the same strips at the back (they share the texture region of the front ones)
    Extrude("strips_back", outline=[(-0.15, 0.03), (0.15, 0.03), (0.15, -0.28), (0.10, -0.32), (0.065, -0.29), (0.033, -0.35),
                                    (0.0, -0.38), (-0.033, -0.35), (-0.065, -0.29), (-0.10, -0.32), (-0.15, -0.28)],
            depth=0.014, origin=(0, 0, 1.02), wrap=(0.185, 0.15, 0.25, 180), bone="hips", team=True, uv="strips"),
]

G = (-0.668, -0.02, 0.975)  # sword grip, in the right hand (rest pose)
weapons = [
    Loft("blade", path=[(G[0], G[1], G[2] - 0.07), (G[0], G[1], G[2] - 0.85), (G[0], G[1], G[2] - 0.95)],
         rx=curve([(0, 0.032), (0.85, 0.024), (1, 0.002)]), ry=0.008, segments=4, rings=4,
         bone="hand.R", color=STEEL),
    Extrude("guard", outline=[(-0.10, -0.014), (0.10, -0.014), (0.11, 0.012), (-0.11, 0.012)], depth=0.03,
            origin=(G[0], G[1], G[2] - 0.06), bone="hand.R", color=GOLD),
    Loft("grip", path=[(G[0], G[1], G[2] - 0.06), (G[0], G[1], G[2] + 0.08)], r=curve([(0, 0.016), (0.9, 0.016), (1, 0.028)]),
         segments=6, rings=3, bone="hand.R", color=LEATHER),
    Extrude("shield", outline=[(-0.25, 0.38), (0.0, 0.41), (0.25, 0.38), (0.25, 0.05), (0.20, -0.17), (0.11, -0.31),
                               (0.0, -0.40), (-0.11, -0.31), (-0.20, -0.17), (-0.25, 0.05)],
            depth=0.04, origin=(0.50, -0.115, 1.10), bone="forearm.L", color=DARK),
]

MODEL = Model(
    id="proto_knight",
    skeleton=HUMANOID,
    parts=body + armor + weapons,
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(stride=0.6),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "die": anim.die(),
    },
    skins=["central"],
    texture="proto/proto_knight",
    texture_size=512,
    budget_tris=3000,
    team_tint=False,   # the player color is shown by the ring under the unit
)
