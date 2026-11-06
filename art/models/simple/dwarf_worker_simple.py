"""Worker (Dwarves: no armour, a leather apron, a hammer) — stand-in from plain shapes (art/models/simple/kit.py)."""
from modelc import Model
from art.skeletons.humanoid import HUMANOID
from art.anims import humanoid as anim
from art.models.simple.kit import *

MODEL = Model(
    id="dwarf_worker_simple",
    skeleton=HUMANOID,
    parts=humanoid(torso=(0.55, 0.45, 0.32), arms=(0.55, 0.45, 0.32), legs=(0.35, 0.28, 0.20), head='bare', skin=(0.80, 0.62, 0.50), build=1.35) + beard((0.50, 0.28, 0.14)) + hammer(),
    clips={
        "idle": anim.idle(stance=anim.TOOL_RELAXED, tip=0.01),
        "walk": anim.walk(stride=0.6, carry=anim.TOOL_RELAXED),
        "run": anim.run(carry=anim.TOOL_RELAXED),
        "attack": anim.attack_sword(rest=anim.TOOL_RELAXED),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1400, scale=0.78,
)
