package nav

import (
	"testing"

	"dnd-rts/internal/fixed"
)

// grid builds a test grid from rows of '.' (walkable) and '#' (blocked).
func grid(rows ...string) *Grid {
	g := &Grid{Cols: len(rows[0]), Rows: len(rows), CellSize: fixed.One}
	g.Width, g.Height = g.Cols, g.Rows
	for _, row := range rows {
		for _, ch := range row {
			g.Blocked = append(g.Blocked, ch == '#')
		}
	}
	return g
}

func TestFieldGoesThroughGap(t *testing.T) {
	g := grid(
		"#######",
		"#..#..#",
		"#..#..#",
		"#.....#", // gap in the wall at column 3
		"#######",
	)
	target := g.Index(5, 1)
	f := g.BuildField(target)
	start := g.Index(1, 1)
	if !f.Reachable(start) {
		t.Fatal("start should be reachable")
	}
	// Follow the field: must pass through the gap (3,3) and end at the target.
	cur, passedGap := start, false
	for steps := 0; cur != target; steps++ {
		if steps > 50 {
			t.Fatal("no path to target")
		}
		if cur == g.Index(3, 3) {
			passedGap = true
		}
		if g.Blocked[cur] {
			t.Fatalf("path enters blocked cell %d", cur)
		}
		cur = int(f.Next[cur])
	}
	if !passedGap {
		t.Fatal("path did not use the gap")
	}
}

func TestNoCornerCutting(t *testing.T) {
	g := grid(
		"####",
		"#.##",
		"##.#",
		"####",
	)
	f := g.BuildField(g.Index(2, 2))
	if f.Reachable(g.Index(1, 1)) {
		t.Fatal("diagonal squeeze between blocked corners must not be allowed")
	}
}

func TestLineWalkableAndNearest(t *testing.T) {
	g := grid(
		"#####",
		"#.#.#",
		"#...#",
		"#####",
	)
	// World coords: grid is centered, cell (c,r) center = (c-2.5+0.5, r-2+0.5).
	a := g.Center(g.Index(1, 1))
	b := g.Center(g.Index(3, 1))
	if g.LineWalkable(a, b) {
		t.Fatal("line through the wall must be blocked")
	}
	if !g.LineWalkable(g.Center(g.Index(1, 2)), g.Center(g.Index(3, 2))) {
		t.Fatal("open corridor must be walkable")
	}
	if n := g.NearestWalkable(g.Index(2, 1)); g.Blocked[n] {
		t.Fatal("nearest walkable is blocked")
	}
}

func TestCacheEvicts(t *testing.T) {
	g := grid("....", "....")
	c := NewCache(g, 2)
	f0 := c.Field(0)
	c.Field(1)
	if c.Field(0) != f0 {
		t.Fatal("cached field should be reused")
	}
	c.Field(2) // evicts 0
	if c.Field(0) == f0 {
		t.Fatal("oldest field should have been evicted")
	}
}
