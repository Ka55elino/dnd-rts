"""Crossbowman (Dark Elves: a crossbow in each hand) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="crossbowman_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.12, 0.10, 0.14), arms=(0.35, 0.15, 0.42), forearms=(0.12, 0.10, 0.14), legs=(0.12, 0.10, 0.14), head='helmet', head_color=(0.25, 0.22, 0.28), skin=(0.42, 0.38, 0.48)) + crossbow() + crossbow(left=True),
    clips={
        "idle": anim.idle(stance=anim.GUN_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.55, carry=anim.GUN_CARRY),
        "run": anim.run(carry=anim.GUN_CARRY),
        "attack": anim.attack_gun(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.0,
)
