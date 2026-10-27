package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"dnd-rts/content"
	"dnd-rts/internal/bench"
	"dnd-rts/internal/game"
	"dnd-rts/internal/services"
	"dnd-rts/internal/terrain"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	settings := services.NewSettingsService()

	contentSvc, err := services.NewContentService(content.FS)
	if err != nil {
		log.Fatal(err)
	}

	app := application.New(application.Options{
		Name:        "dnd-rts",
		Description: "Real-time strategy game",
		Services: []application.Service{
			application.NewService(settings),
			application.NewService(contentSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Game sessions: binary snapshots Go → JS, commands JS → Go.
	gameServer := &game.Server{
		Catalog: services.Catalog(contentSvc),
		Terrain: func(mapID string) (*terrain.Heightfield, error) { return services.Heightfield(contentSvc, mapID) },
		// TODO: off in release builds / multiplayer unless cheats are enabled.
		DevConsole: true,
	}
	app.HandleStream(game.StreamName, gameServer.Handle)

	// Dev benchmark: binary snapshots Go → JS (see docs/ROADMAP.md, phase 0).
	app.HandleStream(bench.StreamName, bench.Handle)

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "dnd-rts",
		URL:   "/",

		// Saved display mode (fullscreen by default). Width/Height apply in windowed mode.
		StartState: services.StartState(settings),
		Width:      1280,
		Height:     720,
		MinWidth:   1024,
		MinHeight:  600,

		BackgroundColour: application.NewRGB(10, 10, 12),

		// Right click is a game command (move/attack), not a browser menu.
		DefaultContextMenuDisabled: true,

		KeyBindings: map[string]func(window application.Window){
			"F11": func(application.Window) { services.ToggleDisplayMode(settings) },
		},

		Mac: application.MacWindow{
			// Esc is used by the game (menus, cancel orders), don't let macOS exit fullscreen on it.
			DisableEscapeExitsFullscreen: true,
		},
		Linux: application.LinuxWindow{
			// WebGL needs hardware acceleration on WebKitGTK.
			WebviewGpuPolicy: application.WebviewGpuPolicyAlways,
		},
	})
	services.AttachWindow(settings, window)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
