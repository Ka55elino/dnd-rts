"""Poison Trap (Dark Elves: a cracked flask half buried, green ooze around it) — stand-in from plain shapes on a rigid skeleton."""
from modelc import Model
from art.skeletons.machines import TRAP
from art.anims import machines as anim
from art.models.simple.kit import *

MODEL = Model(
    id="poison_trap_simple",
    skeleton=TRAP,
    parts=[cyl("mound", (0, 0, 0.0), (0, 0, 0.05), 0.55, seg=8, bone="base", color=(0.36, 0.28, 0.18)), cyl("ooze", (0, 0, 0.04), (0, 0, 0.06), 0.45, seg=8, bone="base", color=(0.30, 0.62, 0.18)), ball("flask", (0, 0, 0.16), 0.16, bone="lid", color=(0.40, 0.75, 0.30)), cyl("neck", (0, 0, 0.28), (0, 0, 0.38), 0.05, bone="lid", color=(0.40, 0.75, 0.30)), cyl("cork", (0, 0, 0.38), (0, 0, 0.43), 0.06, bone="lid", team=True)],
    clips={
        "idle": anim.trap_idle(glow=0.0),
        "walk": anim.trap_idle(glow=0.0),
        "run": anim.trap_idle(glow=0.0),
        "attack": anim.trap_spring(),
        "die": anim.trap_spring(),
    },
    texture=None, texture_size=128, budget_tris=600, foot_bone="base",
)
