"""Alkar Archer (Alkars elite: black armour, a bow; shoots air) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="alkar_archer_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.15, 0.14, 0.15), arms=(0.50, 0.08, 0.08), forearms=(0.15, 0.14, 0.15), legs=(0.15, 0.14, 0.15), head='hood', head_color=(0.50, 0.08, 0.08), skin=(0.72, 0.72, 0.74)) + bow(),
    clips={
        "idle": anim.idle(stance=anim.BOW_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.65, carry=anim.BOW_CARRY),
        "run": anim.run(carry=anim.BOW_CARRY),
        "attack": anim.attack_bow(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.0,
)
