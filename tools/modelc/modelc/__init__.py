"""modelc: models and animations described as code (formulas), compiled into the game's VAT format.

Model files (art/models/**/*.py) import what they need from here:
    from modelc import Model, Loft, Extrude, Cycle, Action, leg_ik, curve, merge
"""
from .math3d import curve, smoothstep, euler, cubic_bezier
from .geometry import Loft, Extrude, Mesh, Part
from .skeleton import Skeleton
from .model import Model, build
from .anim import Cycle, Action, Motion, Reverse, leg_ik, merge, blend, aim, point_on, limb_ik, plant_foot, place_hand

__all__ = ["curve", "smoothstep", "euler", "cubic_bezier", "Loft", "Extrude", "Mesh", "Part", "Skeleton", "Model", "build",
           "Cycle", "Action", "Motion", "Reverse", "leg_ik", "merge", "blend",
           "aim", "point_on", "limb_ik", "plant_foot", "place_hand"]
