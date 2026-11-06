"""Dwarf King hero (Dwarves: the King under the Mountain — gold plate, a crown, a hammer and a shield) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="dwarf_king_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=GOLD, arms=STEEL, forearms=GOLD, legs=(0.22, 0.30, 0.48), shins=STEEL, head='crown', skin=(0.80, 0.62, 0.50), build=1.35) + beard((0.85, 0.85, 0.82)) + mace() + shield(),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(stride=0.6),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "brace": anim.brace(),
        "unbrace": anim.unbrace(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=0.82,
)
