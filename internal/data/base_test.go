package data

import (
	"os"
	"testing"

	"dnd-rts/internal/fixed"
)

func TestKeepBaseLayout(t *testing.T) {
	c, err := Load(os.DirFS("../../content"))
	if err != nil {
		t.Fatal(err)
	}
	k := c.Buildings["keep"]
	if len(k.Plots) != 24 {
		t.Fatalf("keep plots = %d, want 24 (4 m + 6 s/l + 6 m + 8 tower sockets)", len(k.Plots))
	}
	count := map[string]int{}
	for _, p := range k.Plots {
		count[p.Class]++
		if p.Size != c.PlotSizes[p.Class] {
			t.Errorf("plot %v: size %v for class %q", p.Offset, p.Size, p.Class)
		}
	}
	if count["m"] != 10 || count["s"] != 4 || count["l"] != 2 || count["tower"] != 8 {
		t.Errorf("classes %v", count)
	}
	l := k.Base
	if len(l.Outline) != 32 || len(l.GatePos) != 2 || len(l.TowerSockets) != 8 {
		t.Errorf("outline %d gates %d towers %d", len(l.Outline), len(l.GatePos), len(l.TowerSockets))
	}
	if d := l.GatePos[0].Sub(fixed.Vec2{Y: l.Radius}).Len(); d > fixed.FromFraction(1, 10) {
		t.Errorf("front gate at %v, want (0, %v)", l.GatePos[0], l.Radius)
	}
	if n := len(l.Segments); n != 68 {
		t.Errorf("wall slots = %d, want 68 (2 gatehouses + 2 arcs of 33 segments)", n)
	}
	gates := 0
	for _, s := range l.Segments {
		if s.Gate {
			gates++
			if s.Span != 3 {
				t.Errorf("gatehouse spans %d segments, want 3", s.Span)
			}
		}
		if d := (s.Offset.Len() - l.Radius).Abs(); d > fixed.FromFraction(1, 10) {
			t.Errorf("segment off the circle by %v", d)
		}
	}
	if gates != 2 {
		t.Errorf("gatehouses = %d, want 2", gates)
	}
	for _, s := range l.Segments {
		if s.Gate && s.Offset.Sub(l.GatePos[0]).Len() > fixed.One && s.Offset.Sub(l.GatePos[1]).Len() > fixed.One {
			t.Errorf("gatehouse at %v is not at a gate", s.Offset)
		}
	}
	if len(l.StairList) < 4 {
		t.Errorf("stairs = %d, want at least one on each side of each gate", len(l.StairList))
	}
	for _, st := range l.StairList {
		if l.Segments[st.Node].Gate || st.Ground.Len() >= l.Radius {
			t.Errorf("stair %+v: on a gate or outside", st)
		}
	}
	if k.NoBuildRadius < l.Radius {
		t.Errorf("free buildings may reach inside the wall: noBuildRadius %v", k.NoBuildRadius)
	}
}

func TestSquareBaseWallPoints(t *testing.T) {
	l := &BaseLayout{Shape: "square", Radius: fixed.FromInt(40), Gates: []fixed.Fixed{0, fixed.FromInt(45)}}
	c := &Catalog{Buildings: map[string]*Building{"x": {Base: l}}, BuildingIDs: []string{"x"}, PlotSizes: PlotSizes{}}
	c.expandBases()
	if len(l.Outline) != 4 {
		t.Fatalf("square outline %d", len(l.Outline))
	}
	front := l.GatePos[0]
	if (front.Y-fixed.FromInt(40)).Abs() > fixed.FromFraction(1, 10) || front.X.Abs() > fixed.FromFraction(1, 10) {
		t.Errorf("front gate of a square at %v, want (0, 40)", front)
	}
	corner := l.GatePos[1]
	if (corner.X-fixed.FromInt(40)).Abs() > fixed.FromFraction(2, 10) || (corner.Y-fixed.FromInt(40)).Abs() > fixed.FromFraction(2, 10) {
		t.Errorf("45° gate of a square at %v, want the corner (40, 40)", corner)
	}
}
