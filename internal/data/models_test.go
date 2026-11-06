package data

import (
	"os"
	"path/filepath"
	"testing"
)

// Every model the content names is baked in frontend/public/models (stand-ins: tools/modelc for
// units, tools/propc for buildings and machines).
func TestContentModelsExist(t *testing.T) {
	c, err := Load(os.DirFS("../../content"))
	if err != nil {
		t.Fatal(err)
	}
	root := "../../frontend/public/models"
	check := func(kind, id, model string) {
		if model == "" {
			return
		}
		if _, err := os.Stat(filepath.Join(root, model, "model.json")); err != nil {
			t.Errorf("%s %s: model %q is not baked", kind, id, model)
		}
	}
	for _, id := range c.BuildingIDs {
		b := c.Buildings[id]
		check("building", id, b.Model)
		for _, m := range b.FactionModels {
			check("building", id, m)
		}
	}
	for _, id := range c.UnitIDs {
		u := c.Units[id]
		check("unit", id, u.Model)
		check("unit", id, u.Mesh)
	}
}
