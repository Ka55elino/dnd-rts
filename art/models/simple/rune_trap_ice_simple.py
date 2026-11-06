"""Rune Trap, ice (Dwarves: a flat carved stone with a glowing ice rune above it) — stand-in from plain shapes on a rigid skeleton."""
from modelc import Model
from art.skeletons.machines import TRAP
from art.anims import machines as anim
from art.models.simple.kit import *

MODEL = Model(
    id="rune_trap_ice_simple",
    skeleton=TRAP,
    parts=[cyl("stone", (0, 0, 0.0), (0, 0, 0.08), 0.45, seg=6, bone="base", color=(0.50, 0.50, 0.52)), cyl("inlay", (0, 0, 0.08), (0, 0, 0.09), 0.30, seg=6, bone="base", color=(0.55, 0.80, 0.98)), box("rim", (0, 0.42, 0.04), (0, -0.42, 0.04), 0.46, 0.02, bone="base", team=True), ball("rune", (0, 0, 0.18), 0.11, bone="lid", color=(0.55, 0.80, 0.98))],
    clips={
        "idle": anim.trap_idle(glow=0.05),
        "walk": anim.trap_idle(glow=0.05),
        "run": anim.trap_idle(glow=0.05),
        "attack": anim.trap_spring(),
        "die": anim.trap_spring(),
    },
    texture=None, texture_size=128, budget_tris=600, foot_bone="base",
)
