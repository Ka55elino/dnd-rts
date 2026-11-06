"""Dark Mage hero (Alkars: a black robe with red, a fiery staff) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="alkar_dark_mage_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.15, 0.14, 0.15), arms=(0.50, 0.08, 0.08), robe=(0.15, 0.14, 0.15), tabard=False, sash=True, head='hat', head_color=(0.15, 0.14, 0.15), skin=(0.72, 0.72, 0.74)) + staff((1.0, 0.40, 0.10), head_r=0.08),
    clips={
        "idle": anim.idle(stance=anim.STAFF_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.55, carry=anim.STAFF_CARRY),
        "run": anim.run(carry=anim.STAFF_CARRY),
        "attack": anim.cast_staff(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.05,
)
