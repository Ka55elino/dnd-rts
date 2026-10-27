package terrain

import (
	"hash/fnv"
	"testing"

	"dnd-rts/internal/fixed"
)

func newTest(t *testing.T) *Heightfield {
	t.Helper()
	h, err := New(64, 32, fixed.FromInt(2), fixed.FromInt(20))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestNewSize(t *testing.T) {
	h := newTest(t)
	if h.Cols != 33 || h.Rows != 17 || len(h.Samples) != 33*17 {
		t.Fatalf("cols=%d rows=%d", h.Cols, h.Rows)
	}
	if _, err := New(65, 32, fixed.FromInt(2), fixed.One); err == nil {
		t.Fatal("size not multiple of cell must fail")
	}
}

func TestHeightAtInterpolates(t *testing.T) {
	h := newTest(t)
	// Sample (0,0) at world (-32,-16) = max height, neighbours 0.
	h.Samples[0] = 65535
	if got := h.HeightAt(fixed.V2i(-32, -16)); got != fixed.FromInt(20) {
		t.Fatalf("corner = %v", got)
	}
	// Halfway to the next sample along X → half height.
	got := h.HeightAt(fixed.V2i(-31, -16))
	if d := (got - fixed.FromInt(10)).Abs(); d > fixed.FromFraction(1, 1000) {
		t.Fatalf("mid = %v", got)
	}
	// Outside the map clamps to the edge.
	if got := h.HeightAt(fixed.V2i(-500, -500)); got != fixed.FromInt(20) {
		t.Fatalf("clamped = %v", got)
	}
}

func TestR16RoundTrip(t *testing.T) {
	a := newTest(t)
	a.Generate(DefaultGenerateOptions(7))
	b := newTest(t)
	if err := b.LoadR16(a.R16()); err != nil {
		t.Fatal(err)
	}
	for i := range a.Samples {
		if a.Samples[i] != b.Samples[i] {
			t.Fatalf("sample %d differs", i)
		}
	}
	if err := b.LoadR16([]byte{1, 2, 3}); err == nil {
		t.Fatal("wrong size must fail")
	}
}

func TestFlatSpots(t *testing.T) {
	h, _ := New(256, 256, fixed.FromInt(2), fixed.FromInt(30))
	o := DefaultGenerateOptions(42)
	spot := fixed.V2i(-60, 20)
	o.FlatSpots = []fixed.Vec2{spot}
	h.Generate(o)
	center := h.HeightAt(spot)
	for _, d := range []fixed.Vec2{fixed.V2i(30, 0), fixed.V2i(0, -30), fixed.V2i(-20, 20)} {
		if got := h.HeightAt(spot.Add(d)); (got - center).Abs() > fixed.FromFraction(1, 100) {
			t.Errorf("not flat at %v: %v vs %v", d, got, center)
		}
	}
}

// Same seed must give bit-identical terrain on every platform.
func TestGenerateDeterminismGolden(t *testing.T) {
	const golden uint64 = 0x5be6f2a00f640f6a

	h, _ := New(512, 512, fixed.FromInt(2), fixed.FromInt(28))
	o := DefaultGenerateOptions(1337)
	o.FlatSpots = []fixed.Vec2{fixed.V2i(-180, 0), fixed.V2i(180, 0)}
	h.Generate(o)

	f := fnv.New64a()
	f.Write(h.R16())
	if got := f.Sum64(); got != golden {
		t.Fatalf("terrain hash = %#x, want %#x", got, golden)
	}
}
