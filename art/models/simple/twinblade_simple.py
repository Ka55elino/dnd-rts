"""Twinblade (High Elves swordsman: a double-bladed sword and a shield) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="twinblade_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.80, 0.82, 0.86), arms=(0.22, 0.45, 0.30), forearms=(0.80, 0.82, 0.86), shins=(0.80, 0.82, 0.86), head='helmet', head_color=(0.80, 0.82, 0.86), skin=(0.86, 0.75, 0.64), build=0.95) + ears((0.86, 0.75, 0.64)) + sword(0.7) + [box('blade_back', off(G, dz=0.12), off(G, dz=0.75), 0.022, 0.006, bone='hand.R', color=STEEL)] + shield(),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(stride=0.6),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "brace": anim.brace(),
        "unbrace": anim.unbrace(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.04,
)
