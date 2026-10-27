package protocol

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestEncodeSnapshot(t *testing.T) {
	buf := EncodeSnapshot(42, 1234, []float32{1.5, -2}, []float32{3, 4.25}, []float32{0.5, -1})
	if len(buf) != SnapshotHeaderSize+2*SnapshotEntitySize {
		t.Fatalf("len = %d", len(buf))
	}
	if buf[0] != MsgSnapshot || buf[1] != Version {
		t.Fatalf("header = %v", buf[:2])
	}
	le := binary.LittleEndian
	if le.Uint32(buf[4:]) != 42 || le.Uint32(buf[8:]) != 2 || le.Uint32(buf[12:]) != 1234 {
		t.Fatal("bad header fields")
	}
	f := func(off int) float32 { return math.Float32frombits(le.Uint32(buf[off:])) }
	// second entity: x=-2, z=4.25, rot=-1
	o := SnapshotHeaderSize + SnapshotEntitySize
	if f(o) != -2 || f(o+4) != 4.25 || f(o+8) != -1 {
		t.Fatalf("entity 1 = %v %v %v", f(o), f(o+4), f(o+8))
	}
}

func TestDecodeBenchSetCount(t *testing.T) {
	frame := []byte{MsgBenchSetCount, Version, 0, 0, 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(frame[4:], 5000)
	if n, ok := DecodeBenchSetCount(frame); !ok || n != 5000 {
		t.Fatalf("got %d %v", n, ok)
	}
	if _, ok := DecodeBenchSetCount(frame[:4]); ok {
		t.Fatal("short frame must fail")
	}
}
