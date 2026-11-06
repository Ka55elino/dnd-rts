"""Fire Mage (dark red robe, protective bracers, staff with a fire focus) — stand-in from plain shapes.
Team color: a sash at the waist."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import humanoid, staff, STEEL

ROBE = (0.45, 0.10, 0.08)
FOCUS = (1.0, 0.45, 0.10)

MODEL = Model(
    id="fire_mage_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=ROBE, arms=ROBE, forearms=STEEL, robe=ROBE, tabard=False, sash=True,
                   head="hood", head_color=ROBE) + staff(FOCUS),
    clips={
        "idle": anim.idle(stance=anim.STAFF_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.55, carry=anim.STAFF_CARRY),
        "run": anim.run(carry=anim.STAFF_CARRY),
        "attack": anim.cast_staff(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200,
)
