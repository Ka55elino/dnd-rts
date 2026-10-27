package game

import (
	"encoding/json"
	"os"
	"path/filepath"

	"dnd-rts/internal/sim"
)

// ReplayDir is where finished matches are saved: <user config>/dnd-rts/replays.
func ReplayDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "dnd-rts", "replays"), nil
}

// SaveReplay writes the replay as replays/last.json (overwriting the previous one).
func SaveReplay(r *sim.Replay) error {
	if r == nil || r.FinalTick == 0 {
		return nil
	}
	dir, err := ReplayDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "last.json"), data, 0o644)
}
