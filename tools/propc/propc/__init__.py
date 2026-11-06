"""propc — static models (buildings, machines) as code: plain shapes → LOD glb files for the game."""
from .shapes import *  # noqa: F401,F403


class Prop:
    """A static model: id (frontend/public/models/<id>/), parts (shapes), a one-line description."""

    def __init__(self, id, parts, doc=""):
        self.id, self.parts, self.doc = id, [p for p in parts if p is not None], doc
