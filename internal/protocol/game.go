package protocol

import (
	"encoding/binary"
	"errors"
	"math"
)

// Game messages. Mirror: frontend/src/net/protocol.ts
const (
	MsgGameSnapshot  uint8 = 2
	MsgConsoleOutput uint8 = 3 // server → client: dev console text

	MsgHello       uint8 = 110 // client → server: start a match on a map
	MsgCmdMove     uint8 = 111 // client → server: move hordes
	MsgCmdAttack   uint8 = 112 // client → server: attack an enemy horde
	MsgCmdAbility  uint8 = 113 // client → server: cast the hordes' ability
	MsgCmdConsole  uint8 = 114 // client → server: dev console line
	MsgCmdSiege    uint8 = 115 // client → server: attack an enemy building
	MsgCmdPlot     uint8 = 116 // client → server: build on a plot
	MsgCmdCancel   uint8 = 117 // client → server: cancel a construction
	MsgCmdBuild    uint8 = 118 // client → server: workers put up a free building
	MsgCmdHelp     uint8 = 119 // client → server: workers help build a foundation
	MsgCmdTrain    uint8 = 120 // client → server: queue a horde in a building
	MsgCmdUntrain  uint8 = 121 // client → server: cancel a queue entry
	MsgCmdRally    uint8 = 122 // client → server: set a building's rally point
	MsgCmdStop     uint8 = 123 // client → server: hordes stop (ability layout)
	MsgCmdHold     uint8 = 124 // client → server: hordes hold position (ability layout)
	MsgCmdAMove    uint8 = 125 // client → server: attack-move (move layout)
	MsgCmdPatrol   uint8 = 126 // client → server: patrol (move layout)
	MsgCmdCast     uint8 = 127 // client → server: cast an ability
	MsgCmdToggle   uint8 = 128 // client → server: switch a toggle ability
	MsgCmdAuto     uint8 = 129 // client → server: set autocast on / off
	MsgCmdResearch uint8 = 130 // client → server: research an upgrade (train layout)
	MsgCmdGate     uint8 = 131 // client → server: open / close a gate
	MsgCmdBToggle  uint8 = 132 // client → server: switch a tower's toggle (fire arrows)
	MsgCmdFireAt   uint8 = 133 // client → server: ranged hordes shoot at a ground point (move layout)
)

// Game snapshot layout (little endian):
//
//	header (64 bytes)
//	  u8   type = MsgGameSnapshot
//	  u8   version
//	  u16  hordeCount
//	  u32  tick
//	  u32  soldierCount
//	  u32  simMicros
//	  u32  projectileCount
//	  u32  hashTick  tick of the latest state hash (desync detection)
//	  u64  hash
//	  u32  buildingCount
//	  u8   over      1 = the match has ended
//	  u8   winner    player id, 255 = none (draw / not decided)
//	  u8   localPlayer  the player this client controls
//	  u8   playerCount
//	  u8   resourceCount
//	  u8   upgradeCount   researched upgrades of localPlayer
//	  u16  queueCount
//	  u32  queueBytes
//	  u16  eventCount
//	  u16  abilityCount   hordes with ability records
//	  u32  abilityBytes
//	  u16  heroCount      hero records of localPlayer
//	  u8   reserved * 6
//	hordes (hordeCount * 32 bytes)
//	  u32  id
//	  u8   owner
//	  u8   kind      index into the sorted horde type list
//	  u16  alive     soldiers alive
//	  f32  x
//	  f32  z
//	  u32  hp        sum of soldier health (rounded down)
//	  u32  maxHp     full-strength health (size * unit health)
//	  u8   level     hero level 1..5, 0 = not a hero
//	  u8   flags     bit 0: transformed; bit 1: summoned
//	  u16  scale     form size relative to the base unit, ×100 (100 = normal)
//	  u32  xp        hero experience
//	soldiers (soldierCount * 24 bytes)
//	  u32  id
//	  u32  hordeId
//	  f32  x
//	  f32  z
//	  f32  rotation  three.js Y rotation (model faces +Z)
//	  u8   hp        0..255 fraction of max health
//	  u8   state     SoldierIdle / SoldierMoving / SoldierAttacking / SoldierKnocked
//	  u16  status    effect bits: 1 burning, 2 poisoned, 4 slowed, 8 stunned, 16 acid, 32 grounded (flyer pulled down), 64 on a wall walkway
//	projectiles (projectileCount * 32 bytes)
//	  u32  id
//	  u8   owner
//	  u8   flags     bit 0: shot from the air; bit 1: aimed at a flying target; bit 2: shot from a wall / tower; bit 3: area shot
//	  u16  arrows
//	  f32  fromX, fromZ, toX, toZ
//	  u32  launchTick
//	  u32  impactTick
//	buildings (buildingCount * 36 bytes)
//	  u32  id
//	  u8   owner     255 = neutral
//	  u8   capturer  player capturing a neutral outpost, 255 = nobody
//	  u16  kind      index into the sorted building type list
//	  f32  x
//	  f32  z
//	  f32  rotation  three.js Y rotation (model faces +Z)
//	  u32  hp
//	  u32  maxHp
//	  u32  flags     bit 0: under construction; bit 1: being captured; bit 2: burning; bit 3: gate open; bit 4: tower toggle on;
//	                 bits 8–15: construction or capture progress 0..255;
//	                 bits 16–31: occupied plots 0–15 of a fortress / outpost (bit i = plot i)
//	  u32  plotsHi   occupied plots 16–47 (bit i = plot 16 + i; tower sockets of a base wall come last)
//	players (playerCount * 4 bytes)
//	  u8   id
//	  u8   faction   index into the sorted faction list
//	  u8   flags     bit 0: AI controlled; bit 1: defeated; bit 2: this client
//	  u8   reserved
//	resources of localPlayer (resourceCount * 8 bytes, sorted resource id order)
//	  i32  amount    stock: amount; cap: used
//	  i32  limit     cap: provided; stock: 0
//	queues of localPlayer's buildings (queueBytes in total), each:
//	  u32  building id
//	  u8   count
//	  u8   reserved
//	  u16  progress  of the front entry, 0..65535
//	  u16  kind * count, padded to 4 bytes: horde kind, or 0x8000 | upgrade kind for research
//	events (eventCount * 24 bytes) — this tick's ability casts, for visuals
//	  u16  ability kind  index into the sorted ability list
//	  u8   owner
//	  u8   event kind    low 4 bits: 0 circle, 1 chain jump, 2 cone;
//	                     high 4 bits: damage type + 1 (index into the data.DamageTypes order), 0 = none
//	  f32  x, z          circle: centre; chain: jump start; cone: apex (the caster)
//	  f32  a, b          circle: radius, duration (zones, 0 = instant); chain: jump end x, z;
//	                     cone: length, width in degrees
//	  f32  facing        cone: direction (three.js Y rotation, like soldiers); else 0
//	abilities of localPlayer's hordes (abilityBytes in total), each:
//	  u32  horde id
//	  u8   count
//	  u8   reserved * 3
//	  per ability: u16 cooldown ticks, u8 flags (1 toggled, 2 autocast, 4 pending, 8 locked by hero level), u8 mode index (mode abilities)
//	upgrades of localPlayer (upgradeCount * u16 upgrade kind, padded to 4 bytes)
//	heroes of localPlayer (heroCount * 8 bytes) — unique hordes, alive or waiting for revival
//	  u16  horde kind
//	  u8   level
//	  u8   flags     bit 0: alive (on the field)
//	  u32  xp
const (
	GameHeaderSize     = 64
	GameHeroSize       = 8
	GameEventSize      = 24
	GamePlayerSize     = 4
	GameResourceSize   = 8
	GameBuildingSize   = 36
	GameHordeSize      = 32
	GameSoldierSize    = 24
	GameProjectileSize = 32
)

// Soldier animation states.
const (
	SoldierIdle      uint8 = 0
	SoldierMoving    uint8 = 1
	SoldierAttacking uint8 = 2
	SoldierKnocked   uint8 = 3
)

// HordeState / SoldierState are the render-facing view of the simulation.
type HordeState struct {
	ID        uint32
	Owner     uint8
	Kind      uint8
	Alive     uint16
	X, Z      float32
	HP, MaxHP uint32
	Level     uint8
	Flags     uint8  // HordeTransformed | HordeSummoned
	Scale     uint16 // ×100
	XP        uint32
}

// Horde flags.
const (
	HordeTransformed = 1 << 0
	HordeSummoned    = 1 << 1
)

// Event kinds.
const (
	EventCircle = 0
	EventChain  = 1
	EventCone   = 2
)

// HeroState is a player's unique horde (alive or fallen).
type HeroState struct {
	Kind  uint16
	Level uint8
	Alive bool
	XP    uint32
}

// EventState is an ability cast for client visuals.
type EventState struct {
	Ability           uint16
	Owner             uint8
	Kind              uint8   // EventCircle / EventChain / EventCone
	X, Z, Radius, Dur float32 // see the layout: Radius / Dur carry the chain end or the cone size
	Facing            float32
}

// Ability flags.
const (
	AbilityToggled  = 1 << 0
	AbilityAutocast = 1 << 1
	AbilityPending  = 1 << 2
	AbilityLocked   = 1 << 3
)

type AbilitySlot struct {
	Cooldown uint16
	Flags    uint8
	Mode     uint8
}

type HordeAbilities struct {
	Horde uint32
	Slots []AbilitySlot
}

func (a HordeAbilities) size() int { return 8 + 4*len(a.Slots) }

type ProjectileState struct {
	ID                     uint32
	Owner                  uint8
	Flags                  uint8 // ProjectileFromAir | ProjectileToAir
	Arrows                 uint16
	FromX, FromZ, ToX, ToZ float32
	Launch, Impact         uint32
}

type SoldierState struct {
	ID, Horde uint32
	X, Z, Rot float32
	HP        uint8
	State     uint8
	Status    uint16
}

type BuildingState struct {
	ID        uint32
	Owner     uint8
	Capturer  uint8
	Kind      uint16
	X, Z, Rot float32
	HP, MaxHP uint32
	Flags     uint32
	PlotsHi   uint32 // occupied plots 16–47
}

// Projectile flags.
const (
	ProjectileFromAir  = 1 << 0
	ProjectileToAir    = 1 << 1
	ProjectileFromWall = 1 << 2
	ProjectileSplash   = 1 << 3 // an area shot (a trebuchet stone)
)

// Player flags.
const (
	PlayerAI       = 1 << 0
	PlayerDefeated = 1 << 1
	PlayerLocal    = 1 << 2
)

type PlayerState struct {
	ID, Faction, Flags uint8
}

type ResourceState struct {
	Amount, Limit int32
}

type QueueState struct {
	Building uint32
	Progress uint16
	Hordes   []uint16 // horde kinds, front first
}

func (q QueueState) size() int { return 8 + (len(q.Hordes)*2+3)/4*4 }

// GameState is everything in a snapshot after the header.
type GameState struct {
	Hordes      []HordeState
	Soldiers    []SoldierState
	Projectiles []ProjectileState
	Buildings   []BuildingState
	Players     []PlayerState
	Resources   []ResourceState
	Queues      []QueueState
	Events      []EventState
	Abilities   []HordeAbilities
	Upgrades    []uint16 // researched upgrade kinds
	Heroes      []HeroState
}

// QueueResearch marks a queue entry as research: kind = QueueResearch | upgrade kind.
const QueueResearch = 0x8000

// SnapshotHeader carries per-snapshot metadata.
type SnapshotHeader struct {
	Tick, SimMicros uint32
	HashTick        uint32
	Hash            uint64
	Over            bool
	Winner          uint8
	LocalPlayer     uint8
}

// EncodeGameSnapshot writes a snapshot into a new buffer (streams take ownership).
func EncodeGameSnapshot(hdr SnapshotHeader, st *GameState) []byte {
	hordes, soldiers, projectiles, buildings := st.Hordes, st.Soldiers, st.Projectiles, st.Buildings
	queueBytes := 0
	for _, q := range st.Queues {
		queueBytes += q.size()
	}
	abilityBytes := 0
	for _, a := range st.Abilities {
		abilityBytes += a.size()
	}
	buf := make([]byte, GameHeaderSize+len(hordes)*GameHordeSize+len(soldiers)*GameSoldierSize+len(projectiles)*GameProjectileSize+
		len(buildings)*GameBuildingSize+len(st.Players)*GamePlayerSize+len(st.Resources)*GameResourceSize+queueBytes+
		len(st.Events)*GameEventSize+abilityBytes+(len(st.Upgrades)*2+3)/4*4+len(st.Heroes)*GameHeroSize)
	le := binary.LittleEndian

	buf[0] = MsgGameSnapshot
	buf[1] = Version
	le.PutUint16(buf[2:], uint16(len(hordes)))
	le.PutUint32(buf[4:], hdr.Tick)
	le.PutUint32(buf[8:], uint32(len(soldiers)))
	le.PutUint32(buf[12:], hdr.SimMicros)
	le.PutUint32(buf[16:], uint32(len(projectiles)))
	le.PutUint32(buf[20:], hdr.HashTick)
	le.PutUint64(buf[24:], hdr.Hash)
	le.PutUint32(buf[32:], uint32(len(buildings)))
	if hdr.Over {
		buf[36] = 1
	}
	buf[37] = hdr.Winner
	buf[38] = hdr.LocalPlayer
	buf[39] = uint8(len(st.Players))
	buf[40] = uint8(len(st.Resources))
	buf[41] = uint8(len(st.Upgrades))
	le.PutUint16(buf[42:], uint16(len(st.Queues)))
	le.PutUint32(buf[44:], uint32(queueBytes))
	le.PutUint16(buf[48:], uint16(len(st.Events)))
	le.PutUint16(buf[50:], uint16(len(st.Abilities)))
	le.PutUint32(buf[52:], uint32(abilityBytes))
	le.PutUint16(buf[56:], uint16(len(st.Heroes)))

	off := GameHeaderSize
	for _, h := range hordes {
		le.PutUint32(buf[off:], h.ID)
		buf[off+4] = h.Owner
		buf[off+5] = h.Kind
		le.PutUint16(buf[off+6:], h.Alive)
		le.PutUint32(buf[off+8:], math.Float32bits(h.X))
		le.PutUint32(buf[off+12:], math.Float32bits(h.Z))
		le.PutUint32(buf[off+16:], h.HP)
		le.PutUint32(buf[off+20:], h.MaxHP)
		buf[off+24] = h.Level
		buf[off+25] = h.Flags
		le.PutUint16(buf[off+26:], h.Scale)
		le.PutUint32(buf[off+28:], h.XP)
		off += GameHordeSize
	}
	for _, s := range soldiers {
		le.PutUint32(buf[off:], s.ID)
		le.PutUint32(buf[off+4:], s.Horde)
		le.PutUint32(buf[off+8:], math.Float32bits(s.X))
		le.PutUint32(buf[off+12:], math.Float32bits(s.Z))
		le.PutUint32(buf[off+16:], math.Float32bits(s.Rot))
		buf[off+20] = s.HP
		buf[off+21] = s.State
		le.PutUint16(buf[off+22:], s.Status)
		off += GameSoldierSize
	}
	for _, p := range projectiles {
		le.PutUint32(buf[off:], p.ID)
		buf[off+4] = p.Owner
		buf[off+5] = p.Flags
		le.PutUint16(buf[off+6:], p.Arrows)
		le.PutUint32(buf[off+8:], math.Float32bits(p.FromX))
		le.PutUint32(buf[off+12:], math.Float32bits(p.FromZ))
		le.PutUint32(buf[off+16:], math.Float32bits(p.ToX))
		le.PutUint32(buf[off+20:], math.Float32bits(p.ToZ))
		le.PutUint32(buf[off+24:], p.Launch)
		le.PutUint32(buf[off+28:], p.Impact)
		off += GameProjectileSize
	}
	for _, b := range buildings {
		le.PutUint32(buf[off:], b.ID)
		buf[off+4] = b.Owner
		buf[off+5] = b.Capturer
		le.PutUint16(buf[off+6:], b.Kind)
		le.PutUint32(buf[off+8:], math.Float32bits(b.X))
		le.PutUint32(buf[off+12:], math.Float32bits(b.Z))
		le.PutUint32(buf[off+16:], math.Float32bits(b.Rot))
		le.PutUint32(buf[off+20:], b.HP)
		le.PutUint32(buf[off+24:], b.MaxHP)
		le.PutUint32(buf[off+28:], b.Flags)
		le.PutUint32(buf[off+32:], b.PlotsHi)
		off += GameBuildingSize
	}
	for _, p := range st.Players {
		buf[off], buf[off+1], buf[off+2] = p.ID, p.Faction, p.Flags
		off += GamePlayerSize
	}
	for _, r := range st.Resources {
		le.PutUint32(buf[off:], uint32(r.Amount))
		le.PutUint32(buf[off+4:], uint32(r.Limit))
		off += GameResourceSize
	}
	for _, q := range st.Queues {
		le.PutUint32(buf[off:], q.Building)
		buf[off+4] = uint8(len(q.Hordes))
		le.PutUint16(buf[off+6:], q.Progress)
		for i, k := range q.Hordes {
			le.PutUint16(buf[off+8+i*2:], k)
		}
		off += q.size()
	}
	for _, e := range st.Events {
		le.PutUint16(buf[off:], e.Ability)
		buf[off+2] = e.Owner
		buf[off+3] = e.Kind
		le.PutUint32(buf[off+4:], math.Float32bits(e.X))
		le.PutUint32(buf[off+8:], math.Float32bits(e.Z))
		le.PutUint32(buf[off+12:], math.Float32bits(e.Radius))
		le.PutUint32(buf[off+16:], math.Float32bits(e.Dur))
		le.PutUint32(buf[off+20:], math.Float32bits(e.Facing))
		off += GameEventSize
	}
	for _, a := range st.Abilities {
		le.PutUint32(buf[off:], a.Horde)
		buf[off+4] = uint8(len(a.Slots))
		for i, sl := range a.Slots {
			le.PutUint16(buf[off+8+i*4:], sl.Cooldown)
			buf[off+8+i*4+2] = sl.Flags
			buf[off+8+i*4+3] = sl.Mode
		}
		off += a.size()
	}
	for i, k := range st.Upgrades {
		le.PutUint16(buf[off+i*2:], k)
	}
	off += (len(st.Upgrades)*2 + 3) / 4 * 4
	for _, h := range st.Heroes {
		le.PutUint16(buf[off:], h.Kind)
		buf[off+2] = h.Level
		if h.Alive {
			buf[off+3] = 1
		}
		le.PutUint32(buf[off+4:], h.XP)
		off += GameHeroSize
	}
	return buf
}

var ErrBadFrame = errors.New("protocol: malformed frame")

// Hello: u8 type, u8 version, u16 len, then the map id (UTF-8).
func DecodeHello(frame []byte) (mapID string, err error) {
	if len(frame) < 4 || frame[0] != MsgHello || frame[1] != Version {
		return "", ErrBadFrame
	}
	n := int(binary.LittleEndian.Uint16(frame[2:]))
	if len(frame) < 4+n {
		return "", ErrBadFrame
	}
	return string(frame[4 : 4+n]), nil
}

// Move command: u8 type, u8 version, u16 count, i32 x, i32 y (Q16.16 fixed raw),
// then count * u32 horde ids. Coordinates are fixed-point so every machine
// in a lockstep game receives exactly the same target.
type MoveCommand struct {
	X, Y   int32
	Hordes []uint32
}

const maxHordesPerCommand = 256

func DecodeMove(frame []byte) (MoveCommand, error) {
	le := binary.LittleEndian
	if len(frame) < 12 || frame[0] != MsgCmdMove || frame[1] != Version {
		return MoveCommand{}, ErrBadFrame
	}
	n := int(le.Uint16(frame[2:]))
	if n > maxHordesPerCommand || len(frame) < 12+n*4 {
		return MoveCommand{}, ErrBadFrame
	}
	cmd := MoveCommand{
		X:      int32(le.Uint32(frame[4:])),
		Y:      int32(le.Uint32(frame[8:])),
		Hordes: make([]uint32, n),
	}
	for i := range cmd.Hordes {
		cmd.Hordes[i] = le.Uint32(frame[12+i*4:])
	}
	return cmd, nil
}

// Attack command: u8 type, u8 version, u16 count, u32 target horde,
// then count * u32 attacking horde ids.
type AttackCommand struct {
	Target uint32
	Hordes []uint32
}

func DecodeAttack(frame []byte) (AttackCommand, error) {
	le := binary.LittleEndian
	if len(frame) < 8 || frame[0] != MsgCmdAttack || frame[1] != Version {
		return AttackCommand{}, ErrBadFrame
	}
	n := int(le.Uint16(frame[2:]))
	if n > maxHordesPerCommand || len(frame) < 8+n*4 {
		return AttackCommand{}, ErrBadFrame
	}
	cmd := AttackCommand{Target: le.Uint32(frame[4:]), Hordes: make([]uint32, n)}
	for i := range cmd.Hordes {
		cmd.Hordes[i] = le.Uint32(frame[8+i*4:])
	}
	return cmd, nil
}

// DecodeSiege: same layout as an attack command, target = building id.
func DecodeSiege(frame []byte) (AttackCommand, error) {
	if len(frame) < 1 || frame[0] != MsgCmdSiege {
		return AttackCommand{}, ErrBadFrame
	}
	f := append([]byte{MsgCmdAttack}, frame[1:]...)
	return DecodeAttack(f)
}

// Building flags.
const (
	BuildingConstructing = 1 << 0
	BuildingCapturing    = 1 << 1
	BuildingBurning      = 1 << 2
	BuildingOpen         = 1 << 3 // a gate is open
	BuildingToggled      = 1 << 4 // a tower's toggle (fire arrows) is on
)

// BuildingFlags packs construction / capture state and plot occupancy.
// progress is the construction progress, or the capture progress while capturing.
func BuildingFlags(constructing, capturing bool, progress float64, occupied uint16) uint32 {
	f := uint32(occupied) << 16
	if constructing {
		f |= BuildingConstructing
	}
	if capturing {
		f |= BuildingCapturing
	}
	if constructing || capturing {
		f |= uint32(min(max(progress, 0), 1)*255) << 8
	}
	return f
}

// PlotCommand: u8 type, u8 version, u16 plot index, u32 host building id,
// u16 building kind (index into the sorted building type list), u16 reserved.
type PlotCommand struct {
	Plot uint16
	Host uint32
	Kind uint16
}

func DecodePlot(frame []byte) (PlotCommand, error) {
	le := binary.LittleEndian
	if len(frame) < 12 || frame[0] != MsgCmdPlot || frame[1] != Version {
		return PlotCommand{}, ErrBadFrame
	}
	return PlotCommand{Plot: le.Uint16(frame[2:]), Host: le.Uint32(frame[4:]), Kind: le.Uint16(frame[8:])}, nil
}

// DecodeCancel: u8 type, u8 version, u16 reserved, u32 building id.
func DecodeCancel(frame []byte) (uint32, error) {
	if len(frame) < 8 || frame[0] != MsgCmdCancel || frame[1] != Version {
		return 0, ErrBadFrame
	}
	return binary.LittleEndian.Uint32(frame[4:]), nil
}

// BuildCommand: u8 type, u8 version, u16 count, u16 building kind, u16 reserved,
// i32 x, i32 z, i32 angle (all Q16.16; angle in radians, sim convention),
// then count * u32 worker horde ids.
type BuildCommand struct {
	Kind        uint16
	X, Z, Angle int32
	Hordes      []uint32
}

func DecodeBuild(frame []byte) (BuildCommand, error) {
	le := binary.LittleEndian
	if len(frame) < 20 || frame[0] != MsgCmdBuild || frame[1] != Version {
		return BuildCommand{}, ErrBadFrame
	}
	n := int(le.Uint16(frame[2:]))
	if n > maxHordesPerCommand || len(frame) < 20+n*4 {
		return BuildCommand{}, ErrBadFrame
	}
	c := BuildCommand{
		Kind:   le.Uint16(frame[4:]),
		X:      int32(le.Uint32(frame[8:])),
		Z:      int32(le.Uint32(frame[12:])),
		Angle:  int32(le.Uint32(frame[16:])),
		Hordes: make([]uint32, n),
	}
	for i := range c.Hordes {
		c.Hordes[i] = le.Uint32(frame[20+i*4:])
	}
	return c, nil
}

// DecodeHelp: same layout as an attack command, target = foundation id.
func DecodeHelp(frame []byte) (AttackCommand, error) {
	if len(frame) < 1 || frame[0] != MsgCmdHelp {
		return AttackCommand{}, ErrBadFrame
	}
	return DecodeAttack(append([]byte{MsgCmdAttack}, frame[1:]...))
}

// DecodeTrain: u8 type, u8 version, u16 horde kind, u32 building id.
func DecodeTrain(frame []byte) (building uint32, kind uint16, err error) {
	if len(frame) < 8 || frame[0] != MsgCmdTrain || frame[1] != Version {
		return 0, 0, ErrBadFrame
	}
	le := binary.LittleEndian
	return le.Uint32(frame[4:]), le.Uint16(frame[2:]), nil
}

// DecodeResearch: u8 type, u8 version, u16 upgrade kind, u32 building id.
func DecodeResearch(frame []byte) (building uint32, kind uint16, err error) {
	if len(frame) < 8 || frame[0] != MsgCmdResearch || frame[1] != Version {
		return 0, 0, ErrBadFrame
	}
	le := binary.LittleEndian
	return le.Uint32(frame[4:]), le.Uint16(frame[2:]), nil
}

// DecodeUntrain: u8 type, u8 version, u16 queue index, u32 building id.
func DecodeUntrain(frame []byte) (building uint32, index uint16, err error) {
	if len(frame) < 8 || frame[0] != MsgCmdUntrain || frame[1] != Version {
		return 0, 0, ErrBadFrame
	}
	le := binary.LittleEndian
	return le.Uint32(frame[4:]), le.Uint16(frame[2:]), nil
}

// DecodeRally: u8 type, u8 version, u16 reserved, u32 building id, i32 x, i32 z (Q16.16).
func DecodeRally(frame []byte) (building uint32, x, z int32, err error) {
	if len(frame) < 16 || frame[0] != MsgCmdRally || frame[1] != Version {
		return 0, 0, 0, ErrBadFrame
	}
	le := binary.LittleEndian
	return le.Uint32(frame[4:]), int32(le.Uint32(frame[8:])), int32(le.Uint32(frame[12:])), nil
}

// DecodeGate: u8 type, u8 version, u16 reserved, u32 building, u8 open (1) / close (0), u8 reserved * 3.
func DecodeGate(frame []byte) (building uint32, open bool, err error) {
	if len(frame) < 12 || frame[0] != MsgCmdGate || frame[1] != Version {
		return 0, false, ErrBadFrame
	}
	return binary.LittleEndian.Uint32(frame[4:]), frame[8] != 0, nil
}

// DecodeBuildingToggle: u8 type, u8 version, u16 ability kind, u32 building.
func DecodeBuildingToggle(frame []byte) (building uint32, ability uint16, err error) {
	if len(frame) < 8 || frame[0] != MsgCmdBToggle || frame[1] != Version {
		return 0, 0, ErrBadFrame
	}
	le := binary.LittleEndian
	return le.Uint32(frame[4:]), le.Uint16(frame[2:]), nil
}

// DecodeMoveLike decodes attack-move / patrol / attack-ground frames (same layout as a move).
func DecodeMoveLike(frame []byte) (MoveCommand, error) {
	if len(frame) < 1 || (frame[0] != MsgCmdAMove && frame[0] != MsgCmdPatrol && frame[0] != MsgCmdFireAt) {
		return MoveCommand{}, ErrBadFrame
	}
	return DecodeMove(append([]byte{MsgCmdMove}, frame[1:]...))
}

// DecodeHordeList decodes stop / hold frames (same layout as an ability cast).
func DecodeHordeList(frame []byte) ([]uint32, error) {
	if len(frame) < 1 || (frame[0] != MsgCmdStop && frame[0] != MsgCmdHold) {
		return nil, ErrBadFrame
	}
	return DecodeAbility(append([]byte{MsgCmdAbility}, frame[1:]...))
}

// CastCommand: u8 type, u8 version, u16 count, u16 ability kind, u16 reserved,
// i32 x, i32 z (Q16.16), u32 target horde (0 = none), then count * u32 horde ids.
type CastCommand struct {
	Ability     uint16
	X, Z        int32
	TargetHorde uint32
	Hordes      []uint32
}

func DecodeCast(frame []byte) (CastCommand, error) {
	le := binary.LittleEndian
	if len(frame) < 20 || frame[0] != MsgCmdCast || frame[1] != Version {
		return CastCommand{}, ErrBadFrame
	}
	n := int(le.Uint16(frame[2:]))
	if n > maxHordesPerCommand || len(frame) < 20+n*4 {
		return CastCommand{}, ErrBadFrame
	}
	c := CastCommand{
		Ability:     le.Uint16(frame[4:]),
		X:           int32(le.Uint32(frame[8:])),
		Z:           int32(le.Uint32(frame[12:])),
		TargetHorde: le.Uint32(frame[16:]),
		Hordes:      make([]uint32, n),
	}
	for i := range c.Hordes {
		c.Hordes[i] = le.Uint32(frame[20+i*4:])
	}
	return c, nil
}

// SwitchCommand (toggle / autocast): u8 type, u8 version, u16 count,
// u16 ability kind, u16 value (autocast: on; mode abilities: the mode index),
// then count * u32 horde ids.
type SwitchCommand struct {
	Ability uint16
	On      bool
	Value   uint16
	Hordes  []uint32
}

func DecodeSwitch(frame []byte) (SwitchCommand, error) {
	le := binary.LittleEndian
	if len(frame) < 8 || (frame[0] != MsgCmdToggle && frame[0] != MsgCmdAuto) || frame[1] != Version {
		return SwitchCommand{}, ErrBadFrame
	}
	n := int(le.Uint16(frame[2:]))
	if n > maxHordesPerCommand || len(frame) < 8+n*4 {
		return SwitchCommand{}, ErrBadFrame
	}
	c := SwitchCommand{Ability: le.Uint16(frame[4:]), Value: le.Uint16(frame[6:]), Hordes: make([]uint32, n)}
	c.On = c.Value != 0
	for i := range c.Hordes {
		c.Hordes[i] = le.Uint32(frame[8+i*4:])
	}
	return c, nil
}

// DecodeAbility: u8 type, u8 version, u16 count, then count * u32 horde ids.
func DecodeAbility(frame []byte) ([]uint32, error) {
	le := binary.LittleEndian
	if len(frame) < 4 || frame[0] != MsgCmdAbility || frame[1] != Version {
		return nil, ErrBadFrame
	}
	n := int(le.Uint16(frame[2:]))
	if n > maxHordesPerCommand || len(frame) < 4+n*4 {
		return nil, ErrBadFrame
	}
	ids := make([]uint32, n)
	for i := range ids {
		ids[i] = le.Uint32(frame[4+i*4:])
	}
	return ids, nil
}

// ConsoleCommand is a dev console line with the client's context.
//
//	u8 type, u8 version, u16 selected count, i32 cursorX, i32 cursorZ (Q16.16),
//	u16 text length, u16 reserved, selected * u32 horde ids, text (UTF-8)
type ConsoleCommand struct {
	CursorX, CursorZ int32
	Selected         []uint32
	Text             string
}

const maxConsoleText = 1024

func DecodeConsole(frame []byte) (ConsoleCommand, error) {
	le := binary.LittleEndian
	if len(frame) < 16 || frame[0] != MsgCmdConsole || frame[1] != Version {
		return ConsoleCommand{}, ErrBadFrame
	}
	n := int(le.Uint16(frame[2:]))
	textLen := int(le.Uint16(frame[12:]))
	if n > maxHordesPerCommand || textLen > maxConsoleText || len(frame) < 16+n*4+textLen {
		return ConsoleCommand{}, ErrBadFrame
	}
	cmd := ConsoleCommand{
		CursorX:  int32(le.Uint32(frame[4:])),
		CursorZ:  int32(le.Uint32(frame[8:])),
		Selected: make([]uint32, n),
	}
	for i := range cmd.Selected {
		cmd.Selected[i] = le.Uint32(frame[16+i*4:])
	}
	cmd.Text = string(frame[16+n*4 : 16+n*4+textLen])
	return cmd, nil
}

// Console output levels.
const (
	ConsoleInfo  uint8 = 0
	ConsoleError uint8 = 1
)

// EncodeConsoleOutput: u8 type, u8 version, u8 level, u8 reserved, u32 length, text (UTF-8).
func EncodeConsoleOutput(level uint8, text string) []byte {
	buf := make([]byte, 8+len(text))
	buf[0] = MsgConsoleOutput
	buf[1] = Version
	buf[2] = level
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(text)))
	copy(buf[8:], text)
	return buf
}

// MessageType returns the type byte of a frame (0 if empty).
func MessageType(frame []byte) uint8 {
	if len(frame) == 0 {
		return 0
	}
	return frame[0]
}
