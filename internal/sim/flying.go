package sim

// Flying units (docs/FACTIONS.md §1): they move in a straight line over
// terrain, water and buildings, don't collide with ground units and can only
// be hit by attacks that target "air" — until something pulls them down
// (status "ground"), when they count as ground units for a while.

// Airborne: a flying horde that is not pulled down right now. Only attacks
// that target "air" hit it; it ignores terrain and ground units.
func (h *Horde) Airborne() bool { return h.Unit.Flying && h.groundTicks == 0 }
