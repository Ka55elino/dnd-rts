package devconsole

import (
	"os"
	"strings"
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/fixed"
	"dnd-rts/internal/sim"
	"dnd-rts/internal/terrain"
)

type fakeControl struct {
	paused bool
	steps  int
	speed  float64
	local  sim.PlayerID
}

func (f *fakeControl) Paused() bool        { return f.paused }
func (f *fakeControl) SetPaused(p bool)    { f.paused = p }
func (f *fakeControl) Step(n int)          { f.steps += n }
func (f *fakeControl) Speed() float64      { return f.speed }
func (f *fakeControl) SetSpeed(v float64)  { f.speed = v }
func (f *fakeControl) Local() sim.PlayerID { return f.local }
func (f *fakeControl) SetLocal(p sim.PlayerID) error {
	f.local = p
	return nil
}

func world(t *testing.T) *sim.World {
	t.Helper()
	cat, err := data.Load(os.DirFS("../../content"))
	if err != nil {
		t.Fatal(err)
	}
	hf, err := terrain.ForMap(os.DirFS("../../content"), cat.Maps["plains"])
	if err != nil {
		t.Fatal(err)
	}
	w := sim.NewWorld(cat, hf)
	if err := sim.SetupSkirmish(w, "plains", []sim.PlayerSetup{{ID: 0, Faction: "empire"}, {ID: 1, Faction: "warband"}}); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestSpawnCommand(t *testing.T) {
	w := world(t)
	r := Run(w, &fakeControl{speed: 1}, Context{Cursor: fixed.V2i(0, 40)}, "spawn swordsmen 2 enemy")
	if r.Err || len(r.Commands) != 1 {
		t.Fatalf("result %+v", r)
	}
	c := r.Commands[0]
	if c.Kind != sim.CmdDevSpawnHorde || c.Player != 1 || c.Count != 2 || c.Name != "swordsmen" {
		t.Fatalf("command %+v", c)
	}
	before := len(w.Hordes)
	w.Step(r.Commands)
	if len(w.Hordes) != before+2 {
		t.Fatalf("hordes %d → %d", before, len(w.Hordes))
	}
}

func TestErrorsAndHelp(t *testing.T) {
	w := world(t)
	ctl := &fakeControl{speed: 1}
	for line, want := range map[string]string{
		"nope":               "unknown command",
		"spawn ghosts":       "unknown horde",
		"spawn swordsmen 99": "count must be",
		"kill":               "nothing to target",
		"speed 100":          "speed must be",
	} {
		if r := Run(w, ctl, Context{}, line); !r.Err || !strings.Contains(r.Output, want) {
			t.Errorf("%q → %+v, want error containing %q", line, r, want)
		}
	}
	if r := Run(w, ctl, Context{}, "help"); r.Err || !strings.Contains(r.Output, "spawn <horde>") {
		t.Errorf("help → %+v", r)
	}
	Run(w, ctl, Context{}, "step 5")
	if !ctl.paused || ctl.steps != 5 {
		t.Errorf("step: paused=%v steps=%d", ctl.paused, ctl.steps)
	}
}

func TestKillBuildingUnderCursor(t *testing.T) {
	w := world(t)
	fort := w.Buildings[1]
	r := Run(w, &fakeControl{}, Context{Cursor: fort.Pos}, "kill")
	if r.Err || len(r.Commands) != 1 || len(r.Commands[0].Buildings) != 1 {
		t.Fatalf("result %+v", r)
	}
	w.Step(r.Commands)
	if w.Building(fort.ID) != nil {
		t.Error("building survived")
	}
}
