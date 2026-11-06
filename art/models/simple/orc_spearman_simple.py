"""Orc Spearman (Alkars: spear and shield, spear wall) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="orc_spearman_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.15, 0.14, 0.15), arms=(0.42, 0.55, 0.30), forearms=(0.15, 0.14, 0.15), hands=(0.42, 0.55, 0.30), legs=(0.15, 0.14, 0.15), head='helmet', head_color=(0.15, 0.14, 0.15), skin=(0.42, 0.55, 0.30), build=1.2) + spear() + shield(),
    clips={
        "idle": anim.idle(stance=anim.SPEAR_RELAXED, tip=0.01),
        "walk": anim.walk(stride=0.6, carry=anim.SPEAR_CARRY),
        "run": anim.run(carry=anim.SPEAR_CARRY),
        "attack": anim.attack_spear(),
        "brace": anim.brace(rest=anim.SPEAR_RELAXED, target=anim.SPEAR_BRACE),
        "unbrace": anim.unbrace(rest=anim.SPEAR_RELAXED, target=anim.SPEAR_BRACE),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.05,
)
