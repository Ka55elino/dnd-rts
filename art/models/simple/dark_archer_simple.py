"""Dark Archer hero (Dark Elves: a black-and-silver archer) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="dark_archer_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.12, 0.10, 0.14), arms=(0.80, 0.78, 0.82), forearms=(0.12, 0.10, 0.14), legs=(0.12, 0.10, 0.14), head='hood', head_color=(0.12, 0.10, 0.14), skin=(0.42, 0.38, 0.48)) + bow(),
    clips={
        "idle": anim.idle(stance=anim.BOW_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.65, carry=anim.BOW_CARRY),
        "run": anim.run(carry=anim.BOW_CARRY),
        "attack": anim.attack_bow(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.05,
)
