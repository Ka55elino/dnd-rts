package sim

import (
	"fmt"
	"slices"

	"dnd-rts/internal/fixed"
)

// PlayerSetup describes one player in a match.
type PlayerSetup struct {
	ID      PlayerID `json:"id"`
	Faction string   `json:"faction"`
}

// SetupSkirmish places each player's fortress and the map's neutral outposts,
// then spawns every horde of each player's faction near their start position,
// lined up and facing the map center.
func SetupSkirmish(w *World, mapID string, players []PlayerSetup) error {
	m, ok := w.Catalog.Maps[mapID]
	if !ok {
		return fmt.Errorf("sim: unknown map %q", mapID)
	}
	if len(players) > len(m.Players) {
		return fmt.Errorf("sim: map %q has %d start positions, got %d players", mapID, len(m.Players), len(players))
	}

	for i, p := range players {
		faction, ok := w.Catalog.Factions[p.Faction]
		if !ok {
			return fmt.Errorf("sim: unknown faction %q", p.Faction)
		}

		slot := m.Players[i].Fortress
		f, err := w.SpawnBuilding(p.ID, faction.Fortress, slot.Pos, slot.Facing())
		if err != nil {
			return fmt.Errorf("map %q player %d fortress: %w", mapID, i, err)
		}
		w.spawnWalls(f)
	}
	sorted := slices.Clone(players)
	slices.SortFunc(sorted, func(a, b PlayerSetup) int { return int(a.ID) - int(b.ID) })
	for _, p := range sorted {
		w.Players = append(w.Players, p.ID)
		w.factions = append(w.factions, p.Faction)
	}
	w.initEconomy()
	for i, o := range m.Outposts {
		if _, err := w.SpawnBuilding(Neutral, o.Building, o.Pos, o.Facing()); err != nil {
			return fmt.Errorf("map %q outpost %d: %w", mapID, i, err)
		}
	}

	gap := fixed.FromInt(14)
	for i, p := range players {
		faction, ok := w.Catalog.Factions[p.Faction]
		if !ok {
			return fmt.Errorf("sim: unknown faction %q", p.Faction)
		}
		start := m.Players[i].Start
		facing := fixed.Vec2{}.Sub(start).Angle() // towards the center
		if start.IsZero() {
			facing = 0
		}
		forward := fixed.FromAngle(facing)
		right := fixed.Vec2{X: forward.Y, Y: -forward.X}

		army := faction.Start
		if len(army) == 0 {
			army = faction.Hordes
		}
		n := len(army)
		for j, hordeType := range army {
			offset := gap.MulInt(2*j - (n - 1)).DivInt(2)
			pos := start.Add(right.Scale(offset)).Add(forward.Scale(fixed.FromInt(12)))
			if _, err := w.SpawnHorde(p.ID, hordeType, pos, facing); err != nil {
				return err
			}
		}
	}
	return nil
}
