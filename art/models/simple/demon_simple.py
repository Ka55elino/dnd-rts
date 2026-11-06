"""Demon (summoned by the Alkar Dark Mage: a huge red horned brute) — stand-in from plain shapes."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="demon_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.55, 0.10, 0.08), arms=(0.55, 0.10, 0.08), forearms=(0.30, 0.06, 0.05), hands=(0.30, 0.06, 0.05), legs=(0.30, 0.06, 0.05), shins=(0.55, 0.10, 0.08), feet=(0.20, 0.05, 0.04), head='horned', head_color=(0.30, 0.06, 0.05), skin=(0.55, 0.10, 0.08), tabard=False, sash=True, build=1.4) + club(color=(0.20, 0.18, 0.18)),
    clips={
        "idle": anim.idle(stance=anim.TOOL_RELAXED, tip=0.01),
        "walk": anim.walk(stride=0.6, carry=anim.TOOL_RELAXED),
        "run": anim.run(carry=anim.TOOL_RELAXED),
        "attack": anim.attack_sword(rest=anim.TOOL_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.7,
)
