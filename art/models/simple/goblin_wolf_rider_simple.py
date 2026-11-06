"""Goblin Wolf Rider (Alkars cavalry: a goblin with a dagger on a black wolf) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.cavalry import WOLF
from art.anims import cavalry as anim
from art.models.simple.kit import *

MODEL = Model(
    id="goblin_wolf_rider_simple",
    skeleton=WOLF,
    parts=horse(coat=(0.18, 0.17, 0.17), mane=(0.08, 0.08, 0.08), scale=WOLF.mount_scale)
          + mount_rider(humanoid(torso=(0.42, 0.28, 0.18), arms=(0.58, 0.62, 0.30), forearms=(0.58, 0.62, 0.30), hands=(0.58, 0.62, 0.30), legs=(0.42, 0.28, 0.18), head='bare', skin=(0.58, 0.62, 0.30)) + dagger(), WOLF),
    clips={
        "idle": anim.idle(anim.SWORD),
        "walk": anim.walk(anim.SWORD, scale=WOLF.mount_scale),
        "run": anim.run(anim.SWORD, scale=WOLF.mount_scale),
        "attack": anim.attack_sword(),
        "die": anim.die(stance=anim.SWORD),
    },
    texture=None, texture_size=256, budget_tris=2000, foot_bone="fshin.L", scale=1.0,
)
