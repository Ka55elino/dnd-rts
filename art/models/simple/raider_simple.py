"""Raider (warband infantry: green skin, leather, an axe) — stand-in from plain shapes. Team color: tabard."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import humanoid, axe, LEATHER, CLOTH

ORC = (0.42, 0.55, 0.30)

MODEL = Model(
    id="raider_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=LEATHER, arms=ORC, forearms=ORC, hands=ORC, legs=CLOTH, head="bare", skin=ORC, build=1.1) + axe(),
    clips={
        "idle": anim.idle(stance=anim.TOOL_RELAXED, tip=0.02),
        "walk": anim.walk(stride=0.6, carry=anim.TOOL_RELAXED),
        "run": anim.run(carry=anim.TOOL_RELAXED),
        "attack": anim.attack_sword(rest=anim.TOOL_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200,
)
