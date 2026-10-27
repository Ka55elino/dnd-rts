package protocol

import (
	"encoding/binary"
	"math"
)

// Version of the wire format. Mirror: frontend/src/net/protocol.ts
//
// Fixed at 1 until the first release (0.0.1): Go and the frontend are always
// built together, so format changes need no bump yet. After 0.0.1 it is
// bumped on every wire format change.
const Version uint8 = 1

// Message types. Server→client < 100, client→server >= 100.
const (
	MsgSnapshot uint8 = 1

	MsgBenchSetCount uint8 = 100
)

// Snapshot layout (little endian):
//
//	header (16 bytes)
//	  u8   type        = MsgSnapshot
//	  u8   version
//	  u16  reserved
//	  u32  tick
//	  u32  count       number of entities
//	  u32  simMicros   server time spent on the tick (diagnostics)
//	entities (count * 12 bytes)
//	  f32  x
//	  f32  z
//	  f32  rotation (radians, around Y)
const (
	SnapshotHeaderSize = 16
	SnapshotEntitySize = 12
)

// EncodeSnapshot writes a snapshot into a freshly allocated buffer.
// A new buffer per frame is required: Wails streams take ownership of the slice.
func EncodeSnapshot(tick uint32, simMicros uint32, xs, zs, rots []float32) []byte {
	n := len(xs)
	buf := make([]byte, SnapshotHeaderSize+n*SnapshotEntitySize)

	buf[0] = MsgSnapshot
	buf[1] = Version
	binary.LittleEndian.PutUint32(buf[4:], tick)
	binary.LittleEndian.PutUint32(buf[8:], uint32(n))
	binary.LittleEndian.PutUint32(buf[12:], simMicros)

	off := SnapshotHeaderSize
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint32(buf[off:], math.Float32bits(xs[i]))
		binary.LittleEndian.PutUint32(buf[off+4:], math.Float32bits(zs[i]))
		binary.LittleEndian.PutUint32(buf[off+8:], math.Float32bits(rots[i]))
		off += SnapshotEntitySize
	}
	return buf
}

// DecodeBenchSetCount parses a client "set entity count" command:
//
//	u8 type = MsgBenchSetCount, u8 version, u16 reserved, u32 count
func DecodeBenchSetCount(frame []byte) (count int, ok bool) {
	if len(frame) < 8 || frame[0] != MsgBenchSetCount || frame[1] != Version {
		return 0, false
	}
	return int(binary.LittleEndian.Uint32(frame[4:])), true
}
