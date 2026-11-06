"""Bear Rider (Dwarves cavalry: a dwarf with an axe on a war bear) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.cavalry import BEAR
from art.anims import cavalry as anim
from art.models.simple.kit import *

MODEL = Model(
    id="bear_rider_simple",
    skeleton=BEAR,
    parts=horse(coat=(0.32, 0.22, 0.14), mane=(0.20, 0.14, 0.09), scale=BEAR.mount_scale)
          + mount_rider(humanoid(torso=STEEL, arms=STEEL, legs=(0.22, 0.30, 0.48), head='helmet', skin=(0.80, 0.62, 0.50), build=1.35) + beard((0.50, 0.28, 0.14)) + axe(), BEAR),
    clips={
        "idle": anim.idle(anim.SWORD),
        "walk": anim.walk(anim.SWORD, scale=BEAR.mount_scale),
        "run": anim.run(anim.SWORD, scale=BEAR.mount_scale),
        "attack": anim.attack_sword(),
        "die": anim.die(stance=anim.SWORD),
    },
    texture=None, texture_size=256, budget_tris=2000, foot_bone="fshin.L", scale=1.0,
)
