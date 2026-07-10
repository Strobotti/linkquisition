//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Gentle snowfall with wind drift and wobble, accumulating at the bottom.
// --- Snow Effect ---

const (
	snowFlakeCount = 150
	snowMaxSize    = 4
)

type snowflake struct {
	x, y   float64
	size   float64
	speed  float64
	drift  float64
	wobble float64
	phase  float64
}

type snowState struct {
	flakes      []snowflake
	width       int
	height      int
	initialized bool
}

func (p *shenanigans) startSnow(pc linkquisition.PickerCanvas) {
	state := &snowState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}

	pc.AddRasterOverlay(0.3, func(w, h int) []uint8 {
		if !state.initialized || w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.init()
			state.initialized = true
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
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

func (s *snowState) init() {
	s.flakes = make([]snowflake, snowFlakeCount)
	for i := range s.flakes {
		// Start most flakes above the window so they drift in gradually.
		// A few start on-screen so it's not completely empty at first.
		onScreen := i < snowFlakeCount/5
		s.flakes[i] = s.newFlake(onScreen)
	}
}

func (s *snowState) newFlake(onScreen bool) snowflake {
	// Default: spawn above the viewport at varying distances
	y := -(rand.Float64() * float64(s.height))
	if onScreen {
		y = rand.Float64() * float64(s.height)
	}

	return snowflake{
		x:      rand.Float64() * float64(s.width),
		y:      y,
		size:   1 + rand.Float64()*float64(snowMaxSize-1),
		speed:  0.5 + rand.Float64()*2.0,
		drift:  (rand.Float64() - 0.5) * 0.3,
		wobble: 0.3 + rand.Float64()*0.7,
		phase:  rand.Float64() * 6.28,
	}
}

func (s *snowState) update() {
	for i := range s.flakes {
		f := &s.flakes[i]
		f.y += f.speed
		f.phase += 0.05

		// Gentle sine-wave wobble for horizontal drift
		f.x += f.drift + f.wobble*sinApprox(f.phase)*0.3

		// Respawn at top if fallen below window
		if f.y > float64(s.height)+10 {
			*f = s.newFlake(false)
		}

		// Wrap horizontally
		if f.x < 0 {
			f.x += float64(s.width)
		} else if f.x >= float64(s.width) {
			f.x -= float64(s.width)
		}
	}
}

func (s *snowState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	for _, f := range s.flakes {
		s.drawFlake(pixels, f)
	}

	return pixels
}

func (s *snowState) drawFlake(pixels []uint8, f snowflake) {
	w, h := s.width, s.height
	cx, cy := int(f.x), int(f.y)
	radius := int(f.size)

	// Draw a soft circle with alpha falloff
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			px := cx + dx
			py := cy + dy

			if px < 0 || px >= w || py < 0 || py >= h {
				continue
			}

			// Distance from center (0.0 to 1.0+)
			dist := float64(dx*dx+dy*dy) / float64(radius*radius+1)
			if dist > 1.0 {
				continue
			}

			// Soft edge: alpha falls off near the border
			alpha := uint8((1.0 - dist*dist) * 220)

			offset := (py*w + px) * rgbaChannels
			// Blend: white snowflake with alpha
			existing := pixels[offset+3]
			if alpha > existing {
				pixels[offset] = 255   // R
				pixels[offset+1] = 255 // G
				pixels[offset+2] = 255 // B
				pixels[offset+3] = alpha
			}
		}
	}
}

