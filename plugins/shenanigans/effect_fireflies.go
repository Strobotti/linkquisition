//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Glowing fireflies drifting lazily with pulsing bioluminescence.
// --- Fireflies Effect ---

const (
	fireflyCount         = 40
	fireflyAlpha         = 70
	fireflyFrameInterval = 30 * time.Millisecond
)

type firefly struct {
	x, y       float64
	vx, vy     float64
	phase      float64
	phaseSpeed float64
	glowSize   float64
	hue        float64
}

type firefliesState struct {
	width, height int
	flies         []firefly
}

func (p *shenanigans) startFireflies(pc linkquisition.PickerCanvas) {
	state := &firefliesState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}
	state.init()

	pc.AddRasterOverlay(0.6, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.init()
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(fireflyFrameInterval)
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

func (s *firefliesState) init() {
	s.flies = make([]firefly, fireflyCount)
	for i := range s.flies {
		s.flies[i] = firefly{
			x:          rand.Float64() * float64(s.width),
			y:          rand.Float64() * float64(s.height),
			vx:         (rand.Float64() - 0.5) * 0.8,
			vy:         (rand.Float64() - 0.5) * 0.8,
			phase:      rand.Float64() * 6.28,
			phaseSpeed: 0.03 + rand.Float64()*0.05,
			glowSize:   6 + rand.Float64()*8,
			hue:        0.1 + rand.Float64()*0.15, // warm yellow-green range
		}
	}
}

func (s *firefliesState) update() {
	w := float64(s.width)
	h := float64(s.height)

	for i := range s.flies {
		f := &s.flies[i]

		// Advance blink phase
		f.phase += f.phaseSpeed

		// Gentle random drift
		f.vx += (rand.Float64() - 0.5) * 0.1
		f.vy += (rand.Float64() - 0.5) * 0.1

		// Dampen velocity
		f.vx *= 0.98
		f.vy *= 0.98

		// Clamp speed
		maxSpeed := 1.5
		if f.vx > maxSpeed {
			f.vx = maxSpeed
		} else if f.vx < -maxSpeed {
			f.vx = -maxSpeed
		}
		if f.vy > maxSpeed {
			f.vy = maxSpeed
		} else if f.vy < -maxSpeed {
			f.vy = -maxSpeed
		}

		f.x += f.vx
		f.y += f.vy

		// Soft bounce off edges
		margin := 20.0
		if f.x < margin {
			f.vx += 0.1
		} else if f.x > w-margin {
			f.vx -= 0.1
		}
		if f.y < margin {
			f.vy += 0.1
		} else if f.y > h-margin {
			f.vy -= 0.1
		}
	}
}

func (s *firefliesState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	for _, f := range s.flies {
		// Brightness oscillates with phase — soft blink
		brightness := sinApprox(f.phase)
		if brightness < 0 {
			brightness = 0 // off half the time
		}
		brightness = brightness * brightness // sharper on/off

		if brightness < 0.05 {
			continue
		}

		// Glow color (warm yellow-green)
		r, g, b := fireflyColor(f.hue, brightness)
		alpha := uint8(float64(fireflyAlpha) * brightness)

		// Draw soft glow
		cx, cy := int(f.x), int(f.y)
		radius := int(f.glowSize * (0.5 + brightness*0.5))

		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				dist := float64(dx*dx+dy*dy) / float64(radius*radius+1)
				if dist > 1.0 {
					continue
				}
				// Soft falloff
				falloff := (1.0 - dist) * (1.0 - dist)
				px := cx + dx
				py := cy + dy
				if px >= 0 && px < w && py >= 0 && py < h {
					a := uint8(float64(alpha) * falloff)
					offset := (py*w + px) * rgbaChannels
					if a > pixels[offset+3] {
						pixels[offset] = r
						pixels[offset+1] = g
						pixels[offset+2] = b
						pixels[offset+3] = a
					}
				}
			}
		}
	}

	return pixels
}

func fireflyColor(hue, brightness float64) (uint8, uint8, uint8) {
	// Warm glow: yellow core fading to green at edges
	r := uint8((0.9 + hue*0.5) * brightness * 255)
	g := uint8((0.8 + hue) * brightness * 255)
	b := uint8(0.1 * brightness * 255)
	return r, g, b
}

