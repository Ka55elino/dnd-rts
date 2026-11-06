"""Golem (Dwarves: a stone construct with a glowing rune core) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="golem_simple",
    skeleton=HUMANOID,
    parts=elemental((0.45, 0.42, 0.38), (1.0, 0.55, 0.20)),
    clips={
        "idle": anim.idle(stance=anim.FIST_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.5, carry=anim.FIST_RELAXED),
        "run": anim.run(carry=anim.FIST_RELAXED),
        "attack": anim.attack_sword(rest=anim.FIST_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.7,
)
