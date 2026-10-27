package fixed

// Vec2 is a 2D vector on the ground plane. X/Y map to world X/Z in the renderer.
type Vec2 struct {
	X Fixed `json:"x"`
	Y Fixed `json:"y"`
}

func V2(x, y Fixed) Vec2 { return Vec2{x, y} }

// V2i builds a vector from integers.
func V2i(x, y int) Vec2 { return Vec2{FromInt(x), FromInt(y)} }

func (a Vec2) Add(b Vec2) Vec2        { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2        { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Neg() Vec2              { return Vec2{-a.X, -a.Y} }
func (a Vec2) Scale(s Fixed) Vec2     { return Vec2{a.X.Mul(s), a.Y.Mul(s)} }
func (a Vec2) DivScalar(s Fixed) Vec2 { return Vec2{a.X.Div(s), a.Y.Div(s)} }
func (a Vec2) Dot(b Vec2) Fixed       { return a.X.Mul(b.X) + a.Y.Mul(b.Y) }

// Cross returns the z component of the 3D cross product (a.X*b.Y - a.Y*b.X).
func (a Vec2) Cross(b Vec2) Fixed { return a.X.Mul(b.Y) - a.Y.Mul(b.X) }

func (a Vec2) LenSq() Fixed        { return a.Dot(a) }
func (a Vec2) Len() Fixed          { return a.LenSq().Sqrt() }
func (a Vec2) DistSq(b Vec2) Fixed { return a.Sub(b).LenSq() }
func (a Vec2) Dist(b Vec2) Fixed   { return a.Sub(b).Len() }
func (a Vec2) IsZero() bool        { return a.X == 0 && a.Y == 0 }

// Normalize returns a unit vector in the same direction, or zero for a zero vector.
func (a Vec2) Normalize() Vec2 {
	l := a.Len()
	if l == 0 {
		return Vec2{}
	}
	return Vec2{a.X.Div(l), a.Y.Div(l)}
}

// ClampLen limits the length to max (keeps direction).
func (a Vec2) ClampLen(max Fixed) Vec2 {
	if a.LenSq() <= max.Mul(max) {
		return a
	}
	return a.Normalize().Scale(max)
}

// Angle returns the heading of the vector: Atan2(Y, X).
func (a Vec2) Angle() Fixed { return Atan2(a.Y, a.X) }

// FromAngle returns the unit vector (cos a, sin a).
func FromAngle(a Fixed) Vec2 { return Vec2{Cos(a), Sin(a)} }

// Rotate rotates the vector by angle a (counter-clockwise).
func (a Vec2) Rotate(angle Fixed) Vec2 {
	s, c := SinCos(angle)
	return Vec2{a.X.Mul(c) - a.Y.Mul(s), a.X.Mul(s) + a.Y.Mul(c)}
}

// MoveTowards moves a towards target by at most step; returns the new position
// and whether the target was reached.
func (a Vec2) MoveTowards(target Vec2, step Fixed) (Vec2, bool) {
	d := target.Sub(a)
	dist := d.Len()
	if dist <= step || dist == 0 {
		return target, true
	}
	// Multiply before dividing to keep precision.
	return Vec2{a.X + d.X.Mul(step).Div(dist), a.Y + d.Y.Mul(step).Div(dist)}, false
}

// LerpV2 interpolates between two vectors.
func LerpV2(a, b Vec2, t Fixed) Vec2 { return Vec2{Lerp(a.X, b.X, t), Lerp(a.Y, b.Y, t)} }
