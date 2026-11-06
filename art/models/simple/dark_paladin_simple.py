"""Dark Paladin hero (Dark Elves: black plate, a sword and a shield) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="dark_paladin_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.18, 0.16, 0.22), arms=(0.18, 0.16, 0.22), forearms=(0.35, 0.15, 0.42), legs=(0.12, 0.10, 0.14), shins=(0.18, 0.16, 0.22), head='horned', head_color=(0.18, 0.16, 0.22), skin=(0.42, 0.38, 0.48), build=1.15) + sword(0.85) + shield(),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(stride=0.6),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "brace": anim.brace(),
        "unbrace": anim.unbrace(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.05,
)
