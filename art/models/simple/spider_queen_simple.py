"""Spider Queen hero (Dark Elves: the high priestess of the Great Spider, a crowned robe, a staff) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="spider_queen_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.35, 0.15, 0.42), arms=(0.12, 0.10, 0.14), robe=(0.12, 0.10, 0.14), tabard=False, sash=True, head='crown', skin=(0.42, 0.38, 0.48)) + staff((0.55, 1.0, 0.45), head_r=0.08),
    clips={
        "idle": anim.idle(stance=anim.STAFF_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.55, carry=anim.STAFF_CARRY),
        "run": anim.run(carry=anim.STAFF_CARRY),
        "attack": anim.cast_staff(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.05,
)
