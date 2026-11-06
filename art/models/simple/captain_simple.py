"""Captain hero (test hero: a human officer in steel with a gold helmet, sword and
shield) — stand-in from plain shapes. Team color: tabard, shield face."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import humanoid, sword, shield, STEEL, GOLD

MODEL = Model(
    id="captain_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=STEEL, arms=STEEL, forearms=GOLD, shins=STEEL, head="helmet", head_color=GOLD, build=1.1)
          + sword(0.85) + shield(),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(stride=0.6),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200,
)
