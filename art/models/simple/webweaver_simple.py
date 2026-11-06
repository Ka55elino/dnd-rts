"""Webweaver (Dark Elves builder: a hooded figure with a hook) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="webweaver_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.12, 0.10, 0.14), arms=(0.35, 0.15, 0.42), legs=(0.12, 0.10, 0.14), head='hood', head_color=(0.35, 0.15, 0.42), skin=(0.42, 0.38, 0.48)) + hammer(),
    clips={
        "idle": anim.idle(stance=anim.TOOL_RELAXED, tip=0.01),
        "walk": anim.walk(stride=0.6, carry=anim.TOOL_RELAXED),
        "run": anim.run(carry=anim.TOOL_RELAXED),
        "attack": anim.attack_sword(rest=anim.TOOL_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.0,
)
