"""modelc: models and animations described as code (formulas), compiled into the game's VAT format.

Model files (art/models/**/*.py) import what they need from here:
    from modelc import Model, Loft, Extrude, Cycle, Action, leg_ik, curve, merge
"""
from .math3d import curve, smoothstep, euler
from .geometry import Loft, Extrude, Mesh, Part
from .skeleton import Skeleton
from .model import Model, build
from .anim import Cycle, Action, leg_ik, merge, blend

__all__ = ["curve", "smoothstep", "euler", "Loft", "Extrude", "Mesh", "Part", "Skeleton", "Model", "build",
           "Cycle", "Action", "leg_ik", "merge", "blend"]
