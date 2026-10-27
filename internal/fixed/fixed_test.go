package fixed

import (
	"encoding/json"
	"hash/fnv"
	"math"
	"testing"
)

func near(t *testing.T, name string, got Fixed, want, tol float64) {
	t.Helper()
	if d := math.Abs(got.Float() - want); d > tol {
		t.Errorf("%s = %v, want %v (±%v), diff %v", name, got.Float(), want, tol, d)
	}
}

func TestConstants(t *testing.T) {
	near(t, "Pi", Pi, math.Pi, 1e-5)
	near(t, "TwoPi", TwoPi, 2*math.Pi, 1e-5)
	near(t, "HalfPi", HalfPi, math.Pi/2, 1e-5)
}

func TestArithmetic(t *testing.T) {
	a, b := FromFloat(3.25), FromFloat(-1.5)
	near(t, "add", a+b, 1.75, 0)
	near(t, "sub", a-b, 4.75, 0)
	near(t, "mul", a.Mul(b), -4.875, 0)
	near(t, "div", a.Div(b), -2.1666667, 2e-5)
	near(t, "fraction", FromFraction(1, 3), 1.0/3, 2e-5)
	if FromInt(7).Int() != 7 || FromFloat(-2.5).Int() != -3 {
		t.Error("Int() must floor")
	}
	near(t, "floor", FromFloat(-2.5).Floor(), -3, 0)
	near(t, "ceil", FromFloat(2.1).Ceil(), 3, 0)
	near(t, "round+", FromFloat(2.5).Round(), 3, 0)
	near(t, "round-", FromFloat(-2.5).Round(), -3, 0)
	near(t, "clamp", FromInt(10).Clamp(Zero, FromInt(5)), 5, 0)
	near(t, "lerp", Lerp(FromInt(2), FromInt(4), Half), 3, 0)
}

func TestSqrt(t *testing.T) {
	for _, v := range []float64{0, 0.0001, 0.25, 1, 2, 3, 10, 100, 12345.678, 500000} {
		f := FromFloat(v)
		near(t, "sqrt", f.Sqrt(), math.Sqrt(f.Float()), 1e-4)
	}
	if FromInt(-4).Sqrt() != 0 {
		t.Error("sqrt of negative must be 0")
	}
}

func TestTrig(t *testing.T) {
	for deg := -720; deg <= 720; deg += 3 {
		r := float64(deg) * math.Pi / 180
		a := FromFloat(r)
		near(t, "sin", Sin(a), math.Sin(r), 2e-4)
		near(t, "cos", Cos(a), math.Cos(r), 2e-4)
	}
}

func TestAtan2(t *testing.T) {
	for deg := -179; deg <= 180; deg++ {
		r := float64(deg) * math.Pi / 180
		for _, radius := range []float64{0.01, 1, 300} {
			x, y := radius*math.Cos(r), radius*math.Sin(r)
			got := Atan2(FromFloat(y), FromFloat(x))
			want := math.Atan2(FromFloat(y).Float(), FromFloat(x).Float())
			// Wrap-around at ±π is equivalent.
			d := math.Abs(got.Float() - want)
			if d > math.Pi {
				d = 2*math.Pi - d
			}
			if d > 2e-3 {
				t.Errorf("atan2(%v,%v) = %v, want %v", y, x, got.Float(), want)
			}
		}
	}
	if Atan2(0, 0) != 0 {
		t.Error("atan2(0,0) must be 0")
	}
}

func TestNormalizeAngle(t *testing.T) {
	for deg := -1000; deg <= 1000; deg += 7 {
		a := NormalizeAngle(FromFloat(float64(deg) * math.Pi / 180))
		if a <= -Pi || a > Pi {
			t.Fatalf("NormalizeAngle(%d°) = %v out of range", deg, a)
		}
	}
}

func TestVec2(t *testing.T) {
	v := V2i(3, 4)
	near(t, "len", v.Len(), 5, 1e-4)
	n := v.Normalize()
	near(t, "norm.x", n.X, 0.6, 1e-4)
	near(t, "norm.y", n.Y, 0.8, 1e-4)
	if !(Vec2{}).Normalize().IsZero() {
		t.Error("normalize zero must be zero")
	}

	p, reached := V2i(0, 0).MoveTowards(V2i(10, 0), FromInt(3))
	near(t, "move.x", p.X, 3, 1e-4)
	if reached {
		t.Error("should not reach")
	}
	if _, reached = V2i(9, 0).MoveTowards(V2i(10, 0), FromInt(3)); !reached {
		t.Error("should reach")
	}

	r := V2i(1, 0).Rotate(HalfPi)
	near(t, "rot.x", r.X, 0, 2e-4)
	near(t, "rot.y", r.Y, 1, 2e-4)
	near(t, "clamp", V2i(30, 40).ClampLen(FromInt(5)).Len(), 5, 1e-3)
}

func TestJSON(t *testing.T) {
	var v struct{ Speed Fixed }
	if err := json.Unmarshal([]byte(`{"Speed": 3.5}`), &v); err != nil {
		t.Fatal(err)
	}
	if v.Speed != FromFloat(3.5) {
		t.Errorf("got %v", v.Speed)
	}
	out, _ := json.Marshal(v)
	if string(out) != `{"Speed":3.5}` {
		t.Errorf("marshal: %s", out)
	}
}

// TestDeterminismGolden runs a long chain of operations and compares a hash of
// all intermediate raw values with a golden constant. The same value must come
// out on every OS/CPU (macOS arm64, Windows/Linux amd64). If this fails after an
// intentional math change, update the constant — and know that it breaks
// replays and multiplayer compatibility with older builds.
func TestDeterminismGolden(t *testing.T) {
	const golden uint64 = 0x54a296df1b32c55c

	h := fnv.New64a()
	write := func(f Fixed) {
		var b [8]byte
		for i := range b {
			b[i] = byte(f >> (8 * i))
		}
		h.Write(b[:])
	}

	pos := V2i(0, 0)
	vel := V2(FromFloat(0.37), FromFloat(-0.21))
	angle := Zero
	for i := 0; i < 10000; i++ {
		angle = NormalizeAngle(angle + FromFraction(7, 100))
		dir := FromAngle(angle)
		vel = vel.Add(dir.Scale(FromFraction(1, 50))).ClampLen(FromInt(2))
		pos = pos.Add(vel)
		target := V2i(i%97-48, i%89-44)
		pos, _ = pos.MoveTowards(target, FromFraction(1, 10))
		write(pos.X)
		write(pos.Y)
		write(pos.Len())
		write(Atan2(vel.Y, vel.X))
		write(Sin(angle).Mul(Cos(angle)))
	}

	got := h.Sum64()
	if golden == 0 {
		t.Fatalf("golden not set; computed %#x", got)
	}
	if got != golden {
		t.Fatalf("determinism hash = %#x, want %#x", got, golden)
	}
}
