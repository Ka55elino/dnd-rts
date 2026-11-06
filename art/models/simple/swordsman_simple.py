"""Swordsman (sword and shield) — stand-in from plain shapes. Team color: tabard, shield face."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import humanoid, sword, shield, STEEL

MODEL = Model(
    id="swordsman_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=STEEL, shins=STEEL) + sword() + shield(),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(stride=0.6),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "brace": anim.brace(),
        "unbrace": anim.unbrace(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200,
)
