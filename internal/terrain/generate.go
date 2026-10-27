package terrain

import "dnd-rts/internal/fixed"

// GenerateOptions controls procedural terrain.
type GenerateOptions struct {
	Seed        int64
	FeatureSize fixed.Fixed  // world size of the largest hills
	Octaves     int          // detail layers
	FlatSpots   []fixed.Vec2 // e.g. player starts: flattened plateaus
	FlatRadius  fixed.Fixed  // fully flat inside this radius
	FlatFalloff fixed.Fixed  // blend distance outside the radius
}

// DefaultGenerateOptions gives gentle rolling hills.
func DefaultGenerateOptions(seed int64) GenerateOptions {
	return GenerateOptions{
		Seed:        seed,
		FeatureSize: fixed.FromInt(160),
		Octaves:     5,
		FlatRadius:  fixed.FromInt(80), // a fortress base (radius 70) plus its wall
		FlatFalloff: fixed.FromInt(40),
	}
}

// Generate fills the heightfield with deterministic fractal value noise.
// Integer/fixed-point only: the same seed gives the same terrain everywhere.
func (h *Heightfield) Generate(o GenerateOptions) {
	octaves := max(1, o.Octaves)

	// Precompute flat-spot target heights from the raw noise.
	flatHeights := make([]fixed.Fixed, len(o.FlatSpots))
	for i, p := range o.FlatSpots {
		flatHeights[i] = shape(fbm(p, o.Seed, o.FeatureSize, octaves))
	}

	for r := 0; r < h.Rows; r++ {
		for c := 0; c < h.Cols; c++ {
			p := h.SamplePos(c, r)
			v := shape(fbm(p, o.Seed, o.FeatureSize, octaves))

			for i, spot := range o.FlatSpots {
				w := flatWeight(p.Dist(spot), o.FlatRadius, o.FlatFalloff)
				v = fixed.Lerp(v, flatHeights[i], w)
			}
			h.Samples[r*h.Cols+c] = toSample(v)
		}
	}
}

// shape remaps noise to leave lowlands flatter and hills more pronounced.
func shape(n fixed.Fixed) fixed.Fixed {
	// (n - 0.15) / 0.7, clamped, then mostly squared: flat-ish valleys
	// with a little undulation, steeper hills.
	v := (n - fixed.FromFraction(3, 20)).Div(fixed.FromFraction(7, 10)).Clamp(0, fixed.One)
	return v.Mul(v).Mul(fixed.FromFraction(17, 20)) + v.Mul(fixed.FromFraction(3, 20))
}

// flatWeight is 1 inside radius, smoothly 0 at radius+falloff.
func flatWeight(dist, radius, falloff fixed.Fixed) fixed.Fixed {
	if dist <= radius {
		return fixed.One
	}
	if falloff <= 0 || dist >= radius+falloff {
		return 0
	}
	t := fixed.One - (dist - radius).Div(falloff)
	return smoothstep(t)
}

// fbm sums octaves of value noise; result in [0, 1].
func fbm(p fixed.Vec2, seed int64, featureSize fixed.Fixed, octaves int) fixed.Fixed {
	var sum, norm fixed.Fixed
	amp := fixed.One
	x, y := p.X.Div(featureSize), p.Y.Div(featureSize)
	for o := 0; o < octaves; o++ {
		sum += valueNoise(x, y, seed+int64(o)*1013).Mul(amp)
		norm += amp
		amp = amp.DivInt(2)
		x, y = x.MulInt(2), y.MulInt(2)
	}
	return sum.Div(norm)
}

// valueNoise returns smooth noise in [0, 1) at grid coordinates (x, y).
func valueNoise(x, y fixed.Fixed, seed int64) fixed.Fixed {
	ix, iy := x.Int(), y.Int()
	tx, ty := smoothstep(x.Frac()), smoothstep(y.Frac())

	v00 := lattice(ix, iy, seed)
	v10 := lattice(ix+1, iy, seed)
	v01 := lattice(ix, iy+1, seed)
	v11 := lattice(ix+1, iy+1, seed)
	return fixed.Lerp(fixed.Lerp(v00, v10, tx), fixed.Lerp(v01, v11, tx), ty)
}

// lattice is a hashed random value in [0, 1) at an integer grid point.
func lattice(x, y int, seed int64) fixed.Fixed {
	h := uint64(int64(x))*0x9E3779B97F4A7C15 ^ uint64(int64(y))*0xC2B2AE3D27D4EB4F ^ uint64(seed)*0x165667B19E3779F9
	h ^= h >> 29
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 32
	return fixed.FromRaw(int64(h & 0xFFFF)) // 16 fractional bits → [0, 1)
}

func smoothstep(t fixed.Fixed) fixed.Fixed {
	t = t.Clamp(0, fixed.One)
	return t.Mul(t).Mul(fixed.FromInt(3) - t.MulInt(2))
}
