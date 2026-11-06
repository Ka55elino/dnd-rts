"""Builder (work clothes, leather apron, hammer) — stand-in from plain shapes. Team color: tabard (apron)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import humanoid, hammer, LEATHER

LINEN = (0.62, 0.55, 0.42)

MODEL = Model(
    id="worker_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=LINEN, arms=LINEN, forearms=LINEN, legs=LEATHER, head="bare") + hammer(),
    clips={
        "idle": anim.idle(stance=anim.TOOL_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.6, carry=anim.TOOL_RELAXED),
        "run": anim.run(carry=anim.TOOL_RELAXED),
        "attack": anim.attack_sword(rest=anim.TOOL_RELAXED),   # also the work swing
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200,
)
