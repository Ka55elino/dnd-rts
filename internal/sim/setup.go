package sim

import (
	"cmp"
	"fmt"
	"slices"

	"dnd-rts/internal/data"
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
	m, _, err := setupBases(w, mapID, players)
	if err != nil {
		return err
	}
	gap := fixed.FromInt(14)
	for i, p := range players {
		faction := w.Catalog.Factions[p.Faction]
		start := m.Players[i].Start
		facing, forward, right := startFrame(start)

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
	w.SetFog(true)
	return nil
}

// SetupPlayground is a test field: the bases of a skirmish, then every player gets one finished
// building of each type its faction can build — inside the base around the keep (or, when it is
// full, in rows further out), defensive towers spread along the wall — and one horde of each type
// it can train, in rows of five outside the base wall, between the base and the map center.
func SetupPlayground(w *World, mapID string, players []PlayerSetup) error {
	m, forts, err := setupBases(w, mapID, players)
	if err != nil {
		return err
	}
	for i, p := range players {
		faction := w.Catalog.Factions[p.Faction]
		fort := forts[i]
		facing, forward, right := startFrame(fort.Pos)
		// Outside the wall: the base radius (or, without a base, the start position's distance) + a margin.
		out := fixed.FromInt(20)
		if fort.Def.Base != nil {
			out = fort.Def.Base.Radius + fixed.FromInt(20)
		} else if d := m.Players[i].Start.Sub(fort.Pos).Len(); d > 0 {
			out = d
		}
		front := fort.Pos.Add(forward.Scale(out))

		// Biggest first, so the large ones get the room near the keep.
		ids := slices.Clone(faction.Buildings)
		slices.SortStableFunc(ids, func(a, b string) int {
			return cmp.Compare(footprintSize(w.Catalog.Buildings[b]), footprintSize(w.Catalog.Buildings[a]))
		})
		// A finished wall across the front of the base, with a gatehouse in the middle (for the towers).
		w.playgroundWall(p.ID, fort, faction, forward, right)
		towers := 0
		for _, id := range ids {
			def, ok := w.Catalog.Buildings[id]
			if !ok || def.Kind == data.BuildingWall || def.Kind == data.BuildingGate {
				continue
			}
			if def.Allows(data.PlacementTower) {
				w.placeFinishedTower(p.ID, fort, def, towers)
				towers++
				continue
			}
			w.placeFinishedFree(fort, p.ID, def, front, forward, right, facing)
		}

		for j, hordeType := range faction.Hordes {
			row, col := j/5, j%5
			pos := front.Add(right.Scale(fixed.FromInt(14*col - 28))).Add(forward.Scale(fixed.FromInt(14 * row)))
			if _, err := w.SpawnHorde(p.ID, hordeType, pos, facing); err != nil {
				return err
			}
		}
	}
	return nil
}

// playgroundWall: a finished line of the faction's wall 30 m in front of the keep, 14 segments
// long, with its gatehouse in the middle.
func (w *World) playgroundWall(player PlayerID, fort *Building, faction *data.Faction, forward, right fixed.Vec2) {
	var wall, gate *data.Building
	for _, id := range faction.Buildings {
		switch b := w.Catalog.Buildings[id]; b.Kind {
		case data.BuildingWall:
			wall = b
		case data.BuildingGate:
			gate = b
		}
	}
	if wall == nil {
		return
	}
	centre := fort.Pos.Add(forward.Scale(fixed.FromInt(30)))
	half := right.Scale(wall.Footprint.X.MulInt(7))
	segs := w.layWall(player, wall, centre.Sub(half), centre.Add(half), true, false)
	if gate == nil || len(segs) < 5 {
		return
	}
	r := w.Walls[len(w.Walls)-1]
	span := max((gate.Footprint.X.Div(r.segLen) + fixed.One/2).Int(), 1)
	w.putGate(player, gate, r, len(segs)/2, span)
}

// placeFinishedTower puts a finished tower of type def on the fortress's wall: the k-th tower goes
// a step further around the ring.
func (w *World) placeFinishedTower(player PlayerID, fort *Building, def *data.Building, k int) {
	var segs []*Building
	for _, b := range w.Buildings {
		if b.Owner == player && b.Def.Kind == data.BuildingWall && b.Alive() && b.DistTo(fort.Pos) < fixed.FromInt(200) {
			segs = append(segs, b)
		}
	}
	if len(segs) == 0 {
		return
	}
	step := max(len(segs)/12, 1)
	for t := 0; t < len(segs); t++ {
		s := segs[(k*step*2+t*step+step/2)%len(segs)]
		if spot, facing, ok := w.towerSpotAny(player, def, s.Pos); ok {
			b := w.placeBuilding(player, def.ID, spot, facing)
			b.onWall = true
			return
		}
	}
}

// towerSpotAny is TowerSpot without the requirement check (the test field has no tech yet).
func (w *World) towerSpotAny(player PlayerID, def *data.Building, pos fixed.Vec2) (fixed.Vec2, fixed.Fixed, bool) {
	free := *def
	free.Requires = nil
	return w.TowerSpot(player, &free, pos)
}

// placeFinishedFree puts a finished building of type def inside the base (rings around the keep,
// nearest first), or failing that in rows in front of the base.
func (w *World) placeFinishedFree(fort *Building, player PlayerID, def *data.Building, front, forward, right fixed.Vec2, facing fixed.Fixed) {
	inner := fixed.FromInt(60)
	if l := fort.Def.Base; l != nil {
		inner = l.Radius - fixed.FromInt(8)
	}
	for r := fixed.FromInt(16); r <= inner; r += fixed.FromInt(4) {
		n := max(8, r.MulInt(2).Int()/3)
		for k := 0; k < n; k++ {
			a := fixed.Pi.MulInt(2 * k).DivInt(n)
			pos := fort.Pos.Add(fixed.FromAngle(a).Scale(r))
			face := fort.Pos.Sub(pos).Angle() // the front towards the keep
			if w.fitsFree(def, pos, face, true) {
				if _, err := w.SpawnBuilding(player, def.ID, pos, face); err == nil {
					return
				}
			}
		}
	}
	for k := 0; k < 40; k++ {
		pos := front.Add(forward.Scale(fixed.FromInt(62 + 10*(k/8)))).Add(right.Scale(fixed.FromInt(14 * (k%8 - 4))))
		if w.fitsFree(def, pos, facing, true) {
			if _, err := w.SpawnBuilding(player, def.ID, pos, facing); err == nil {
				return
			}
		}
	}
}

// setupBases places each player's fortress with its walls, registers the players and their
// factions, starts the economy and places the map's neutral outposts. Returns the map and the
// fortresses in player order.
func setupBases(w *World, mapID string, players []PlayerSetup) (*data.Map, []*Building, error) {
	m, ok := w.Catalog.Maps[mapID]
	if !ok {
		return nil, nil, fmt.Errorf("sim: unknown map %q", mapID)
	}
	if len(players) > len(m.Players) {
		return nil, nil, fmt.Errorf("sim: map %q has %d start positions, got %d players", mapID, len(m.Players), len(players))
	}

	forts := make([]*Building, len(players))
	for i, p := range players {
		faction, ok := w.Catalog.Factions[p.Faction]
		if !ok {
			return nil, nil, fmt.Errorf("sim: unknown faction %q", p.Faction)
		}

		slot := m.Players[i].Fortress
		f, err := w.SpawnBuilding(p.ID, faction.Fortress, slot.Pos, slot.Facing())
		if err != nil {
			return nil, nil, fmt.Errorf("map %q player %d fortress: %w", mapID, i, err)
		}
		w.spawnWalls(f)
		forts[i] = f
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
			return nil, nil, fmt.Errorf("map %q outpost %d: %w", mapID, i, err)
		}
	}
	w.spawnForests(m.Forests)
	return m, forts, nil
}

// startFrame: the facing towards the map center from a start position, and the forward / right axes.
func startFrame(start fixed.Vec2) (facing fixed.Fixed, forward, right fixed.Vec2) {
	facing = fixed.Vec2{}.Sub(start).Angle()
	if start.IsZero() {
		facing = 0
	}
	forward = fixed.FromAngle(facing)
	right = fixed.Vec2{X: forward.Y, Y: -forward.X}
	return facing, forward, right
}

// footprintSize: the larger side of a building's footprint (0 for an unknown type).
func footprintSize(def *data.Building) fixed.Fixed {
	if def == nil {
		return 0
	}
	return def.Footprint.X.Max(def.Footprint.Y)
}
