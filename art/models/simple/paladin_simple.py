"""Paladin hero (support warrior: white-and-gold plate, mace and shield) — stand-in from plain shapes.
Team color: tabard, shield face."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import humanoid, mace, shield, WHITE, GOLD, STEEL

MODEL = Model(
    id="paladin_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=WHITE, arms=STEEL, forearms=STEEL, legs=STEEL, shins=WHITE, head="helmet", head_color=GOLD,
                   build=1.1) + mace() + shield(),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(stride=0.6),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200,
)
