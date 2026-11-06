"""Ent (summoned by the Elf King: a walking tree with leafy shoulders) — stand-in from plain shapes."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="ent_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.38, 0.27, 0.15), arms=(0.38, 0.27, 0.15), forearms=(0.38, 0.27, 0.15), hands=(0.30, 0.45, 0.20), legs=(0.32, 0.22, 0.12), shins=(0.32, 0.22, 0.12), feet=(0.30, 0.20, 0.10), head='bare', skin=(0.38, 0.27, 0.15), tabard=False, sash=True, build=1.4) + [ball('crown_leaves', (0, 0.02, 1.85), 0.28, bone='head', color=(0.25, 0.50, 0.20)), ball('leaves', (0.24, 0, 1.55), 0.16, bone='upperarm.L', color=(0.25, 0.50, 0.20), mirror='x')],
    clips={
        "idle": anim.idle(stance=anim.FIST_RELAXED, tip=0.0),
        "walk": anim.walk(stride=0.5, carry=anim.FIST_RELAXED),
        "run": anim.run(carry=anim.FIST_RELAXED),
        "attack": anim.attack_sword(rest=anim.FIST_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=1.8,
)
