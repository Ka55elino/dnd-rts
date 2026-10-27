package fixed

// Angles are Fixed radians. Sin/Cos/Atan2 use committed lookup tables
// (trig_tables.go) with linear interpolation: no runtime float math, so
// results are identical on every platform.

const (
	// Pi is π (raw = round(π * 65536)).
	Pi Fixed = 205887
	// TwoPi is 2π.
	TwoPi Fixed = 411775
	// HalfPi is π/2.
	HalfPi Fixed = 102944
)

// NormalizeAngle wraps an angle into (-π, π].
func NormalizeAngle(a Fixed) Fixed {
	a %= TwoPi
	if a > Pi {
		a -= TwoPi
	} else if a <= -Pi {
		a += TwoPi
	}
	return a
}

// Sin returns sin(a), a in radians.
func Sin(a Fixed) Fixed {
	a %= TwoPi
	if a < 0 {
		a += TwoPi
	}
	// Position in the table as a Q.16 index: a / 2π * N.
	pos := (int64(a) * sinTableSize << FracBits) / int64(TwoPi)
	i := pos >> FracBits
	t := Fixed(pos & fracMask)
	return Lerp(sinTable[i], sinTable[i+1], t)
}

// Cos returns cos(a), a in radians.
func Cos(a Fixed) Fixed { return Sin(a + HalfPi) }

// SinCos returns sin(a) and cos(a).
func SinCos(a Fixed) (sin, cos Fixed) { return Sin(a), Cos(a) }

// Atan2 returns the angle of the vector (x, y) in (-π, π]; Atan2(0, 0) = 0.
// Argument order matches math.Atan2(y, x).
func Atan2(y, x Fixed) Fixed {
	if x == 0 && y == 0 {
		return 0
	}
	ax, ay := x.Abs(), y.Abs()

	var angle Fixed
	if ax >= ay {
		angle = atanUnit(ay.Div(ax))
	} else {
		angle = HalfPi - atanUnit(ax.Div(ay))
	}
	if x < 0 {
		angle = Pi - angle
	}
	if y < 0 {
		angle = -angle
	}
	return angle
}

// atanUnit returns atan(t) for t in [0, 1].
func atanUnit(t Fixed) Fixed {
	pos := int64(t) * atanTableSize // Q.16 index
	i := pos >> FracBits
	if i >= atanTableSize {
		return atanTable[atanTableSize]
	}
	return Lerp(atanTable[i], atanTable[i+1], Fixed(pos&fracMask))
}
