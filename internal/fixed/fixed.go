package fixed

import (
	"encoding/json"
	"math"
	"strconv"
)

// Fixed is a signed Q47.16 fixed-point number: int64 with 16 fractional bits.
//
// Precision: 1/65536 ≈ 0.000015. Safe range for Mul: |a*b| < ~2.1e9
// (e.g. squared distances up to ~46 000 units). The game map is a few
// thousand units at most, so this is plenty.
//
// All operations are pure integer math, so results are bit-identical on
// every CPU (amd64, arm64) — required for lockstep multiplayer.
type Fixed int64

const (
	FracBits = 16
	scale    = 1 << FracBits
	fracMask = scale - 1
)

const (
	Zero Fixed = 0
	One  Fixed = scale
	Half Fixed = scale / 2
	Two  Fixed = 2 * scale

	// Max and Min are the extreme representable values.
	Max Fixed = math.MaxInt64
	Min Fixed = math.MinInt64

	// Epsilon is the smallest positive value.
	Epsilon Fixed = 1
)

// FromInt converts an integer.
func FromInt(n int) Fixed { return Fixed(int64(n) << FracBits) }

// FromFraction returns num/den, rounded toward zero.
func FromFraction(num, den int) Fixed {
	return Fixed((int64(num) << FracBits) / int64(den))
}

// FromFloat converts a float64, rounding to nearest.
//
// Deterministic for a given input (IEEE multiply by a power of two is exact),
// so it is safe for loading content. Do NOT use floats inside the simulation.
func FromFloat(f float64) Fixed {
	return Fixed(math.Round(f * scale))
}

// Raw returns the underlying integer (for serialization and hashing).
func (a Fixed) Raw() int64 { return int64(a) }

// FromRaw builds a Fixed from its underlying integer.
func FromRaw(raw int64) Fixed { return Fixed(raw) }

// Float converts to float64. For rendering, debugging and tests only.
func (a Fixed) Float() float64 { return float64(a) / scale }

// Float32 converts to float32. For rendering / snapshots only.
func (a Fixed) Float32() float32 { return float32(a.Float()) }

// Int returns the integer part, rounded toward negative infinity.
func (a Fixed) Int() int { return int(int64(a) >> FracBits) }

// Add returns a+b. (Plain + also works; methods exist for readability in chains.)
func (a Fixed) Add(b Fixed) Fixed { return a + b }

// Sub returns a-b.
func (a Fixed) Sub(b Fixed) Fixed { return a - b }

// Mul returns a*b, rounded toward negative infinity.
func (a Fixed) Mul(b Fixed) Fixed { return Fixed((int64(a) * int64(b)) >> FracBits) }

// MulInt returns a*n.
func (a Fixed) MulInt(n int) Fixed { return a * Fixed(n) }

// Div returns a/b, rounded toward zero. Panics if b == 0.
func (a Fixed) Div(b Fixed) Fixed { return Fixed((int64(a) << FracBits) / int64(b)) }

// DivInt returns a/n, rounded toward zero.
func (a Fixed) DivInt(n int) Fixed { return a / Fixed(n) }

// Neg returns -a.
func (a Fixed) Neg() Fixed { return -a }

// Abs returns |a|.
func (a Fixed) Abs() Fixed {
	if a < 0 {
		return -a
	}
	return a
}

// Sign returns -1, 0 or 1.
func (a Fixed) Sign() int {
	switch {
	case a < 0:
		return -1
	case a > 0:
		return 1
	}
	return 0
}

// Floor rounds toward negative infinity.
func (a Fixed) Floor() Fixed { return a &^ fracMask }

// Ceil rounds toward positive infinity.
func (a Fixed) Ceil() Fixed { return (a + fracMask).Floor() }

// Round rounds to the nearest integer, halves away from zero.
func (a Fixed) Round() Fixed {
	if a < 0 {
		return -((-a + Half).Floor())
	}
	return (a + Half).Floor()
}

// Frac returns the fractional part (always >= 0).
func (a Fixed) Frac() Fixed { return a & fracMask }

// Min returns the smaller of a and b.
func (a Fixed) Min(b Fixed) Fixed { return min(a, b) }

// Max returns the larger of a and b.
func (a Fixed) Max(b Fixed) Fixed { return max(a, b) }

// Clamp limits a to [lo, hi].
func (a Fixed) Clamp(lo, hi Fixed) Fixed { return max(lo, min(a, hi)) }

// Lerp returns a + (b-a)*t.
func Lerp(a, b, t Fixed) Fixed { return a + (b - a).Mul(t) }

// Sqrt returns the square root, rounded down. Negative input returns 0.
func (a Fixed) Sqrt() Fixed {
	if a <= 0 {
		return 0
	}
	// sqrt(a / 2^16) * 2^16 == sqrt(a * 2^16)
	return Fixed(isqrt(uint64(a) << FracBits))
}

// isqrt is the integer square root (floor), bit by bit — no floats.
func isqrt(n uint64) uint64 {
	var res uint64
	bit := uint64(1) << 62
	for bit > n {
		bit >>= 2
	}
	for bit != 0 {
		if n >= res+bit {
			n -= res + bit
			res = (res >> 1) + bit
		} else {
			res >>= 1
		}
		bit >>= 2
	}
	return res
}

// String formats the value as a decimal (for logs and debugging).
func (a Fixed) String() string { return strconv.FormatFloat(a.Float(), 'f', -1, 64) }

// MarshalJSON writes the value as a JSON number, e.g. 3.5.
func (a Fixed) MarshalJSON() ([]byte, error) { return []byte(a.String()), nil }

// UnmarshalJSON reads a JSON number, e.g. "speed": 3.5 in content files.
func (a *Fixed) UnmarshalJSON(data []byte) error {
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	*a = FromFloat(f)
	return nil
}
