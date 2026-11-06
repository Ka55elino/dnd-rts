"""Storm Elemental (a summoned column of storm with shoulders and fists, no legs) — stand-in from
plain shapes, 1.4x size. No team-colored part: the ring under the unit shows the owner."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import elemental

MODEL = Model(
    id="storm_elemental_simple",
    skeleton=HUMANOID,
    parts=elemental((0.20, 0.22, 0.40), (0.55, 0.75, 1.0), shards=False),
    clips={
        "idle": anim.idle(stance=anim.FIST_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.5, carry=anim.FIST_RELAXED),
        "run": anim.run(carry=anim.FIST_RELAXED),
        "attack": anim.cast_staff(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200, scale=1.4,
)
