"""Archmage hero (long robe, tall pointed hat, a large staff) — stand-in from plain shapes.
Team color: a sash at the waist."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import humanoid, staff, STEEL

ROBE = (0.30, 0.22, 0.45)
FOCUS = (0.9, 0.85, 1.0)

MODEL = Model(
    id="archmage_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=ROBE, arms=ROBE, forearms=ROBE, robe=ROBE, tabard=False, sash=True,
                   head="hat", head_color=ROBE) + staff(FOCUS, length=1.95, head_r=0.09),
    clips={
        "idle": anim.idle(stance=anim.STAFF_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.55, carry=anim.STAFF_CARRY),
        "run": anim.run(carry=anim.STAFF_CARRY),
        "attack": anim.cast_staff(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200,
)
