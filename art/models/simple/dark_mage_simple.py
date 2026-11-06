"""Dark Mage (Dark Elves: a black robe, a staff with a violet orb) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="dark_mage_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.12, 0.10, 0.14), arms=(0.12, 0.10, 0.14), robe=(0.12, 0.10, 0.14), tabard=False, sash=True, head='hood', head_color=(0.35, 0.15, 0.42), skin=(0.42, 0.38, 0.48)) + staff((0.70, 0.35, 1.0)),
    clips={
        "idle": anim.idle(stance=anim.STAFF_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.55, carry=anim.STAFF_CARRY),
        "run": anim.run(carry=anim.STAFF_CARRY),
        "attack": anim.cast_staff(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.0,
)
