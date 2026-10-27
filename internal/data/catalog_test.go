package data

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"dnd-rts/internal/fixed"
)

// The real content/ folder must always load cleanly.
func TestLoadRepoContent(t *testing.T) {
	c, err := Load(os.DirFS("../../content"))
	if err != nil {
		t.Fatalf("content/ has errors:\n%v", err)
	}
	if len(c.UnitIDs) == 0 || len(c.HordeIDs) == 0 || len(c.FactionIDs) == 0 || len(c.MapIDs) == 0 {
		t.Fatalf("expected sample content, got %d units, %d hordes, %d factions, %d maps",
			len(c.UnitIDs), len(c.HordeIDs), len(c.FactionIDs), len(c.MapIDs))
	}
}

func TestLoadParsesFixed(t *testing.T) {
	fsys := fstest.MapFS{
		"units/spear.json": {Data: []byte(`{
			"id": "spear", "name": "Spear", "model": "",
			"health": 100, "armor": 1.5, "speed": 3.25, "radius": 0.4, "mass": 1,
			"melee": {"damage": 10, "range": 1.5, "cooldown": 1.2}
		}`)},
	}
	c, _ := Load(fsys) // no hordes/factions/maps → validation errors are fine here
	u := c.Units["spear"]
	if u == nil {
		t.Fatal("unit not loaded")
	}
	if u.Speed != fixed.FromFloat(3.25) || u.Armor != fixed.FromFloat(1.5) {
		t.Errorf("speed=%v armor=%v", u.Speed, u.Armor)
	}
}

func TestLoadReportsErrors(t *testing.T) {
	fsys := fstest.MapFS{
		"units/a.json":    {Data: []byte(`{"id": "a", "heatlh": 10}`)},               // typo → unknown field
		"units/b.json":    {Data: []byte(`{"id": "not_b"}`)},                         // id/file mismatch
		"hordes/h.json":   {Data: []byte(`{"id": "h", "unit": "ghost", "size": 0}`)}, // bad refs/values
		"factions/f.json": {Data: []byte(`{"id": "f", "color": "red", "hordes": ["nope"]}`)},
		"maps/m/map.json": {Data: []byte(`{"id": "m", "width": 100, "height": 100, "players": []}`)},
	}
	_, err := Load(fsys)
	if err == nil {
		t.Fatal("expected errors")
	}
	msg := err.Error()
	for _, want := range []string{
		`unknown field "heatlh"`,
		`must match file name "b"`,
		`unknown unit "ghost"`,
		`size must be >= 1`,
		`color must be #rrggbb`,
		`unknown horde "nope"`,
		`at least 2 player start`,
		`terrain.cellSize and terrain.maxHeight must be > 0`,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing error %q in:\n%s", want, msg)
		}
	}
}
