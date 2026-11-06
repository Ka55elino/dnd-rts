"""Mortar (Dwarves: a squat iron barrel on a wheeled carriage) — stand-in from plain shapes on a rigid
skeleton. Team color: the carriage's plate."""
from modelc import Model
from art.skeletons.machines import MORTAR
from art.anims import machines as anim
from art.models.simple.kit import *

MODEL = Model(
    id="mortar_simple",
    skeleton=MORTAR,
    parts=[
        box("carriage", (0, 0.5, 0.30), (0, -0.5, 0.30), 0.40, 0.18, bone="base", color=WOOD),
        box("plate", (0, 0.45, 0.50), (0, -0.45, 0.50), 0.36, 0.03, bone="base", team=True),
        cyl("wheel", (0.42, 0.25, 0.25), (0.48, 0.25, 0.25), 0.25, seg=8, bone="base", color=DARKSTEEL, mirror="x"),
        cyl("wheel2", (0.42, -0.30, 0.25), (0.48, -0.30, 0.25), 0.25, seg=8, bone="base", color=DARKSTEEL, mirror="x"),
        cone("barrel", (0, 0.05, 0.50), (0, -0.25, 1.05), 0.20, 0.17, seg=8, bone="barrel", color=DARKSTEEL),
    ],
    clips={
        "idle": anim.mortar_idle(),
        "walk": anim.mortar_move(),
        "run": anim.mortar_move(duration=0.6, ground_speed=2.0),
        "attack": anim.mortar_fire(),
        "die": anim.mortar_wreck(),
    },
    texture=None, texture_size=256, budget_tris=1200, foot_bone="base",
)
