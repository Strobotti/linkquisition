//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"time"

	"github.com/strobotti/linkquisition"
)

// Classic demoscene plasma effect using layered sine waves.
// --- Plasma Effect ---

type plasmaState struct {
	time float64
}

func (p *shenanigans) startPlasma(pc linkquisition.PickerCanvas) {
	state := &plasmaState{}

	pc.AddRasterOverlay(0.5, func(w, h int) []uint8 {
		return state.render(w, h)
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
		defer ticker.Stop()

		for range ticker.C {
			if p.stopped.Load() {
				return
			}
			state.time += 0.06
			pc.ScheduleRefresh()
		}
	}()
}

func (s *plasmaState) render(w, h int) []uint8 {
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	t := s.time

	for py := 0; py < h; py++ {
		fy := float64(py) / float64(h)
		for px := 0; px < w; px++ {
			fx := float64(px) / float64(w)

			// Overlapping sine waves at different frequencies and phases
			v1 := sinApprox((fx*4 + t) * 3.14159)
			v2 := sinApprox((fy*4 + t*0.7) * 3.14159)
			v3 := sinApprox(((fx+fy)*3 + t*1.3) * 3.14159)
			v4 := sinApprox(((fx-fy)*2 + t*0.5) * 3.14159)
			v5 := sinApprox(((fx*fx+fy*fy)*4 - t*0.9) * 3.14159)

			// Combine waves (result in -1 to 1 range, normalize to 0-1)
			val := (v1 + v2 + v3 + v4 + v5) / 5.0
			val = (val + 1.0) / 2.0

			// Map to color using three phase-shifted sine waves for RGB
			r := uint8(sinNorm(val*3.14159*2+t*0.3) * 255)
			g := uint8(sinNorm(val*3.14159*2+t*0.3+2.09) * 255)
			b := uint8(sinNorm(val*3.14159*2+t*0.3+4.19) * 255)

			offset := (py*w + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = 200
		}
	}

	return pixels
}

