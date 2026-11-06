"""Rigid machines for stand-in models (they move as a whole, with a few moving parts)."""
from modelc import Skeleton

# Airship: the hull (and its balloon) is the root; a propeller at the back.
AIRSHIP = Skeleton([
    ("hull", None, (0, 0, 1.2), (0, -1.0, 1.2)),
    ("prop", "hull", (0, 1.25, 1.5), (0, 1.45, 1.5)),
], mirror=False)

# Mortar: a low carriage (root) and a barrel that tilts.
MORTAR = Skeleton([
    ("base", None, (0, 0, 0.3), (0, -0.6, 0.3)),
    ("barrel", "base", (0, 0, 0.55), (0, -0.3, 1.1)),
], mirror=False)

# Trap: a flat thing on the ground (root) and a part that springs up when it goes off (jaws, the
# mine's lid, a rune stone).
TRAP = Skeleton([
    ("base", None, (0, 0, 0.02), (0, -0.4, 0.02)),
    ("lid", "base", (0, 0, 0.08), (0, 0, 0.3)),
], mirror=False)
