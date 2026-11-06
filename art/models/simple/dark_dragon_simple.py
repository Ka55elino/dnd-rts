"""Dark Dragon (Dark Elves: a huge black dragon with wings) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.cavalry import CAVALRY
from art.anims import cavalry as anim
from art.models.simple.kit import *

MODEL = Model(
    id="dark_dragon_simple",
    skeleton=CAVALRY,
    parts=dragon_parts((0.14, 0.12, 0.16), (0.45, 0.30, 0.40)),
    clips={
        "idle": anim.fly(duration=0.9, flap=30),
        "walk": anim.fly(duration=0.7, flap=40, ground_speed=3.0),
        "run": anim.fly(duration=0.55, flap=50, ground_speed=8.0),
        "attack": anim.beast_attack(),
        "die": anim.die(stance=anim.SWORD),
    },
    texture=None, texture_size=256, budget_tris=2000, foot_bone="fshin.L", scale=2.0,
)
