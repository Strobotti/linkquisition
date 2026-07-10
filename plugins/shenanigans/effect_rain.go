//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Rainy window effect with falling droplets and splash particles.
// --- Rain Effect ---

const (
	rainDropCount   = 150
	rainAlpha       = 120
	rainSplashAlpha = 100
)

type raindrop struct {
	x      float64
	y      float64
	speed  float64
	length float64
	width  float64
}

type rainState struct {
	width, height int
	drops         []raindrop
	initialized   bool
}

func (p *shenanigans) startRain(pc linkquisition.PickerCanvas) {
	state := &rainState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}

	pc.AddRasterOverlay(0.5, func(w, h int) []uint8 {
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

func (s *rainState) init() {
	s.drops = make([]raindrop, rainDropCount)
	for i := range s.drops {
		s.drops[i] = s.newDrop(true)
	}
}

func (s *rainState) newDrop(onScreen bool) raindrop {
	y := -(rand.Float64() * float64(s.height))
	if onScreen {
		y = rand.Float64() * float64(s.height)
	}

	return raindrop{
		x:      rand.Float64() * float64(s.width),
		y:      y,
		speed:  8 + rand.Float64()*12,
		length: 15 + rand.Float64()*30,
		width:  1 + rand.Float64()*1.5,
	}
}

func (s *rainState) update() {
	for i := range s.drops {
		s.drops[i].y += s.drops[i].speed

		// Slight wind drift
		s.drops[i].x += 0.5

		// Respawn above when falling off bottom
		if s.drops[i].y > float64(s.height)+s.drops[i].length {
			s.drops[i] = s.newDrop(false)
		}
	}
}

func (s *rainState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	for i := range s.drops {
		s.drawDrop(pixels, s.drops[i])
	}

	return pixels
}

func (s *rainState) drawDrop(pixels []uint8, d raindrop) { //nolint:gocyclo
	// Draw a vertical streak
	startY := int(d.y - d.length)
	endY := int(d.y)
	x := int(d.x)
	dropWidth := int(d.width)

	for py := startY; py <= endY; py++ {
		if py < 0 || py >= s.height {
			continue
		}

		// Fade: brighter at the bottom (leading edge), dimmer at top
		progress := float64(py-startY) / d.length
		alpha := uint8(float64(rainAlpha) * progress)

		for dx := 0; dx < dropWidth; dx++ {
			px := x + dx
			if px < 0 || px >= s.width {
				continue
			}
			offset := (py*s.width + px) * rgbaChannels
			if alpha > pixels[offset+3] {
				pixels[offset] = 180   // R — cool blue-white
				pixels[offset+1] = 200 // G
				pixels[offset+2] = 255 // B
				pixels[offset+3] = alpha
			}
		}
	}

	// Splash at the bottom when the drop hits — wider and a few pixels tall
	if endY >= s.height-3 && endY < s.height {
		splashWidth := int(d.width) + 4
		for dy := 0; dy < 3; dy++ {
			py := s.height - 1 - dy
			splAlpha := rainSplashAlpha - uint8(dy*30)
			for dx := -splashWidth; dx <= splashWidth; dx++ {
				px := x + dx
				if px >= 0 && px < s.width && py >= 0 {
					offset := (py*s.width + px) * rgbaChannels
					if splAlpha > pixels[offset+3] {
						pixels[offset] = 200
						pixels[offset+1] = 220
						pixels[offset+2] = 255
						pixels[offset+3] = splAlpha
					}
				}
			}
		}
	}
}

