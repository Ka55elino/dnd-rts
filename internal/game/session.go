package game

import (
	"fmt"
	"math"
	"slices"
	"time"

	"dnd-rts/internal/data"
	"dnd-rts/internal/devconsole"
	"dnd-rts/internal/fixed"
	"dnd-rts/internal/protocol"
	"dnd-rts/internal/sim"
)

// Session runs one match: a fixed-rate tick loop that applies queued commands,
// steps the world and publishes snapshots.
type Session struct {
	world    *sim.World
	recorder *sim.Recorder
	commands chan sim.Command
	console  chan consoleLine
	snapshot func(frame []byte) bool // returns false when the client is gone

	// Dev controls (tick goroutine only).
	paused bool
	steps  int // ticks to run while paused
	speed  float64
	ticker *time.Ticker
	local  sim.PlayerID  // the player this client controls; every other player is AI
	held   []sim.Command // console commands waiting for the next tick

	hashTick uint32 // tick of the latest state hash
	hash     uint64
}

// NewSession wraps a ready world. publish is called once per tick with an
// encoded snapshot. recorder may be nil.
func NewSession(world *sim.World, recorder *sim.Recorder, local sim.PlayerID, publish func(frame []byte) bool) *Session {
	return &Session{
		local:    local,
		world:    world,
		recorder: recorder,
		commands: make(chan sim.Command, 256),
		console:  make(chan consoleLine, 16),
		snapshot: publish,
		speed:    1,
	}
}

// Replay returns the match recording (nil if not recording).
func (s *Session) Replay() *sim.Replay {
	if s.recorder == nil {
		return nil
	}
	return s.recorder.Replay()
}

// Submit queues a command for the next tick. Safe from any goroutine.
// Drops the command if the queue is full (client spamming).
func (s *Session) Submit(c sim.Command) {
	select {
	case s.commands <- c:
	default:
	}
}

type consoleLine struct {
	ctx  devconsole.Context
	text string
}

// SubmitConsole queues a dev console line; it runs on the tick goroutine and
// answers with a console output frame. Safe from any goroutine.
func (s *Session) SubmitConsole(ctx devconsole.Context, text string) {
	select {
	case s.console <- consoleLine{ctx, text}:
	default:
	}
}

// devconsole.Control (called on the tick goroutine).
func (s *Session) Local() sim.PlayerID { return s.local }
func (s *Session) SetLocal(p sim.PlayerID) error {
	if !slices.Contains(s.world.Players, p) {
		return fmt.Errorf("player %d is not in this match", p)
	}
	s.local = p
	return nil
}
func (s *Session) Paused() bool     { return s.paused }
func (s *Session) SetPaused(p bool) { s.paused, s.steps = p, 0 }
func (s *Session) Step(n int)       { s.steps += n }
func (s *Session) Speed() float64   { return s.speed }
func (s *Session) SetSpeed(v float64) {
	s.speed = v
	s.ticker.Reset(time.Duration(float64(time.Second) / sim.TickRate / v))
}

// runConsole executes queued console lines; their commands join this tick.
func (s *Session) runConsole(pending []sim.Command) ([]sim.Command, bool) {
	for {
		select {
		case l := <-s.console:
			l.ctx.Player = s.local
			r := devconsole.Run(s.world, s, l.ctx, l.text)
			pending = append(pending, r.Commands...)
			level := protocol.ConsoleInfo
			if r.Err {
				level = protocol.ConsoleError
			}
			if r.Output != "" && !s.snapshot(protocol.EncodeConsoleOutput(level, r.Output)) {
				return pending, false
			}
		default:
			return pending, true
		}
	}
}

// Run ticks until done is closed or the client disconnects.
func (s *Session) Run(done <-chan struct{}) {
	s.ticker = time.NewTicker(time.Second / sim.TickRate)
	defer s.ticker.Stop()

	s.publish(0) // initial state right away
	var pending []sim.Command
	for {
		select {
		case <-done:
			return
		case <-s.ticker.C:
			var ok bool
			if s.held, ok = s.runConsole(s.held); !ok {
				return
			}
			if s.paused && s.steps == 0 {
				// Player commands wait in their channel, console commands in
				// s.held; keep the client fed with the frozen state.
				if !s.publish(0) {
					return
				}
				continue
			}
			if s.steps > 0 {
				s.steps--
			}
			pending = append(pending[:0], s.held...)
			s.held = s.held[:0]
		drain:
			for {
				select {
				case c := <-s.commands:
					c.Player = s.local // the client always acts as the player it controls
					pending = append(pending, c)
				default:
					break drain
				}
			}

			if s.recorder != nil {
				s.recorder.Record(s.world.Tick, pending)
			}
			start := time.Now()
			s.world.Step(pending)
			simMicros := uint32(time.Since(start).Microseconds())

			if s.recorder != nil {
				if hash, ok := s.recorder.Checkpoint(s.world); ok {
					s.hashTick, s.hash = s.world.Tick, hash
				}
			} else if s.world.Tick%sim.HashInterval == 0 {
				s.hashTick, s.hash = s.world.Tick, s.world.StateHash()
			}

			if !s.publish(simMicros) {
				return
			}
		}
	}
}

func (s *Session) publish(simMicros uint32) bool {
	w := s.world
	hordes := make([]protocol.HordeState, len(w.Hordes))
	for i, h := range w.Hordes {
		var hp, maxHP fixed.Fixed
		for _, so := range h.Soldiers {
			hp += max(so.HP, 0)
			maxHP = so.MaxHP()
		}
		var level uint8
		if h.IsHero() {
			level = uint8(h.Level())
		}
		hordes[i] = protocol.HordeState{
			HP:    uint32(hp.Int()),
			MaxHP: uint32(maxHP.MulInt(h.Def.Size).Int()),
			Level: level,
			Flags: hordeFlags(h),
			Scale: uint16(h.FormScale().MulInt(100).Int()),
			XP:    uint32(h.XP()),
			ID:    uint32(h.ID),
			Owner: uint8(h.Owner),
			Kind:  uint8(h.Kind),
			Alive: uint16(len(h.Soldiers)),
			X:     h.Pos.X.Float32(),
			Z:     h.Pos.Y.Float32(),
		}
	}
	soldiers := make([]protocol.SoldierState, len(w.Soldiers))
	for i, so := range w.Soldiers {
		state := protocol.SoldierIdle
		switch {
		case so.Knocked():
			state = protocol.SoldierKnocked
		case so.Swinging():
			state = protocol.SoldierAttacking
		case so.Horde.IsMoving():
			state = protocol.SoldierMoving
		}
		soldiers[i] = protocol.SoldierState{
			ID:     uint32(so.ID),
			Horde:  uint32(so.Horde.ID),
			X:      so.Pos.X.Float32(),
			Z:      so.Pos.Y.Float32(),
			Rot:    renderRotation(so.Rot),
			HP:     uint8(so.HP.Clamp(0, so.MaxHP()).MulInt(255).Div(so.MaxHP()).Int()),
			State:  state,
			Status: so.StatusFlags(),
		}
	}
	projectiles := make([]protocol.ProjectileState, len(w.Projectiles))
	for i, p := range w.Projectiles {
		projectiles[i] = protocol.ProjectileState{
			ID:     uint32(p.ID),
			Owner:  uint8(p.Owner),
			Arrows: uint16(p.Arrows),
			Flags:  projectileFlags(p),
			FromX:  p.From.X.Float32(),
			FromZ:  p.From.Y.Float32(),
			ToX:    p.To.X.Float32(),
			ToZ:    p.To.Y.Float32(),
			Launch: p.LaunchTick,
			Impact: p.ImpactTick,
		}
	}
	buildings := make([]protocol.BuildingState, len(w.Buildings))
	for i, b := range w.Buildings {
		buildings[i] = protocol.BuildingState{
			ID:    uint32(b.ID),
			Owner: uint8(b.Owner),
			Kind:  uint16(b.Kind),
			X:     b.Pos.X.Float32(),
			Z:     b.Pos.Y.Float32(),
			Rot:   renderRotation(b.Facing),
			HP:    uint32(max(b.HP, 0).Int()),
			MaxHP: uint32(b.Def.Health.Int()),
		}
		by, capture := b.Capturing()
		buildings[i].Capturer = uint8(by)
		if by != sim.Neutral {
			buildings[i].Flags = protocol.BuildingFlags(false, true, capture.Float(), uint16(plotMask(b)))
		} else {
			buildings[i].Flags = protocol.BuildingFlags(b.UnderConstruction(), false, b.Progress().Float(), uint16(plotMask(b)))
		}
		buildings[i].PlotsHi = uint32(plotMask(b) >> 16)
		if b.GateOpen() {
			buildings[i].Flags |= protocol.BuildingOpen
		}
		if b.ToggleOn() {
			buildings[i].Flags |= protocol.BuildingToggled
		}
		if b.Burning() {
			buildings[i].Flags |= protocol.BuildingBurning
		}
	}
	return s.snapshot(protocol.EncodeGameSnapshot(protocol.SnapshotHeader{
		Tick:        w.Tick,
		SimMicros:   simMicros,
		HashTick:    s.hashTick,
		Hash:        s.hash,
		Over:        w.Over,
		Winner:      uint8(w.Winner),
		LocalPlayer: uint8(s.local),
	}, &protocol.GameState{
		Hordes: hordes, Soldiers: soldiers, Projectiles: projectiles, Buildings: buildings,
		Players: s.players(), Resources: s.resources(), Queues: s.queues(),
		Events: s.events(), Abilities: s.abilities(), Upgrades: s.upgrades(),
		Heroes: s.heroes(),
	}))
}

// upgrades researched by the local player.
func (s *Session) upgrades() []uint16 {
	var out []uint16
	for _, id := range s.world.Upgrades(s.local) {
		out = append(out, uint16(slices.Index(s.world.Catalog.UpgradeIDs, id)))
	}
	return out
}

// heroes: the local player's unique hordes, alive or waiting for revival.
func (s *Session) heroes() []protocol.HeroState {
	var out []protocol.HeroState
	for _, h := range s.world.Heroes(s.local) {
		out = append(out, protocol.HeroState{Kind: uint16(slices.Index(s.world.Catalog.HordeIDs, h.Horde)),
			Level: uint8(h.Level), Alive: h.Alive, XP: uint32(h.XP)})
	}
	return out
}

// events: this tick's ability casts (visuals).
func (s *Session) events() []protocol.EventState {
	out := make([]protocol.EventState, len(s.world.Events))
	for i, e := range s.world.Events {
		kind := uint8(e.Kind)
		if t := slices.Index(data.DamageTypes, e.DamageType); t >= 0 {
			kind |= uint8(t+1) << 4
		}
		ev := protocol.EventState{Ability: uint16(e.Ability), Owner: uint8(e.Owner), Kind: kind,
			X: e.Pos.X.Float32(), Z: e.Pos.Y.Float32(), Radius: e.Radius.Float32(), Dur: e.Duration.Float32()}
		switch e.Kind {
		case sim.EventChain:
			ev.Radius, ev.Dur = e.To.X.Float32(), e.To.Y.Float32()
		case sim.EventCone:
			ev.Dur = e.Angle.Float32()
			ev.Facing = renderRotation(e.Facing)
		}
		out[i] = ev
	}
	return out
}

// abilities: cooldowns and switches of the local player's hordes.
func (s *Session) abilities() []protocol.HordeAbilities {
	var out []protocol.HordeAbilities
	for _, h := range s.world.Hordes {
		if h.Owner != s.local {
			continue
		}
		st := h.AbilityStates()
		if len(st) == 0 {
			continue
		}
		rec := protocol.HordeAbilities{Horde: uint32(h.ID), Slots: make([]protocol.AbilitySlot, len(st))}
		for i, a := range st {
			var f uint8
			if a.Toggled {
				f |= protocol.AbilityToggled
			}
			if a.Autocast {
				f |= protocol.AbilityAutocast
			}
			if a.Pending {
				f |= protocol.AbilityPending
			}
			if a.Locked {
				f |= protocol.AbilityLocked
			}
			rec.Slots[i] = protocol.AbilitySlot{Cooldown: uint16(min(a.Cooldown, 0xFFFF)), Flags: f, Mode: uint8(a.Mode)}
		}
		out = append(out, rec)
	}
	return out
}

// players lists every player with its faction and controller.
func (s *Session) players() []protocol.PlayerState {
	w := s.world
	out := make([]protocol.PlayerState, len(w.Players))
	for i, p := range w.Players {
		st := protocol.PlayerState{ID: uint8(p)}
		if f := w.FactionOf(p); f != nil {
			st.Faction = uint8(slices.Index(w.Catalog.FactionIDs, f.ID))
		}
		if p == s.local {
			st.Flags |= protocol.PlayerLocal
		} else {
			st.Flags |= protocol.PlayerAI
		}
		if w.IsDefeated(p) {
			st.Flags |= protocol.PlayerDefeated
		}
		out[i] = st
	}
	return out
}

// resources of the local player, in sorted resource id order.
func (s *Session) resources() []protocol.ResourceState {
	w := s.world
	out := make([]protocol.ResourceState, len(w.Catalog.ResourceIDs))
	for r, id := range w.Catalog.ResourceIDs {
		if w.Catalog.Resources[id].Kind == data.ResourceCap {
			out[r] = protocol.ResourceState{Amount: int32(w.CapUsed(s.local, r)), Limit: int32(w.CapProvided(s.local, r))}
		} else {
			out[r] = protocol.ResourceState{Amount: int32(w.Stock(s.local, r))}
		}
	}
	return out
}

// queues of the local player's buildings.
func (s *Session) queues() []protocol.QueueState {
	w := s.world
	var out []protocol.QueueState
	for _, b := range w.Buildings {
		q := b.Queue()
		if b.Owner != s.local || q == nil {
			continue
		}
		st := protocol.QueueState{Building: uint32(b.ID), Progress: uint16(min(q.Progress.Float(), 1) * 65535)}
		for i, id := range q.Hordes {
			if id != "" {
				st.Hordes = append(st.Hordes, uint16(slices.Index(w.Catalog.HordeIDs, id)))
			} else {
				st.Hordes = append(st.Hordes, protocol.QueueResearch|uint16(slices.Index(w.Catalog.UpgradeIDs, q.Upgrades[i])))
			}
		}
		out = append(out, st)
	}
	return out
}

// plotMask has bit i set when plot i of b is occupied.
func plotMask(b *sim.Building) uint64 {
	var m uint64
	for i := range min(len(b.Def.Plots), 48) {
		if b.PlotOccupied(i) {
			m |= 1 << i
		}
	}
	return m
}

// renderRotation converts a sim heading (angle of (cos a, sin a) on the X/Z
// plane) to a three.js Y rotation for a model facing +Z: θ = π/2 - a.
func renderRotation(a fixed.Fixed) float32 {
	return float32(math.Pi/2 - a.Float())
}

func projectileFlags(p *sim.Projectile) uint8 {
	var f uint8
	if p.FromAir {
		f |= protocol.ProjectileFromAir
	}
	if p.ToAir {
		f |= protocol.ProjectileToAir
	}
	if p.FromWall {
		f |= protocol.ProjectileFromWall
	}
	if p.Splash > 0 {
		f |= protocol.ProjectileSplash
	}
	return f
}

func hordeFlags(h *sim.Horde) uint8 {
	var f uint8
	if h.Transformed() {
		f |= protocol.HordeTransformed
	}
	if h.Summoned() {
		f |= protocol.HordeSummoned
	}
	return f
}
