"""Dwarf Engineer hero (Dwarves: a leather coat, goggles, a big wrench-hammer) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="engineer_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.40, 0.28, 0.16), arms=(0.40, 0.28, 0.16), legs=(0.22, 0.30, 0.48), head='helmet', head_color=(0.70, 0.55, 0.25), skin=(0.80, 0.62, 0.50), build=1.35) + beard((0.60, 0.25, 0.10)) + hammer(),
    clips={
        "idle": anim.idle(stance=anim.TOOL_RELAXED, tip=0.01),
        "walk": anim.walk(stride=0.6, carry=anim.TOOL_RELAXED),
        "run": anim.run(carry=anim.TOOL_RELAXED),
        "attack": anim.attack_sword(rest=anim.TOOL_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=0.8,
)
