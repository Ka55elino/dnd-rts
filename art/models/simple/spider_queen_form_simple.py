"""Spider Queen in spider form (a giant purple-black spider) — stand-in from plain shapes."""
from modelc import Model
from art.skeletons.cavalry import SPIDER
from art.anims import cavalry as anim
from art.models.simple.kit import *

MODEL = Model(
    id="spider_queen_form_simple",
    skeleton=SPIDER,
    parts=spider_body((0.30, 0.12, 0.38), SPIDER.mount_scale) + [box("saddle", (0, 0.10, 1.0), (0, -0.10, 1.0), 0.12, 0.03, bone="body", team=True)],
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(scale=SPIDER.mount_scale),
        "run": anim.run(scale=SPIDER.mount_scale),
        "attack": anim.beast_attack(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, foot_bone="fshin.L", scale=1.7,
)
