"""Elf King hero (High Elves: a crowned king with a sword on a white stag) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.cavalry import CAVALRY
from art.anims import cavalry as anim
from art.models.simple.kit import *

MODEL = Model(
    id="elf_king_simple",
    skeleton=CAVALRY,
    parts=horse(coat=(0.86, 0.82, 0.72), mane=(0.70, 0.62, 0.50), scale=0.95) + antlers()
          + mount_rider(humanoid(torso=(0.80, 0.82, 0.86), arms=(0.80, 0.82, 0.86), forearms=GOLD, legs=(0.22, 0.45, 0.30), head='crown', skin=(0.86, 0.75, 0.64), build=1.0) + sword(0.85), CAVALRY),
    clips={
        "idle": anim.idle(anim.SWORD),
        "walk": anim.walk(anim.SWORD, scale=CAVALRY.mount_scale),
        "run": anim.run(anim.SWORD, scale=CAVALRY.mount_scale),
        "attack": anim.attack_sword(),
        "die": anim.die(stance=anim.SWORD),
    },
    texture=None, texture_size=256, budget_tris=2000, foot_bone="fshin.L", scale=1.0,
)
