package sim

import (
	"os"
	"testing"

	"dnd-rts/internal/data"
	"dnd-rts/internal/terrain"
)

func TestSetupPlayground(t *testing.T) {
	cat, err := data.Load(os.DirFS("../../content"))
	if err != nil {
		t.Fatal(err)
	}
	hf, err := terrain.ForMap(os.DirFS("../../content"), cat.Maps["plains"])
	if err != nil {
		t.Fatal(err)
	}
	pairs := [][2]string{{"empire", "empire"}}
	for _, f := range cat.FactionIDs {
		pairs = append(pairs, [2]string{f, "empire"}, [2]string{"warband", f})
	}
	for _, pair := range pairs {
		w := NewWorld(cat, hf)
		if err := SetupPlayground(w, "plains", []PlayerSetup{{0, pair[0]}, {1, pair[1]}}); err != nil {
			t.Fatalf("%v: %v", pair, err)
		}
		for i, fid := range pair {
			p := PlayerID(i)
			f := cat.Factions[fid]
			hordes := 0
			for _, h := range w.Hordes {
				if h.Owner == p {
					hordes++
				}
			}
			if hordes != len(f.Hordes) {
				t.Errorf("%s: %d hordes, want one of each of %d types", fid, hordes, len(f.Hordes))
			}
			for _, bid := range f.Buildings {
				n := 0
				for _, b := range w.Buildings {
					if b.Owner == p && b.Def.ID == bid {
						n++
						if b.UnderConstruction() {
							t.Errorf("%s %s is under construction, want finished", fid, bid)
						}
					}
				}
				if want := w.Catalog.Buildings[bid].Kind; want == "wall" && n < 5 {
					t.Errorf("%s: %d wall segments, want a wall line", fid, n)
				} else if want != "wall" && n != 1 {
					t.Errorf("%s: %d × %s, want 1", fid, n, bid)
				}
			}
		}
	}
}
