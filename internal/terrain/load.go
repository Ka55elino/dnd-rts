package terrain

import (
	"fmt"
	"io/fs"
	"path"

	"dnd-rts/internal/data"
)

// ForMap builds the heightfield of a map: from its heightmap file if set,
// otherwise procedurally from its seed (flattening player start areas).
func ForMap(fsys fs.FS, m *data.Map) (*Heightfield, error) {
	t := m.Terrain
	h, err := New(m.Width, m.Height, t.CellSize, t.MaxHeight)
	if err != nil {
		return nil, fmt.Errorf("map %q: %w", m.ID, err)
	}

	if t.Heightmap != "" {
		file := path.Join("maps", m.ID, t.Heightmap)
		raw, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, fmt.Errorf("map %q: %w", m.ID, err)
		}
		if err := h.LoadR16(raw); err != nil {
			return nil, fmt.Errorf("map %q: %w", m.ID, err)
		}
		return h, nil
	}

	opts := DefaultGenerateOptions(t.Seed)
	for _, p := range m.Players {
		// The fortress first: its base (plots, wall) needs level ground; the army start next to it.
		opts.FlatSpots = append(opts.FlatSpots, p.Fortress.Pos)
		opts.FlatSpots = append(opts.FlatSpots, p.Start)
	}
	h.Generate(opts)
	return h, nil
}
