"""Humanoid skeleton shared by every human-sized unit. Authoring axes: Z up, the character faces -Y,
its left side is +X. Rest pose: A-pose (arms ~45° down). Positions in meters; .R bones are mirrored."""
from modelc import Skeleton

HUMANOID_BONES = [
    # name          parent        head                  tail
    ("hips",        None,         (0, 0, 0.98),         (0, 0, 1.12)),
    ("spine",       "hips",       (0, 0, 1.12),         (0, 0, 1.30)),
    ("chest",       "spine",      (0, 0, 1.30),         (0, 0, 1.52)),
    ("neck",        "chest",      (0, 0, 1.52),         (0, 0, 1.60)),
    ("head",        "neck",       (0, 0, 1.60),         (0, 0, 1.82)),
    ("upperarm.L",  "chest",      (0.19, 0, 1.45),      (0.40, 0, 1.24)),
    ("forearm.L",   "upperarm.L", (0.40, 0, 1.24),      (0.58, 0, 1.06)),
    ("hand.L",      "forearm.L",  (0.58, 0, 1.06),      (0.65, 0, 0.99)),
    ("thigh.L",     "hips",       (0.10, 0, 0.94),      (0.11, 0, 0.52)),
    ("shin.L",      "thigh.L",    (0.11, 0, 0.52),      (0.115, 0.01, 0.09)),
    ("foot.L",      "shin.L",     (0.115, 0.01, 0.09),  (0.12, -0.14, 0.03)),
]

HUMANOID = Skeleton(HUMANOID_BONES)
