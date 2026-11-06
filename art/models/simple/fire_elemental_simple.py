"""Fire Elemental (a summoned column of fire with shoulders and fists, no legs) — stand-in from
plain shapes, 1.4x size. No team-colored part: the ring under the unit shows the owner."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import elemental

MODEL = Model(
    id="fire_elemental_simple",
    skeleton=HUMANOID,
    parts=elemental((0.55, 0.12, 0.05), (1.0, 0.55, 0.12), shards=False),
    clips={
        "idle": anim.idle(stance=anim.FIST_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.5, carry=anim.FIST_RELAXED),
        "run": anim.run(carry=anim.FIST_RELAXED),
        "attack": anim.attack_sword(rest=anim.FIST_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200, scale=1.4,
)
