package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type DisplayMode string

const (
	DisplayFullscreen DisplayMode = "fullscreen"
	DisplayWindowed   DisplayMode = "windowed"
)

// Settings are user preferences persisted between launches.
type Settings struct {
	DisplayMode DisplayMode `json:"displayMode"`
}

func defaultSettings() Settings {
	return Settings{DisplayMode: DisplayFullscreen}
}

// SettingsService loads/saves settings and applies them to the main window.
// Exposed to the frontend via Wails bindings.
type SettingsService struct {
	mu       sync.Mutex
	path     string
	settings Settings
	window   *application.WebviewWindow
}

// NewSettingsService loads settings from the user config dir, falling back to defaults.
func NewSettingsService() *SettingsService {
	s := &SettingsService{settings: defaultSettings()}

	dir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("settings: no user config dir, using defaults: %v", err)
		return s
	}
	s.path = filepath.Join(dir, "dnd-rts", "settings.json")

	data, err := os.ReadFile(s.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// First launch: keep defaults.
	case err != nil:
		log.Printf("settings: read %s: %v", s.path, err)
	default:
		if err := json.Unmarshal(data, &s.settings); err != nil {
			log.Printf("settings: parse %s: %v", s.path, err)
			s.settings = defaultSettings()
		}
	}
	if !s.settings.DisplayMode.valid() {
		s.settings.DisplayMode = DisplayFullscreen
	}
	return s
}

// Package-level helpers below are deliberately not methods: Wails binds every exported
// method of a service to the frontend, and these are for Go-side setup only.

// StartState returns the window state matching the saved display mode.
func StartState(s *SettingsService) application.WindowState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settings.DisplayMode == DisplayWindowed {
		return application.WindowStateNormal
	}
	return application.WindowStateFullscreen
}

// AttachWindow binds the service to the main window and keeps the saved mode in sync
// when fullscreen is changed outside the options menu (F11, macOS green button).
func AttachWindow(s *SettingsService, w *application.WebviewWindow) {
	s.mu.Lock()
	s.window = w
	s.mu.Unlock()

	w.OnWindowEvent(events.Common.WindowFullscreen, func(*application.WindowEvent) {
		s.syncMode(DisplayFullscreen)
	})
	w.OnWindowEvent(events.Common.WindowUnFullscreen, func(*application.WindowEvent) {
		s.syncMode(DisplayWindowed)
	})
}

// Get returns the current settings.
func (s *SettingsService) Get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

// SetDisplayMode switches between fullscreen and windowed and saves the choice.
func (s *SettingsService) SetDisplayMode(mode DisplayMode) error {
	if !mode.valid() {
		return fmt.Errorf("unknown display mode %q", mode)
	}

	s.mu.Lock()
	s.settings.DisplayMode = mode
	w := s.window
	err := s.saveLocked()
	s.mu.Unlock()

	if w != nil {
		if mode == DisplayFullscreen {
			w.Fullscreen()
		} else {
			w.UnFullscreen()
		}
	}
	return err
}

// ToggleDisplayMode flips fullscreen/windowed (used by the F11 binding).
func ToggleDisplayMode(s *SettingsService) {
	next := DisplayFullscreen
	if s.Get().DisplayMode == DisplayFullscreen {
		next = DisplayWindowed
	}
	if err := s.SetDisplayMode(next); err != nil {
		log.Printf("settings: %v", err)
	}
}

func (s *SettingsService) syncMode(mode DisplayMode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settings.DisplayMode == mode {
		return
	}
	s.settings.DisplayMode = mode
	if err := s.saveLocked(); err != nil {
		log.Printf("settings: %v", err)
	}
}

func (s *SettingsService) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return err
	}
	// Write to a temp file and rename, so a crash never leaves a half-written file.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return os.Rename(tmp, s.path)
}

func (m DisplayMode) valid() bool {
	return m == DisplayFullscreen || m == DisplayWindowed
}
