package game

import (
	"errors"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"dnd-rts/internal/data"
	"dnd-rts/internal/devconsole"
	"dnd-rts/internal/fixed"
	"dnd-rts/internal/protocol"
	"dnd-rts/internal/sim"
	"dnd-rts/internal/terrain"
)

// StreamName is what the frontend passes to Stream(...).
const StreamName = "game"

// LocalPlayer is the player the client controls at the start (the dev
// console can switch). Commands from the client carry no player: the session
// stamps them with whoever the client controls.
const LocalPlayer sim.PlayerID = 0

// Server creates a single-player session for each frontend connection.
type Server struct {
	Catalog *data.Catalog
	// Terrain returns the (cached) heightfield of a map.
	Terrain func(mapID string) (*terrain.Heightfield, error)
	// DevConsole accepts developer console commands (spawn, kill, pause, ...).
	DevConsole bool
}

// Handle is a Wails stream handler: the first frame must be Hello{mapID}.
func (srv *Server) Handle(c *application.StreamConn) {
	defer c.Close()

	frame, err := c.Receive()
	if err != nil {
		return
	}
	mapID, err := protocol.DecodeHello(frame)
	if err != nil {
		log.Printf("game: bad hello: %v", err)
		return
	}

	players := []sim.PlayerSetup{
		{ID: LocalPlayer, Faction: "empire"},
		{ID: 1, Faction: "warband"},
	}
	world, err := srv.NewWorld(mapID, players)
	if err != nil {
		log.Printf("game: %v", err)
		return
	}

	session := NewSession(world, sim.NewRecorder(mapID, players), LocalPlayer, func(frame []byte) bool {
		err := c.TrySend(frame)
		if errors.Is(err, application.ErrStreamFull) {
			return true // frontend is behind: drop this snapshot, keep running
		}
		return err == nil
	})

	go srv.readCommands(c, session)
	session.Run(c.Context().Done())

	if err := SaveReplay(session.Replay()); err != nil {
		log.Printf("game: saving replay: %v", err)
	}
}

// NewWorld builds a set-up world for a map (also used to verify replays).
func (srv *Server) NewWorld(mapID string, players []sim.PlayerSetup) (*sim.World, error) {
	hf, err := srv.Terrain(mapID)
	if err != nil {
		return nil, err
	}
	w := sim.NewWorld(srv.Catalog, hf)
	return w, sim.SetupSkirmish(w, mapID, players)
}

func (srv *Server) readCommands(c *application.StreamConn, s *Session) {
	for {
		frame, err := c.Receive()
		if err != nil {
			return
		}
		switch protocol.MessageType(frame) {
		case protocol.MsgCmdMove:
			m, err := protocol.DecodeMove(frame)
			if err != nil {
				continue
			}
			s.Submit(sim.Command{
				Kind:   sim.CmdMove,
				Hordes: toHordeIDs(m.Hordes),
				Target: fixed.Vec2{X: fixed.FromRaw(int64(m.X)), Y: fixed.FromRaw(int64(m.Y))},
			})
		case protocol.MsgCmdAMove, protocol.MsgCmdPatrol, protocol.MsgCmdFireAt:
			m, err := protocol.DecodeMoveLike(frame)
			if err != nil {
				continue
			}
			kind := sim.CmdAttackMove
			switch protocol.MessageType(frame) {
			case protocol.MsgCmdPatrol:
				kind = sim.CmdPatrol
			case protocol.MsgCmdFireAt:
				kind = sim.CmdAttackGround
			}
			s.Submit(sim.Command{
				Kind:   kind,
				Hordes: toHordeIDs(m.Hordes),
				Target: fixed.Vec2{X: fixed.FromRaw(int64(m.X)), Y: fixed.FromRaw(int64(m.Y))},
			})
		case protocol.MsgCmdStop, protocol.MsgCmdHold:
			ids, err := protocol.DecodeHordeList(frame)
			if err != nil {
				continue
			}
			kind := sim.CmdStop
			if protocol.MessageType(frame) == protocol.MsgCmdHold {
				kind = sim.CmdHold
			}
			s.Submit(sim.Command{Kind: kind, Hordes: toHordeIDs(ids)})
		case protocol.MsgCmdAttack:
			a, err := protocol.DecodeAttack(frame)
			if err != nil {
				continue
			}
			s.Submit(sim.Command{
				Kind:        sim.CmdAttack,
				Hordes:      toHordeIDs(a.Hordes),
				TargetHorde: sim.HordeID(a.Target),
			})
		case protocol.MsgCmdSiege:
			a, err := protocol.DecodeSiege(frame)
			if err != nil {
				continue
			}
			s.Submit(sim.Command{
				Kind:           sim.CmdAttackBuilding,
				Hordes:         toHordeIDs(a.Hordes),
				TargetBuilding: sim.BuildingID(a.Target),
			})
		case protocol.MsgCmdPlot:
			p, err := protocol.DecodePlot(frame)
			if err != nil || int(p.Kind) >= len(srv.Catalog.BuildingIDs) {
				continue
			}
			s.Submit(sim.Command{
				Kind:           sim.CmdBuildOnPlot,
				TargetBuilding: sim.BuildingID(p.Host),
				Count:          int(p.Plot),
				Name:           srv.Catalog.BuildingIDs[p.Kind],
			})
		case protocol.MsgCmdCancel:
			id, err := protocol.DecodeCancel(frame)
			if err != nil {
				continue
			}
			s.Submit(sim.Command{Kind: sim.CmdCancelBuild, TargetBuilding: sim.BuildingID(id)})
		case protocol.MsgCmdBuild:
			b, err := protocol.DecodeBuild(frame)
			if err != nil || int(b.Kind) >= len(srv.Catalog.BuildingIDs) {
				continue
			}
			s.Submit(sim.Command{
				Kind:   sim.CmdBuildFree,
				Hordes: toHordeIDs(b.Hordes),
				Name:   srv.Catalog.BuildingIDs[b.Kind],
				Target: fixed.Vec2{X: fixed.FromRaw(int64(b.X)), Y: fixed.FromRaw(int64(b.Z))},
				Angle:  fixed.FromRaw(int64(b.Angle)),
			})
		case protocol.MsgCmdHelp:
			a, err := protocol.DecodeHelp(frame)
			if err != nil {
				continue
			}
			s.Submit(sim.Command{
				Kind:           sim.CmdConstruct,
				Hordes:         toHordeIDs(a.Hordes),
				TargetBuilding: sim.BuildingID(a.Target),
			})
		case protocol.MsgCmdConsole:
			if !srv.DevConsole {
				continue
			}
			cmd, err := protocol.DecodeConsole(frame)
			if err != nil {
				continue
			}
			s.SubmitConsole(devconsole.Context{
				Cursor:   fixed.Vec2{X: fixed.FromRaw(int64(cmd.CursorX)), Y: fixed.FromRaw(int64(cmd.CursorZ))},
				Selected: toHordeIDs(cmd.Selected),
			}, cmd.Text)
		case protocol.MsgCmdCast:
			c, err := protocol.DecodeCast(frame)
			if err != nil || int(c.Ability) >= len(srv.Catalog.AbilityIDs) {
				continue
			}
			s.Submit(sim.Command{
				Kind:        sim.CmdCast,
				Hordes:      toHordeIDs(c.Hordes),
				Name:        srv.Catalog.AbilityIDs[c.Ability],
				Target:      fixed.Vec2{X: fixed.FromRaw(int64(c.X)), Y: fixed.FromRaw(int64(c.Z))},
				TargetHorde: sim.HordeID(c.TargetHorde),
			})
		case protocol.MsgCmdToggle, protocol.MsgCmdAuto:
			c, err := protocol.DecodeSwitch(frame)
			if err != nil || int(c.Ability) >= len(srv.Catalog.AbilityIDs) {
				continue
			}
			cmd := sim.Command{Kind: sim.CmdToggle, Hordes: toHordeIDs(c.Hordes), Name: srv.Catalog.AbilityIDs[c.Ability], Count: int(c.Value)}
			if protocol.MessageType(frame) == protocol.MsgCmdAuto {
				cmd.Kind = sim.CmdAutocast
				if c.On {
					cmd.Count = 1
				}
			}
			s.Submit(cmd)
		case protocol.MsgCmdTrain:
			b, kind, err := protocol.DecodeTrain(frame)
			if err != nil || int(kind) >= len(srv.Catalog.HordeIDs) {
				continue
			}
			s.Submit(sim.Command{Kind: sim.CmdTrain, TargetBuilding: sim.BuildingID(b), Name: srv.Catalog.HordeIDs[kind]})
		case protocol.MsgCmdResearch:
			b, kind, err := protocol.DecodeResearch(frame)
			if err != nil || int(kind) >= len(srv.Catalog.UpgradeIDs) {
				continue
			}
			s.Submit(sim.Command{Kind: sim.CmdResearch, TargetBuilding: sim.BuildingID(b), Name: srv.Catalog.UpgradeIDs[kind]})
		case protocol.MsgCmdUntrain:
			b, i, err := protocol.DecodeUntrain(frame)
			if err != nil {
				continue
			}
			s.Submit(sim.Command{Kind: sim.CmdCancelTrain, TargetBuilding: sim.BuildingID(b), Count: int(i)})
		case protocol.MsgCmdBToggle:
			b, ab, err := protocol.DecodeBuildingToggle(frame)
			if err != nil || int(ab) >= len(srv.Catalog.AbilityIDs) {
				continue
			}
			s.Submit(sim.Command{Kind: sim.CmdToggle, TargetBuilding: sim.BuildingID(b), Name: srv.Catalog.AbilityIDs[ab]})
		case protocol.MsgCmdGate:
			b, open, err := protocol.DecodeGate(frame)
			if err != nil {
				continue
			}
			c := sim.Command{Kind: sim.CmdGate, TargetBuilding: sim.BuildingID(b)}
			if open {
				c.Count = 1
			}
			s.Submit(c)
		case protocol.MsgCmdRally:
			b, x, z, err := protocol.DecodeRally(frame)
			if err != nil {
				continue
			}
			s.Submit(sim.Command{Kind: sim.CmdRally, TargetBuilding: sim.BuildingID(b),
				Target: fixed.Vec2{X: fixed.FromRaw(int64(x)), Y: fixed.FromRaw(int64(z))}})
		}
	}
}

func toHordeIDs(ids []uint32) []sim.HordeID {
	out := make([]sim.HordeID, len(ids))
	for i, id := range ids {
		out[i] = sim.HordeID(id)
	}
	return out
}
