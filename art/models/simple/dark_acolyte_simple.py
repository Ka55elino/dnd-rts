"""Dark Acolyte (Alkars: a red-and-black robe, a necrotic staff) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="dark_acolyte_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.50, 0.08, 0.08), arms=(0.15, 0.14, 0.15), robe=(0.15, 0.14, 0.15), tabard=False, sash=True, head='hood', head_color=(0.50, 0.08, 0.08), skin=(0.72, 0.72, 0.74)) + staff((0.45, 1.0, 0.55)),
    clips={
        "idle": anim.idle(stance=anim.STAFF_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.55, carry=anim.STAFF_CARRY),
        "run": anim.run(carry=anim.STAFF_CARRY),
        "attack": anim.cast_staff(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.0,
)
