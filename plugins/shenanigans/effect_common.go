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
	skipInvert    bool // if true, skip invertForLight (for effects with custom palettes)
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
		pixels := state.render()
		if cfg.skipInvert {
			return pixels
		}
		return p.invertForLight(pixels)
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

// simpleEffectState is the interface for effects that don't track dimensions
// in their state — they receive width/height on every render call.
type simpleEffectState interface {
	// update advances the effect by one frame.
	update()
	// render produces an RGBA pixel buffer for the given dimensions.
	render(width, height int) []uint8
}

// simpleEffectConfig holds parameters for starting a simple (stateless-dimension) effect.
type simpleEffectConfig struct {
	state         simpleEffectState
	opacity       float64
	frameInterval time.Duration
}

// startSimpleEffect wires up the boilerplate for effects that don't store
// width/height in their state. These receive the current dimensions on every
// render call instead.
func (p *shenanigans) startSimpleEffect(pc linkquisition.PickerCanvas, cfg simpleEffectConfig) {
	state := cfg.state

	pc.AddRasterOverlay(cfg.opacity, func(w, h int) []uint8 {
		return state.render(w, h)
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

// drawRect draws a filled rectangle into an RGBA pixel buffer with bounds checking.
// Pixels are only written if the new alpha exceeds the existing alpha at that position.
func drawRect(pixels []uint8, bufW, bufH, x, y, rw, rh int, r, g, b, a uint8) {
	for dy := range rh {
		py := y + dy
		if py < 0 || py >= bufH {
			continue
		}
		rowOffset := py * bufW * rgbaChannels
		for dx := range rw {
			px := x + dx
			if px < 0 || px >= bufW {
				continue
			}
			offset := rowOffset + px*rgbaChannels
			if a > pixels[offset+3] {
				pixels[offset] = r
				pixels[offset+1] = g
				pixels[offset+2] = b
				pixels[offset+3] = a
			}
		}
	}
}
