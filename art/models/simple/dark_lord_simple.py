"""Dark Lord hero (Alkars: black-and-red plate, a horned helmet, a long sword) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="dark_lord_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.15, 0.14, 0.15), arms=(0.50, 0.08, 0.08), forearms=(0.15, 0.14, 0.15), legs=(0.15, 0.14, 0.15), shins=(0.15, 0.14, 0.15), head='horned', head_color=(0.15, 0.14, 0.15), skin=(0.72, 0.72, 0.74), build=1.2) + sword(1.0) + shield(),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(stride=0.6),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "brace": anim.brace(),
        "unbrace": anim.unbrace(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.12,
)
