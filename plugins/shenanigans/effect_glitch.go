//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// VHS-style digital glitch with scan lines, color shifts, and block artifacts.
// --- Glitch Effect ---

type glitchState struct {
	frame      int
	slices     []glitchSlice
	burstTimer int
	isBursting bool
}

type glitchSlice struct {
	y, height int
	offsetX   int
	channel   int // 0=R shift, 1=G shift, 2=B shift
	alpha     uint8
}

func (p *shenanigans) startGlitch(pc linkquisition.PickerCanvas) {
	state := &glitchState{}

	pc.AddRasterOverlay(0.0, func(w, h int) []uint8 {
		return state.render(w, h)
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

func (s *glitchState) update() {
	s.frame++
	s.burstTimer--

	// Trigger glitch bursts periodically
	if s.burstTimer <= 0 {
		if s.isBursting {
			// End burst
			s.isBursting = false
			s.slices = nil
			s.burstTimer = 20 + rand.IntN(40)
		} else {
			// Start burst
			s.isBursting = true
			s.burstTimer = 3 + rand.IntN(8)
			s.generateSlices()
		}
	} else if s.isBursting && s.frame%2 == 0 {
		// Regenerate slices during burst for flicker
		s.generateSlices()
	}
}

func (s *glitchState) generateSlices() {
	count := 3 + rand.IntN(8)
	s.slices = make([]glitchSlice, count)

	for i := range s.slices {
		s.slices[i] = glitchSlice{
			y:       rand.IntN(400),
			height:  2 + rand.IntN(20),
			offsetX: -30 + rand.IntN(60),
			channel: rand.IntN(3),
			alpha:   uint8(80 + rand.IntN(176)),
		}
	}
}

func (s *glitchState) render(w, h int) []uint8 {
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	if !s.isBursting {
		return pixels
	}

	// Draw glitch slices
	for _, slice := range s.slices {
		s.drawSlice(pixels, slice, w, h)
	}

	// Add random static noise during bursts
	s.addNoise(pixels, w, h)

	return pixels
}

func (s *glitchState) drawSlice(pixels []uint8, slice glitchSlice, w, h int) {
	sy := slice.y * h / 400
	sh := slice.height * h / 400

	for py := sy; py < sy+sh && py < h; py++ {
		if py < 0 {
			continue
		}
		for px := 0; px < w; px++ {
			offset := (py*w + px) * rgbaChannels

			// RGB channel separation effect
			switch slice.channel {
			case 0: // Red shift
				srcX := px - slice.offsetX
				if srcX >= 0 && srcX < w {
					pixels[offset] = slice.alpha
					pixels[offset+3] = slice.alpha / 2
				}
			case 1: // Green shift
				srcX := px + slice.offsetX
				if srcX >= 0 && srcX < w {
					pixels[offset+1] = slice.alpha
					pixels[offset+3] = slice.alpha / 2
				}
			default: // Blue/cyan shift
				pixels[offset+2] = slice.alpha
				pixels[offset+1] = slice.alpha / 3
				pixels[offset+3] = slice.alpha / 2
			}
		}
	}
}

func (s *glitchState) addNoise(pixels []uint8, w, h int) {
	noiseCount := w * h / 40
	for range noiseCount {
		px := rand.IntN(w)
		py := rand.IntN(h)
		offset := (py*w + px) * rgbaChannels
		v := uint8(rand.IntN(256))
		pixels[offset] = v
		pixels[offset+1] = v
		pixels[offset+2] = v
		pixels[offset+3] = uint8(rand.IntN(100))
	}
}

