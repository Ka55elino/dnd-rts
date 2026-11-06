"""Ice Elemental (a summoned column of ice with shoulders and fists, no legs) — stand-in from
plain shapes, 1.4x size. No team-colored part: the ring under the unit shows the owner."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import elemental

MODEL = Model(
    id="ice_elemental_simple",
    skeleton=HUMANOID,
    parts=elemental((0.55, 0.75, 0.88), (0.85, 0.95, 1.0), shards=True),
    clips={
        "idle": anim.idle(stance=anim.FIST_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.5, carry=anim.FIST_RELAXED),
        "run": anim.run(carry=anim.FIST_RELAXED),
        "attack": anim.attack_sword(rest=anim.FIST_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200, scale=1.4,
)
