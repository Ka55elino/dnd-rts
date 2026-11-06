"""Wolf Rider (warband cavalry: a green raider with an axe on a big grey wolf) — stand-in from plain
shapes. Team color: tabard, the wolf's saddle cloth."""
from modelc import Model
from art.skeletons.cavalry import WOLF
from art.anims import cavalry as anim
from art.models.simple.kit import humanoid, axe, horse, mount_rider, LEATHER, CLOTH

ORC = (0.42, 0.55, 0.30)
FUR = (0.45, 0.45, 0.47)

MODEL = Model(
    id="wolf_rider_simple",
    skeleton=WOLF,
    parts=horse(coat=FUR, mane=(0.25, 0.25, 0.27), scale=WOLF.mount_scale)
          + mount_rider(humanoid(torso=LEATHER, arms=ORC, forearms=ORC, hands=ORC, legs=CLOTH, head="bare", skin=ORC)
                        + axe(), WOLF),
    clips={
        "idle": anim.idle(),
        "walk": anim.walk(scale=WOLF.mount_scale),
        "run": anim.run(scale=WOLF.mount_scale),
        "attack": anim.attack_sword(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1600, foot_bone="fshin.L",
)
