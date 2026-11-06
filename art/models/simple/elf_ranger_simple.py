"""Elf Ranger hero (High Elves: a hooded archer in green and silver) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="elf_ranger_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.22, 0.45, 0.30), arms=(0.80, 0.82, 0.86), forearms=(0.80, 0.82, 0.86), legs=(0.22, 0.45, 0.30), head='hood', head_color=(0.15, 0.30, 0.20), skin=(0.86, 0.75, 0.64), build=0.95) + bow(),
    clips={
        "idle": anim.idle(stance=anim.BOW_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.65, carry=anim.BOW_CARRY),
        "run": anim.run(carry=anim.BOW_CARRY),
        "attack": anim.attack_bow(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.06,
)
