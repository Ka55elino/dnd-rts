"""Warlord hero (warband: a big green warrior, dark armour, a horned helmet, an axe and a shield) — stand-in
from plain shapes, 1.2x size. Team color: tabard, shield face."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import humanoid, axe, shield, DARKSTEEL, LEATHER

ORC = (0.42, 0.55, 0.30)

MODEL = Model(
    id="warlord_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=DARKSTEEL, arms=ORC, forearms=DARKSTEEL, hands=ORC, legs=LEATHER, shins=DARKSTEEL,
                   head="horned", head_color=DARKSTEEL, skin=ORC, build=1.25) + axe() + shield(),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(stride=0.6),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200, scale=1.2,
)
