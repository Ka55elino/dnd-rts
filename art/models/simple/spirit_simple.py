"""Spirit (High Elves builder: a glowing figure without legs) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="spirit_simple",
    skeleton=HUMANOID,
    parts=floating((0.55, 0.85, 0.65), (0.85, 1.0, 0.9)),
    clips={
        "idle": anim.idle(stance=anim.FIST_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.5, carry=anim.FIST_RELAXED),
        "run": anim.run(carry=anim.FIST_RELAXED),
        "attack": anim.attack_sword(rest=anim.FIST_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.0,
)
