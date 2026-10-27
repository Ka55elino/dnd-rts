package sim

import (
	"testing"

	"dnd-rts/internal/fixed"
)

func TestDevSpawnAndKill(t *testing.T) {
	w := newTestWorld(t)
	hordes, buildings := len(w.Hordes), len(w.Buildings)

	w.Step([]Command{
		{Player: 1, Kind: CmdDevSpawnHorde, Name: "swordsmen", Count: 3, Target: fixed.V2i(0, 40)},
		{Player: 0, Kind: CmdDevSpawnBuilding, Name: "farm", Target: fixed.V2i(-60, 60)},
	})
	if got := len(w.Hordes); got != hordes+3 {
		t.Fatalf("hordes = %d, want %d", got, hordes+3)
	}
	if got := len(w.Buildings); got != buildings+1 {
		t.Fatalf("buildings = %d, want %d", got, buildings+1)
	}
	farm := w.Buildings[len(w.Buildings)-1]
	g := w.Nav.Grid()
	if g.WalkableAt(farm.Pos) {
		t.Fatal("farm does not block the ground")
	}

	spawned := []HordeID{w.Hordes[len(w.Hordes)-1].ID}
	w.Step([]Command{{Kind: CmdDevKill, Hordes: spawned, Buildings: []BuildingID{farm.ID}}})
	if w.Horde(spawned[0]) != nil {
		t.Error("killed horde still exists")
	}
	if w.Building(farm.ID) != nil || !g.WalkableAt(farm.Pos) {
		t.Error("removed building still exists or still blocks the ground")
	}
	// Other buildings keep blocking.
	if g.WalkableAt(w.Buildings[0].Pos) {
		t.Error("fortress stopped blocking after another building was removed")
	}
}
