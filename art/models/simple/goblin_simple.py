"""Goblin (Alkars builder: small, yellow-green, a hammer) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="goblin_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.42, 0.28, 0.18), arms=(0.58, 0.62, 0.30), forearms=(0.58, 0.62, 0.30), hands=(0.58, 0.62, 0.30), legs=(0.42, 0.28, 0.18), head='bare', skin=(0.58, 0.62, 0.30), build=0.9) + ears((0.58, 0.62, 0.30)) + hammer(),
    clips={
        "idle": anim.idle(stance=anim.TOOL_RELAXED, tip=0.01),
        "walk": anim.walk(stride=0.6, carry=anim.TOOL_RELAXED),
        "run": anim.run(carry=anim.TOOL_RELAXED),
        "attack": anim.attack_sword(rest=anim.TOOL_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=0.72,
)
