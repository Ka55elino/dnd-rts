"""Gunner (Dwarves: a long gun) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="gunner_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.22, 0.30, 0.48), arms=STEEL, forearms=(0.22, 0.30, 0.48), legs=(0.22, 0.30, 0.48), head='helmet', skin=(0.80, 0.62, 0.50), build=1.35) + beard((0.30, 0.20, 0.12)) + gun(),
    clips={
        "idle": anim.idle(stance=anim.GUN_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.55, carry=anim.GUN_CARRY),
        "run": anim.run(carry=anim.GUN_CARRY),
        "attack": anim.attack_gun(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=0.78,
)
