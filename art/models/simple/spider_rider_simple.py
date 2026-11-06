"""Spider Rider (Dark Elves cavalry: a rider with a spear on a giant spider) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.cavalry import SPIDER
from art.anims import cavalry as anim
from art.models.simple.kit import *

MODEL = Model(
    id="spider_rider_simple",
    skeleton=SPIDER,
    parts=spider_body((0.18, 0.15, 0.20), SPIDER.mount_scale)
          + mount_rider(humanoid(torso=(0.12, 0.10, 0.14), arms=(0.35, 0.15, 0.42), legs=(0.12, 0.10, 0.14), head='helmet', head_color=(0.25, 0.22, 0.28), skin=(0.42, 0.38, 0.48)) + sword(), SPIDER),
    clips={
        "idle": anim.idle(anim.SWORD),
        "walk": anim.walk(anim.SWORD, scale=SPIDER.mount_scale),
        "run": anim.run(anim.SWORD, scale=SPIDER.mount_scale),
        "attack": anim.attack_sword(),
        "die": anim.die(stance=anim.SWORD),
    },
    texture=None, texture_size=256, budget_tris=2000, foot_bone="fshin.L", scale=1.0,
)
