"""Mine (Dark Elves: a buried iron pot, its lid and a fuse spike just above the dirt) — stand-in from plain shapes on a rigid skeleton."""
from modelc import Model
from art.skeletons.machines import TRAP
from art.anims import machines as anim
from art.models.simple.kit import *

MODEL = Model(
    id="mine_simple",
    skeleton=TRAP,
    parts=[cyl("mound", (0, 0, 0.0), (0, 0, 0.06), 0.55, seg=8, bone="base", color=(0.36, 0.28, 0.18)), cyl("pot", (0, 0, 0.0), (0, 0, 0.12), 0.32, seg=8, bone="base", color=DARKSTEEL), cyl("lid", (0, 0, 0.12), (0, 0, 0.16), 0.28, seg=8, bone="lid", color=(0.25, 0.26, 0.28)), cone("fuse", (0, 0, 0.16), (0, 0, 0.32), 0.04, bone="lid", color=STEEL), box("band", (0, 0.34, 0.06), (0, -0.34, 0.06), 0.34, 0.03, bone="base", team=True)],
    clips={
        "idle": anim.trap_idle(glow=0.0),
        "walk": anim.trap_idle(glow=0.0),
        "run": anim.trap_idle(glow=0.0),
        "attack": anim.trap_spring(),
        "die": anim.trap_spring(),
    },
    texture=None, texture_size=128, budget_tris=600, foot_bone="base",
)
