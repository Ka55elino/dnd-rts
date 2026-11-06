"""Tiger Rider (High Elves cavalry: an elf with a sword on a big striped tiger) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.cavalry import TIGER
from art.anims import cavalry as anim
from art.models.simple.kit import *

MODEL = Model(
    id="tiger_rider_simple",
    skeleton=TIGER,
    parts=horse(coat=(0.85, 0.50, 0.15), mane=(0.10, 0.08, 0.06), scale=TIGER.mount_scale)
          + mount_rider(humanoid(torso=(0.80, 0.82, 0.86), arms=(0.22, 0.45, 0.30), legs=(0.22, 0.45, 0.30), head='helmet', head_color=(0.80, 0.82, 0.86), skin=(0.86, 0.75, 0.64)) + sword(), TIGER),
    clips={
        "idle": anim.idle(anim.SWORD),
        "walk": anim.walk(anim.SWORD, scale=TIGER.mount_scale),
        "run": anim.run(anim.SWORD, scale=TIGER.mount_scale),
        "attack": anim.attack_sword(),
        "die": anim.die(stance=anim.SWORD),
    },
    texture=None, texture_size=256, budget_tris=2000, foot_bone="fshin.L", scale=1.0,
)
