"""Paralysis Trap (Dark Elves: open iron jaws with teeth, a web-silk trigger plate) — stand-in from plain shapes on a rigid skeleton."""
from modelc import Model
from art.skeletons.machines import TRAP
from art.anims import machines as anim
from art.models.simple.kit import *

MODEL = Model(
    id="paralysis_trap_simple",
    skeleton=TRAP,
    parts=[cyl("plate", (0, 0, 0.0), (0, 0, 0.04), 0.22, seg=8, bone="base", team=True), box("jaw_l", (0.30, 0.30, 0.02), (0.30, -0.30, 0.02), 0.05, 0.03, bone="base", color=DARKSTEEL), box("jaw_r", (-0.30, 0.30, 0.02), (-0.30, -0.30, 0.02), 0.05, 0.03, bone="base", color=DARKSTEEL), box("hinge", (0.30, 0.0, 0.02), (-0.30, 0.0, 0.02), 0.04, 0.04, bone="base", color=DARKSTEEL), ] + [cone(f"tooth{i}", (0.30, -0.25 + 0.1 * i, 0.04), (0.30, -0.25 + 0.1 * i, 0.16), 0.03, bone="lid", color=STEEL, mirror="x") for i in range(6)],
    clips={
        "idle": anim.trap_idle(glow=0.0),
        "walk": anim.trap_idle(glow=0.0),
        "run": anim.trap_idle(glow=0.0),
        "attack": anim.trap_spring(),
        "die": anim.trap_spring(),
    },
    texture=None, texture_size=128, budget_tris=600, foot_bone="base",
)
