"""Blade Dancer (Dark Elves: light leather, a blade in each hand) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="blade_dancer_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.12, 0.10, 0.14), arms=(0.42, 0.38, 0.48), forearms=(0.35, 0.15, 0.42), legs=(0.12, 0.10, 0.14), head='bare', skin=(0.42, 0.38, 0.48), build=0.9) + ears((0.42, 0.38, 0.48)) + sword(0.6) + left_sword(0.6),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(stride=0.6),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "brace": anim.brace(),
        "unbrace": anim.unbrace(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.03,
)
