"""Warbeast (the warlord's beast form: a huge dark beast on four legs, no rider) — stand-in from plain
shapes, 1.3x size. Team color: a saddle cloth."""
from modelc import Model
from art.skeletons.cavalry import CAVALRY
from art.anims import cavalry as anim
from art.models.simple.kit import horse

MODEL = Model(
    id="warbeast_simple",
    skeleton=CAVALRY,
    parts=horse(coat=(0.22, 0.18, 0.16), mane=(0.08, 0.07, 0.06)),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(),
        "run": anim.run(),
        "attack": anim.beast_attack(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1600, foot_bone="fshin.L", scale=1.3,
)
