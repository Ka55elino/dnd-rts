// Package terrain holds the map heightfield. It is shared by the simulation
// (height queries) and sent to the renderer once per match.
//
// All math is fixed-point / integer so heights are identical on every platform.
package terrain

import (
	"encoding/binary"
	"fmt"

	"dnd-rts/internal/fixed"
)

// Heightfield is a regular grid of height samples covering a map centered at (0,0).
//
// Sample (c, r) is at world position (-Width/2 + c*CellSize, -Height/2 + r*CellSize).
// Heights are stored as uint16: 0 → 0, 65535 → MaxHeight.
type Heightfield struct {
	Width, Height int // world units
	Cols, Rows    int // samples (Width/CellSize + 1, Height/CellSize + 1)
	CellSize      fixed.Fixed
	MaxHeight     fixed.Fixed
	Samples       []uint16 // row-major: index = r*Cols + c
}

// New creates a flat heightfield. width and height must be multiples of cellSize.
func New(width, height int, cellSize, maxHeight fixed.Fixed) (*Heightfield, error) {
	if cellSize <= 0 || maxHeight <= 0 || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("terrain: invalid size %dx%d cell %v max %v", width, height, cellSize, maxHeight)
	}
	w, h := fixed.FromInt(width), fixed.FromInt(height)
	if w%cellSize != 0 || h%cellSize != 0 {
		return nil, fmt.Errorf("terrain: map size %dx%d is not a multiple of cell size %v", width, height, cellSize)
	}
	cols := int(w/cellSize) + 1
	rows := int(h/cellSize) + 1
	return &Heightfield{
		Width: width, Height: height,
		Cols: cols, Rows: rows,
		CellSize: cellSize, MaxHeight: maxHeight,
		Samples: make([]uint16, cols*rows),
	}, nil
}

// LoadR16 fills the heightfield from raw little-endian uint16 samples (row-major).
func (h *Heightfield) LoadR16(raw []byte) error {
	if len(raw) != len(h.Samples)*2 {
		return fmt.Errorf("terrain: heightmap has %d bytes, want %d (%dx%d uint16)", len(raw), len(h.Samples)*2, h.Cols, h.Rows)
	}
	for i := range h.Samples {
		h.Samples[i] = binary.LittleEndian.Uint16(raw[i*2:])
	}
	return nil
}

// R16 returns the samples as raw little-endian bytes (for saving and for the renderer).
func (h *Heightfield) R16() []byte {
	out := make([]byte, len(h.Samples)*2)
	for i, s := range h.Samples {
		binary.LittleEndian.PutUint16(out[i*2:], s)
	}
	return out
}

// SampleHeight returns the height of grid sample (c, r), clamped to the grid.
func (h *Heightfield) SampleHeight(c, r int) fixed.Fixed {
	c = max(0, min(c, h.Cols-1))
	r = max(0, min(r, h.Rows-1))
	return h.toHeight(h.Samples[r*h.Cols+c])
}

// HeightAt returns the bilinearly interpolated height at a world position.
func (h *Heightfield) HeightAt(p fixed.Vec2) fixed.Fixed {
	// Grid coordinates (Q.16).
	gx := (p.X + fixed.FromInt(h.Width).DivInt(2)).Div(h.CellSize)
	gy := (p.Y + fixed.FromInt(h.Height).DivInt(2)).Div(h.CellSize)
	c, r := gx.Int(), gy.Int()
	tx, ty := gx.Frac(), gy.Frac()

	h00 := h.SampleHeight(c, r)
	h10 := h.SampleHeight(c+1, r)
	h01 := h.SampleHeight(c, r+1)
	h11 := h.SampleHeight(c+1, r+1)
	return fixed.Lerp(fixed.Lerp(h00, h10, tx), fixed.Lerp(h01, h11, tx), ty)
}

// SamplePos returns the world position of grid sample (c, r).
func (h *Heightfield) SamplePos(c, r int) fixed.Vec2 {
	return fixed.Vec2{
		X: h.CellSize.MulInt(c) - fixed.FromInt(h.Width).DivInt(2),
		Y: h.CellSize.MulInt(r) - fixed.FromInt(h.Height).DivInt(2),
	}
}

func (h *Heightfield) toHeight(s uint16) fixed.Fixed {
	return fixed.FromRaw(h.MaxHeight.Raw() * int64(s) / 65535)
}

// toSample converts a normalized value in [0, 1] to a sample.
func toSample(v fixed.Fixed) uint16 {
	v = v.Clamp(0, fixed.One)
	return uint16(v.Raw() * 65535 / fixed.One.Raw())
}
