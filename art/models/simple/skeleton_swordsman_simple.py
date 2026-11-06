"""Skeleton Swordsman (Alkars, risen: bones, a rusty sword) — stand-in from plain shapes."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="skeleton_swordsman_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.85, 0.83, 0.74), arms=(0.85, 0.83, 0.74), forearms=(0.85, 0.83, 0.74), hands=(0.85, 0.83, 0.74), legs=(0.85, 0.83, 0.74), shins=(0.85, 0.83, 0.74), feet=(0.85, 0.83, 0.74), head='bare', skin=(0.85, 0.83, 0.74), tabard=False, sash=True, build=0.7) + sword(0.65),
    clips={
        "idle": anim.idle(stance=anim.TOOL_RELAXED, tip=0.01),
        "walk": anim.walk(stride=0.6, carry=anim.TOOL_RELAXED),
        "run": anim.run(carry=anim.TOOL_RELAXED),
        "attack": anim.attack_sword(rest=anim.TOOL_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=0.95,
)
