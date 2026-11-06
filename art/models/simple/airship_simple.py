"""Airship (Dwarves: a balloon over a wooden gondola with a propeller and a swivel gun) — stand-in from
plain shapes on a rigid skeleton. Team color: the balloon's band."""
from modelc import Model
from art.skeletons.machines import AIRSHIP
from art.anims import machines as anim
from art.models.simple.kit import *

MODEL = Model(
    id="airship_simple",
    skeleton=AIRSHIP,
    parts=[
        ball("balloon", (0, 0, 2.6), 0.9, ry=1.6, bone="hull", color=(0.72, 0.62, 0.45)),
        box("band", (0, 0, 2.55), (0, 0, 2.70), 0.92, 1.0, bone="hull", team=True),
        box("gondola", (0, 0.9, 1.0), (0, -0.9, 1.0), 0.40, 0.25, bone="hull", color=WOOD),
        cyl("rope", (0.35, 0, 1.2), (0.6, 0, 2.2), 0.02, bone="hull", color=LEATHER, mirror="x"),
        cyl("swivel", (0, -0.85, 1.3), (0, -1.35, 1.35), 0.05, bone="hull", color=DARKSTEEL),
        box("blade1", (-0.45, 1.3, 1.5), (0.45, 1.3, 1.5), 0.05, 0.02, bone="prop", color=WOOD),
        box("blade2", (0, 1.3, 1.05), (0, 1.3, 1.95), 0.05, 0.02, bone="prop", color=WOOD),
    ],
    clips={
        "idle": anim.airship_hover(),
        "walk": anim.airship_hover(duration=1.2, spin=6),
        "run": anim.airship_hover(duration=0.8, spin=8),
        "attack": anim.airship_fire(),
        "die": anim.airship_crash(),
    },
    texture=None, texture_size=256, budget_tris=1200, foot_bone="hull",
)
