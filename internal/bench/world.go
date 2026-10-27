// Package bench is a throwaway load test for the Go → JS snapshot pipeline.
// It is NOT the game simulation: it uses float math and a global layout on purpose.
package bench

import (
	"math"
	"math/rand"
)

const (
	// MapSize is the side of the square play area in world units.
	MapSize = 512
	// MaxEntities caps what the client may request.
	MaxEntities = 100_000

	speed = 3.0 // units per second
)

// World holds N entities wandering towards random targets.
// Struct-of-arrays layout, same as the wire format.
type World struct {
	rng  *rand.Rand
	X, Z []float32
	Rot  []float32
	tx   []float32
	tz   []float32
}

func NewWorld(seed int64) *World {
	return &World{rng: rand.New(rand.NewSource(seed))}
}

func (w *World) Count() int { return len(w.X) }

// SetCount grows or shrinks the entity set.
func (w *World) SetCount(n int) {
	n = max(0, min(n, MaxEntities))
	for len(w.X) < n {
		x, z := w.randomPoint(), w.randomPoint()
		w.X = append(w.X, x)
		w.Z = append(w.Z, z)
		w.Rot = append(w.Rot, 0)
		w.tx = append(w.tx, w.randomPoint())
		w.tz = append(w.tz, w.randomPoint())
	}
	w.X, w.Z, w.Rot, w.tx, w.tz = w.X[:n], w.Z[:n], w.Rot[:n], w.tx[:n], w.tz[:n]
}

// Step advances every entity by dt seconds.
func (w *World) Step(dt float32) {
	step := speed * dt
	for i := range w.X {
		dx := w.tx[i] - w.X[i]
		dz := w.tz[i] - w.Z[i]
		dist := float32(math.Sqrt(float64(dx*dx + dz*dz)))
		if dist < step {
			w.X[i], w.Z[i] = w.tx[i], w.tz[i]
			w.tx[i], w.tz[i] = w.randomPoint(), w.randomPoint()
			continue
		}
		w.X[i] += dx / dist * step
		w.Z[i] += dz / dist * step
		w.Rot[i] = float32(math.Atan2(float64(dx), float64(dz)))
	}
}

func (w *World) randomPoint() float32 {
	return (w.rng.Float32() - 0.5) * MapSize
}
