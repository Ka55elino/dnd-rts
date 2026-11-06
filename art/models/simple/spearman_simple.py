"""Spearman (long spear and shield, medium armour, a plate on the forearm) — stand-in from plain shapes.
Team color: tabard, shield face. Spear wall = brace."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import humanoid, spear, shield, STEEL, LEATHER, CLOTH

MODEL = Model(
    id="spearman_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=LEATHER, arms=CLOTH, forearms=STEEL, shins=LEATHER) + spear() + shield(),
    clips={
        "idle": anim.idle(stance=anim.SPEAR_RELAXED, tip=0.01),
        "walk": anim.walk(stride=0.6, carry=anim.SPEAR_CARRY),
        "run": anim.run(carry=anim.SPEAR_CARRY),
        "attack": anim.attack_spear(),
        "brace": anim.brace(rest=anim.SPEAR_RELAXED, target=anim.SPEAR_BRACE),
        "unbrace": anim.unbrace(rest=anim.SPEAR_RELAXED, target=anim.SPEAR_BRACE),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200,
)
