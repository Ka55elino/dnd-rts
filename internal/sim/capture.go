package sim

import (
	"slices"

	"dnd-rts/internal/fixed"
)

// Capturing reports who is taking the outpost (Neutral = nobody) and the progress in [0, 1].
func (b *Building) Capturing() (PlayerID, fixed.Fixed) {
	if b.Def.Capture == nil || b.captureBy == Neutral {
		return Neutral, 0
	}
	return b.captureBy, fixed.FromInt(b.captureTicks).DivInt(b.captureTicksNeeded())
}

func (b *Building) captureTicksNeeded() int { return max(1, cooldownTicks(b.Def.Capture.Time)) }

// updateCapture advances captures of neutral outposts: progress grows while
// units of exactly one player stand in range, pauses while contested and
// drains when nobody is there (or another player arrives alone).
func (w *World) updateCapture() {
	var present []PlayerID
	for _, b := range w.Buildings {
		c := b.Def.Capture
		if c == nil || b.Owner != Neutral {
			continue
		}
		present = present[:0]
		reach := c.Radius + fixed.FromInt(2) // + bucket slack; exact check below
		search := reach + b.Def.Footprint.X.Max(b.Def.Footprint.Y).DivInt(2)
		w.forEachSoldierNear(b.Pos, search, func(s *Soldier) {
			p := s.Horde.Owner
			if !s.Alive() || p == Neutral || slices.Contains(present, p) || b.DistTo(s.Pos) > c.Radius {
				return
			}
			present = append(present, p)
		})

		switch {
		case len(present) == 1 && (b.captureBy == present[0] || b.captureBy == Neutral):
			b.captureBy = present[0]
			b.captureTicks++
			if b.captureTicks >= b.captureTicksNeeded() {
				b.Owner = b.captureBy
				w.awardCapture(b, b.Owner)
				b.captureBy, b.captureTicks = Neutral, 0
			}
		case len(present) > 1:
			// Contested: hold.
		default:
			// Nobody, or someone else alone: the previous progress drains first.
			if b.captureTicks > 0 {
				b.captureTicks--
			}
			if b.captureTicks == 0 {
				b.captureBy = Neutral
			}
		}
	}
}
