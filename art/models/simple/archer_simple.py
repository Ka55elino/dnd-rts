"""Archer (bow, quiver, padded jacket, hood) — stand-in from plain shapes. Team color: tabard."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import humanoid, bow, CLOTH, LEATHER

GREEN = (0.28, 0.34, 0.20)

MODEL = Model(
    id="archer_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=LEATHER, arms=LEATHER, head="hood", head_color=GREEN) + bow(),
    clips={
        "idle": anim.idle(stance=anim.BOW_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.65, carry=anim.BOW_CARRY),
        "run": anim.run(carry=anim.BOW_CARRY),
        "attack": anim.attack_bow(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200,
)
