"""Harpy (flying monster: a lean grey body, feathered wings along the arms, bird legs) — stand-in from
plain shapes. Flies: every moving clip is a wing beat. Team color: a sash."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import humanoid, arm_wings

FEATHER = (0.42, 0.36, 0.32)
SKINH = (0.62, 0.60, 0.55)

MODEL = Model(
    id="harpy_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=SKINH, arms=SKINH, forearms=SKINH, hands=SKINH, legs=FEATHER, shins=SKINH, feet=SKINH,
                   head="bare", skin=SKINH, tabard=False, sash=True) + arm_wings(FEATHER),
    clips={
        "idle": anim.fly(duration=0.9, flap=30),
        "walk": anim.fly(duration=0.6, flap=45, speed=3.0),
        "run": anim.fly(duration=0.45, flap=55, speed=6.5),
        "attack": anim.dive_attack(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1200,
)
