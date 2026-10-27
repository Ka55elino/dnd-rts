package bench

import (
	"errors"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"dnd-rts/internal/protocol"
)

const (
	// StreamName is what the frontend passes to Stream(...).
	StreamName = "bench"

	tickRate     = 20
	tickInterval = time.Second / tickRate
	defaultCount = 10_000
)

// Handle runs one benchmark session per frontend connection.
// The world ticks at 20 Hz and every tick is sent as a binary snapshot.
func Handle(c *application.StreamConn) {
	defer c.Close()

	counts := make(chan int, 4)
	go readCommands(c, counts)

	world := NewWorld(1)
	world.SetCount(defaultCount)

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	var tick uint32
	dropped := 0
	for {
		select {
		case <-c.Context().Done():
			return
		case n := <-counts:
			world.SetCount(n)
		case <-ticker.C:
			start := time.Now()
			world.Step(float32(tickInterval.Seconds()))
			simMicros := uint32(time.Since(start).Microseconds())

			tick++
			frame := protocol.EncodeSnapshot(tick, simMicros, world.X, world.Z, world.Rot)

			// Never block the tick loop on a slow frontend: drop the frame instead.
			switch err := c.TrySend(frame); {
			case err == nil:
			case errors.Is(err, application.ErrStreamFull):
				dropped++
				if dropped%20 == 1 {
					log.Printf("bench: frontend is behind, dropped %d snapshots", dropped)
				}
			default:
				return // connection closed
			}
		}
	}
}

func readCommands(c *application.StreamConn, counts chan<- int) {
	for {
		frame, err := c.Receive()
		if err != nil {
			return
		}
		if n, ok := protocol.DecodeBenchSetCount(frame); ok {
			select {
			case counts <- n:
			case <-c.Context().Done():
				return
			}
		}
	}
}
