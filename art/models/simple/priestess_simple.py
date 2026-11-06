"""Priestess hero (High Elves: an ivory robe, a moon staff) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="priestess_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.88, 0.87, 0.80), arms=(0.88, 0.87, 0.80), forearms=(0.88, 0.87, 0.80), robe=(0.88, 0.87, 0.80), tabard=False, sash=True, head='bare', skin=(0.86, 0.75, 0.64)) + ears((0.86, 0.75, 0.64)) + staff((0.85, 0.90, 1.0), head_r=0.08),
    clips={
        "idle": anim.idle(stance=anim.STAFF_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.55, carry=anim.STAFF_CARRY),
        "run": anim.run(carry=anim.STAFF_CARRY),
        "attack": anim.cast_staff(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.04,
)
