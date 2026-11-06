"""Griffon Rider (flying cavalry: a rider with a sword on a griffon — tawny body, white head with a beak,
wings) — stand-in from plain shapes. Team color: tabard, the griffon's saddle cloth."""
from modelc import Model
from art.skeletons.cavalry import CAVALRY
from art.anims import cavalry as anim
from art.models.simple.kit import humanoid, sword, horse, wings, mount_rider, STEEL

TAWNY = (0.62, 0.48, 0.28)

MODEL = Model(
    id="griffon_rider_simple",
    skeleton=CAVALRY,
    parts=horse(coat=TAWNY, mane=(0.90, 0.88, 0.82), beak=(0.85, 0.70, 0.25)) + wings(TAWNY)
          + mount_rider(humanoid(torso=STEEL, legs=STEEL, shins=STEEL) + sword(), CAVALRY),
    clips={
        "idle": anim.fly(duration=0.9, flap=30),
        "walk": anim.fly(duration=0.7, flap=40, ground_speed=3.0),
        "run": anim.fly(duration=0.55, flap=50, ground_speed=8.0),
        "attack": anim.attack_sword(),
        "die": anim.die(),
    },
    texture=None, texture_size=256, budget_tris=1600, foot_bone="fshin.L",
)
