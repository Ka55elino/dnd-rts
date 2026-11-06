"""Rune Master hero (Dwarves: a blue robe, a rune staff) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="runemaster_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.22, 0.30, 0.48), arms=(0.22, 0.30, 0.48), robe=(0.22, 0.30, 0.48), tabard=False, sash=True, head='hood', head_color=(0.22, 0.30, 0.48), skin=(0.80, 0.62, 0.50), build=1.35) + beard((0.80, 0.80, 0.78)) + staff((0.40, 0.80, 1.0), length=1.3),
    clips={
        "idle": anim.idle(stance=anim.STAFF_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.55, carry=anim.STAFF_CARRY),
        "run": anim.run(carry=anim.STAFF_CARRY),
        "attack": anim.cast_staff(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=0.8,
)
