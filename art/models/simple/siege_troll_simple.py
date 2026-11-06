"""Siege Troll (Alkars: a huge grey troll with a club; throws rocks) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="siege_troll_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.45, 0.48, 0.42), arms=(0.45, 0.48, 0.42), forearms=(0.45, 0.48, 0.42), hands=(0.45, 0.48, 0.42), legs=(0.30, 0.25, 0.20), head='bare', skin=(0.45, 0.48, 0.42), build=1.5, tabard=False, sash=True) + club(),
    clips={
        "idle": anim.idle(stance=anim.TOOL_RELAXED, tip=0.01),
        "walk": anim.walk(stride=0.6, carry=anim.TOOL_RELAXED),
        "run": anim.run(carry=anim.TOOL_RELAXED),
        "attack": anim.attack_sword(rest=anim.TOOL_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.7,
)
