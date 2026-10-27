package sim

import (
	"fmt"
	"slices"
)

// Replay is everything needed to re-run a match: the setup, every command with
// the tick it was applied at, and state hashes to verify against.
type Replay struct {
	Version     int                `json:"version"`
	MapID       string             `json:"map"`
	Players     []PlayerSetup      `json:"players"`
	Commands    []RecordedCommand  `json:"commands"`
	Checkpoints []ReplayCheckpoint `json:"checkpoints"`
	FinalTick   uint32             `json:"finalTick"`
}

// ReplayVersion changes whenever simulation rules change in a way that makes
// old replays play out differently.
const ReplayVersion = 1

// RecordedCommand is a command and the tick it was applied at.
type RecordedCommand struct {
	Tick    uint32  `json:"tick"`
	Command Command `json:"command"`
}

// ReplayCheckpoint is a state hash taken after a tick.
type ReplayCheckpoint struct {
	Tick uint32 `json:"tick"`
	Hash uint64 `json:"hash"`
}

// Recorder collects commands and checkpoints while a match runs.
type Recorder struct {
	replay Replay
}

func NewRecorder(mapID string, players []PlayerSetup) *Recorder {
	return &Recorder{replay: Replay{Version: ReplayVersion, MapID: mapID, Players: slices.Clone(players)}}
}

// Record stores the commands applied at tick (call before World.Step).
func (r *Recorder) Record(tick uint32, cmds []Command) {
	for _, c := range cmds {
		c.Hordes = slices.Clone(c.Hordes)
		c.Buildings = slices.Clone(c.Buildings)
		r.replay.Commands = append(r.replay.Commands, RecordedCommand{Tick: tick, Command: c})
	}
}

// Checkpoint stores the world's state hash (call after World.Step).
// Returns the hash when one was taken this tick.
func (r *Recorder) Checkpoint(w *World) (uint64, bool) {
	r.replay.FinalTick = w.Tick
	if w.Tick%HashInterval != 0 {
		return 0, false
	}
	hash := w.StateHash()
	r.replay.Checkpoints = append(r.replay.Checkpoints, ReplayCheckpoint{Tick: w.Tick, Hash: hash})
	return hash, true
}

// Replay returns the recording so far.
func (r *Recorder) Replay() *Replay { return &r.replay }

// Verify re-runs the replay on a fresh world (from newWorld, already set up)
// and checks every checkpoint. It returns the first mismatch.
func (rp *Replay) Verify(newWorld func(mapID string, players []PlayerSetup) (*World, error)) error {
	if rp.Version != ReplayVersion {
		return fmt.Errorf("replay version %d, simulation is version %d", rp.Version, ReplayVersion)
	}
	w, err := newWorld(rp.MapID, rp.Players)
	if err != nil {
		return err
	}
	ci, next := 0, 0
	var cmds []Command
	for w.Tick < rp.FinalTick {
		cmds = cmds[:0]
		for next < len(rp.Commands) && rp.Commands[next].Tick == w.Tick {
			cmds = append(cmds, rp.Commands[next].Command)
			next++
		}
		w.Step(cmds)
		for ci < len(rp.Checkpoints) && rp.Checkpoints[ci].Tick == w.Tick {
			if got := w.StateHash(); got != rp.Checkpoints[ci].Hash {
				return fmt.Errorf("desync at tick %d: hash %#x, recorded %#x", w.Tick, got, rp.Checkpoints[ci].Hash)
			}
			ci++
		}
	}
	return nil
}
