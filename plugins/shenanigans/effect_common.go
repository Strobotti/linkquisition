//nolint:mnd // Visual effects plugin: magic numbers are by design.
package main

import (
	"time"

	"github.com/strobotti/linkquisition"
)

// effectState is the interface each effect's state struct must implement
// to use the shared startEffect helper.
type effectState interface {
	// init initializes or resets the effect state for the given dimensions.
	init(width, height int)
	// update advances the effect by one frame.
	update()
	// render produces an RGBA pixel buffer for the current frame.
	render() []uint8
}

// effectConfig holds the parameters for starting an effect via the shared helper.
type effectConfig struct {
	state         effectState
	opacity       float64
	frameInterval time.Duration
}

// startEffect wires up the common boilerplate for all raster-overlay effects:
// canvas size detection, overlay registration, resize handling, and the
// ticker goroutine. Effects only need to implement effectState.
func (p *shenanigans) startEffect(pc linkquisition.PickerCanvas, cfg effectConfig) {
	w := pc.Width()
	h := pc.Height()
	if w == 0 {
		w = 600
	}
	if h == 0 {
		h = 400
	}

	state := cfg.state
	state.init(w, h)

	currentW, currentH := w, h

	pc.AddRasterOverlay(cfg.opacity, func(newW, newH int) []uint8 {
		if newW != currentW || newH != currentH {
			currentW = newW
			currentH = newH
			state.init(newW, newH)
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(cfg.frameInterval)
		defer ticker.Stop()

		for range ticker.C {
			if p.stopped.Load() {
				return
			}
			state.update()
			pc.ScheduleRefresh()
		}
	}()
}
