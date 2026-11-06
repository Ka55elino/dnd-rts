"""Berserker (Alkars: a bare-chested orc with a great axe) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="berserker_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.42, 0.55, 0.30), arms=(0.42, 0.55, 0.30), forearms=(0.42, 0.55, 0.30), hands=(0.42, 0.55, 0.30), legs=(0.50, 0.08, 0.08), head='bare', skin=(0.42, 0.55, 0.30), build=1.3) + greataxe(),
    clips={
        "idle": anim.idle(stance=anim.TOOL_RELAXED, tip=0.01),
        "walk": anim.walk(stride=0.6, carry=anim.TOOL_RELAXED),
        "run": anim.run(carry=anim.TOOL_RELAXED),
        "attack": anim.attack_sword(rest=anim.TOOL_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.08,
)
