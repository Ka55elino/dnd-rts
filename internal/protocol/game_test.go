package protocol

import (
	"encoding/binary"
	"testing"
)

func TestDecodeHello(t *testing.T) {
	frame := append([]byte{MsgHello, Version, 6, 0}, "plains"...)
	if id, err := DecodeHello(frame); err != nil || id != "plains" {
		t.Fatalf("got %q %v", id, err)
	}
	if _, err := DecodeHello(frame[:6]); err == nil {
		t.Fatal("truncated hello must fail")
	}
}

func TestDecodeMove(t *testing.T) {
	le := binary.LittleEndian
	frame := make([]byte, 12+2*4)
	frame[0], frame[1] = MsgCmdMove, Version
	le.PutUint16(frame[2:], 2)
	x := int32(-5 << 16)
	le.PutUint32(frame[4:], uint32(x))
	le.PutUint32(frame[8:], uint32(int32(7<<16)))
	le.PutUint32(frame[12:], 3)
	le.PutUint32(frame[16:], 9)
	m, err := DecodeMove(frame)
	if err != nil || m.X != -5<<16 || m.Y != 7<<16 || len(m.Hordes) != 2 || m.Hordes[1] != 9 {
		t.Fatalf("got %+v %v", m, err)
	}
	le.PutUint16(frame[2:], 5) // claims more ids than present
	if _, err := DecodeMove(frame); err == nil {
		t.Fatal("short move must fail")
	}
}

func TestEncodeGameSnapshotSize(t *testing.T) {
	buf := EncodeGameSnapshot(SnapshotHeader{Tick: 1, SimMicros: 2, HashTick: 20, Hash: 0xdeadbeef}, &GameState{
		Hordes: make([]HordeState, 3), Soldiers: make([]SoldierState, 10), Projectiles: make([]ProjectileState, 2),
		Buildings: make([]BuildingState, 4), Players: make([]PlayerState, 2), Resources: make([]ResourceState, 4),
		Queues: []QueueState{{Hordes: []uint16{1}}, {Hordes: []uint16{1, 2, 3}}},
	})
	if len(buf) != GameHeaderSize+3*GameHordeSize+10*GameSoldierSize+2*GameProjectileSize+4*GameBuildingSize+
		2*GamePlayerSize+4*GameResourceSize+(8+4)+(8+8) {
		t.Fatalf("size %d", len(buf))
	}
	buf2 := EncodeGameSnapshot(SnapshotHeader{}, &GameState{
		Events:    []EventState{{}, {}},
		Abilities: []HordeAbilities{{Slots: []AbilitySlot{{}, {}}}},
	})
	if len(buf2) != GameHeaderSize+2*GameEventSize+(8+8) {
		t.Fatalf("len = %d", len(buf2))
	}
	if binary.LittleEndian.Uint16(buf[2:]) != 3 || binary.LittleEndian.Uint32(buf[8:]) != 10 || binary.LittleEndian.Uint32(buf[16:]) != 2 {
		t.Fatal("bad counts")
	}
	if binary.LittleEndian.Uint32(buf[20:]) != 20 || binary.LittleEndian.Uint64(buf[24:]) != 0xdeadbeef {
		t.Fatal("bad hash fields")
	}
}

func TestDecodeAttack(t *testing.T) {
	le := binary.LittleEndian
	frame := make([]byte, 8+4)
	frame[0], frame[1] = MsgCmdAttack, Version
	le.PutUint16(frame[2:], 1)
	le.PutUint32(frame[4:], 77)
	le.PutUint32(frame[8:], 5)
	a, err := DecodeAttack(frame)
	if err != nil || a.Target != 77 || len(a.Hordes) != 1 || a.Hordes[0] != 5 {
		t.Fatalf("got %+v %v", a, err)
	}
}

func TestDecodeAbility(t *testing.T) {
	frame := []byte{MsgCmdAbility, Version, 2, 0, 7, 0, 0, 0, 9, 0, 0, 0}
	ids, err := DecodeAbility(frame)
	if err != nil || len(ids) != 2 || ids[0] != 7 || ids[1] != 9 {
		t.Fatalf("got %v %v", ids, err)
	}
	if _, err := DecodeAbility(frame[:6]); err == nil {
		t.Fatal("short frame must fail")
	}
}

func TestConsoleRoundTrip(t *testing.T) {
	text := "spawn swordsmen 2 enemy"
	frame := make([]byte, 16+2*4+len(text))
	frame[0], frame[1] = MsgCmdConsole, Version
	binary.LittleEndian.PutUint16(frame[2:], 2)
	x := int32(-5 * 65536)
	binary.LittleEndian.PutUint32(frame[4:], uint32(x))
	binary.LittleEndian.PutUint32(frame[8:], 7*65536)
	binary.LittleEndian.PutUint16(frame[12:], uint16(len(text)))
	binary.LittleEndian.PutUint32(frame[16:], 11)
	binary.LittleEndian.PutUint32(frame[20:], 12)
	copy(frame[24:], text)
	c, err := DecodeConsole(frame)
	if err != nil {
		t.Fatal(err)
	}
	if c.CursorX != -5*65536 || c.CursorZ != 7*65536 || len(c.Selected) != 2 || c.Selected[1] != 12 || c.Text != text {
		t.Fatalf("decoded %+v", c)
	}
	if _, err := DecodeConsole(frame[:20]); err == nil {
		t.Error("truncated frame accepted")
	}
	out := EncodeConsoleOutput(ConsoleError, "boom")
	if out[0] != MsgConsoleOutput || out[2] != ConsoleError || string(out[8:]) != "boom" {
		t.Errorf("output frame %v", out)
	}
}

func TestPlotAndFlags(t *testing.T) {
	frame := []byte{MsgCmdPlot, Version, 2, 0, 7, 0, 0, 0, 3, 0, 0, 0}
	p, err := DecodePlot(frame)
	if err != nil || p.Plot != 2 || p.Host != 7 || p.Kind != 3 {
		t.Fatalf("plot %+v %v", p, err)
	}
	f := BuildingFlags(true, false, 0.5, 0b101)
	if f&BuildingConstructing == 0 || (f>>8)&0xff != 127 || f>>16 != 0b101 {
		t.Fatalf("flags %#x", f)
	}
	if BuildingFlags(false, false, 0.5, 0)&BuildingConstructing != 0 {
		t.Error("finished building flagged as constructing")
	}
}

func TestDecodeBuild(t *testing.T) {
	frame := make([]byte, 24)
	frame[0], frame[1] = MsgCmdBuild, Version
	binary.LittleEndian.PutUint16(frame[2:], 1)
	binary.LittleEndian.PutUint16(frame[4:], 2)
	binary.LittleEndian.PutUint32(frame[8:], 10<<16)
	binary.LittleEndian.PutUint32(frame[16:], 3<<16)
	binary.LittleEndian.PutUint32(frame[20:], 9)
	b, err := DecodeBuild(frame)
	if err != nil || b.Kind != 2 || b.X != 10<<16 || b.Angle != 3<<16 || len(b.Hordes) != 1 || b.Hordes[0] != 9 {
		t.Fatalf("%+v %v", b, err)
	}
}
