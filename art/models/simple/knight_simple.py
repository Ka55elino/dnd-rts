"""Knight (heavy armour, warhorse with a team-colored caparison, a lance) — stand-in from plain shapes.
Wedge (stance) = the lance couched: brace / walk_brace. Team color: tabard, caparison."""
from modelc import Model
from art.skeletons.cavalry import CAVALRY, RIDER
from art.anims import cavalry as anim
from art.models.simple.kit import humanoid, lance, horse, shift, STEEL

MODEL = Model(
    id="knight_simple",
    skeleton=CAVALRY,
    parts=horse() + shift(humanoid(torso=STEEL, legs=STEEL, shins=STEEL, arms=STEEL, forearms=STEEL, build=1.1)
                          + lance(), RIDER),
    clips={
        "idle": anim.idle(anim.LANCE),
        "walk": anim.walk(anim.LANCE),
        "run": anim.run(anim.LANCE),
        "attack": anim.attack_lance(),
        "brace": anim.brace_lance(),
        "unbrace": anim.Reverse(anim.brace_lance()),
        "walk_brace": anim.run(anim.LANCE_COUCHED),
        "die": anim.die(stance=anim.LANCE),
    },
    texture=None, texture_size=256, budget_tris=1600,
    foot_bone="fshin.L",
)
