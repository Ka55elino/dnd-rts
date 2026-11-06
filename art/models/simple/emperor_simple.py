"""Emperor hero (cavalry commander: gold-trimmed armour, a crown, a sword, a white horse with a
team-colored caparison) — stand-in from plain shapes."""
from modelc import Model
from art.skeletons.cavalry import CAVALRY, RIDER
from art.anims import cavalry as anim
from art.models.simple.kit import humanoid, sword, horse, shift, STEEL, GOLD, WHITE

MODEL = Model(
    id="emperor_simple",
    skeleton=CAVALRY,
    parts=horse(coat=WHITE, mane=(0.75, 0.72, 0.66)) + shift(
        humanoid(torso=GOLD, legs=STEEL, shins=STEEL, arms=STEEL, forearms=GOLD, head="crown", build=1.1)
        + sword(0.9), RIDER),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(),
        "run": anim.run(),
        "attack": anim.attack_sword(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1600,
    foot_bone="fshin.L",
)
