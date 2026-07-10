//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Doom-style procedural fire simulation using a heat propagation algorithm.
// --- Fire Effect ---

type fireState struct {
	grid   [][]uint8
	width  int
	height int
}

func (p *shenanigans) startFire(pc linkquisition.PickerCanvas) {
	state := &fireState{
		width:  fireWidth,
		height: fireHeight,
	}
	state.init()

	pc.AddRasterOverlay(0.4, func(w, h int) []uint8 {
		return state.render(w, h)
	})

	go func() {
		ticker := time.NewTicker(fireFrameInterval)
		defer ticker.Stop()

		for range ticker.C {
			if p.stopped.Load() {
				return
			}
			state.update()
			state.update() // double-step for faster movement
			pc.ScheduleRefresh()
		}
	}()
}

func (s *fireState) init() {
	s.grid = make([][]uint8, s.height)
	for i := range s.grid {
		s.grid[i] = make([]uint8, s.width)
	}
}

func (s *fireState) update() {
	// Set the bottom two rows to random hot values (wider fuel source)
	for x := range s.width {
		s.grid[s.height-1][x] = uint8(180 + rand.IntN(76))
		s.grid[s.height-2][x] = uint8(150 + rand.IntN(106))
	}

	// Propagate fire upward with averaging and cooling
	for y := range s.height - 2 {
		for x := range s.width {
			// Sample a wider neighborhood for smoother spread
			l2 := max(x-2, 0)
			l1 := max(x-1, 0)
			r1 := min(x+1, s.width-1)
			r2 := min(x+2, s.width-1)

			// Weighted average: center-heavy for more vertical flames
			sum := int(s.grid[y+1][l1]) +
				int(s.grid[y+1][x])*3 +
				int(s.grid[y+1][r1]) +
				int(s.grid[y+2][l2]) +
				int(s.grid[y+2][x])*2 +
				int(s.grid[y+2][r2])

			avg := sum / 9

			// Cooling increases toward the top for natural fadeout
			coolBase := 2 + (s.height-y)/15
			cooling := rand.IntN(coolBase + 1)
			val := avg - cooling

			if val < 0 {
				val = 0
			}
			s.grid[y][x] = uint8(val)
		}
	}
}

func (s *fireState) render(targetW, targetH int) []uint8 {
	if targetW == 0 || targetH == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, targetW*targetH*rgbaChannels)

	// Fire occupies the bottom 2/3 of the window
	fireStartY := targetH / 3

	fireH := targetH - fireStartY
	if fireH <= 0 {
		return pixels
	}

	for py := fireStartY; py < targetH; py++ {
		for px := 0; px < targetW; px++ {
			// Map to fire grid with bilinear interpolation
			fy := float64(py-fireStartY) * float64(s.height-1) / float64(fireH-1)
			fx := float64(px) * float64(s.width-1) / float64(targetW-1)

			val := s.sampleBilinear(fx, fy)

			r, g, b, a := fireColor(val)
			offset := (py*targetW + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = a
		}
	}

	return pixels
}

// sampleBilinear performs bilinear interpolation on the fire grid for smooth scaling.
func (s *fireState) sampleBilinear(fx, fy float64) uint8 {
	x0 := int(fx)
	y0 := int(fy)
	x1 := min(x0+1, s.width-1)
	y1 := min(y0+1, s.height-1)

	xFrac := fx - float64(x0)
	yFrac := fy - float64(y0)

	// Four corners
	v00 := float64(s.grid[y0][x0])
	v10 := float64(s.grid[y0][x1])
	v01 := float64(s.grid[y1][x0])
	v11 := float64(s.grid[y1][x1])

	// Interpolate
	top := v00*(1-xFrac) + v10*xFrac
	bottom := v01*(1-xFrac) + v11*xFrac
	val := top*(1-yFrac) + bottom*yFrac

	return uint8(min(max(int(val), 0), 255))
}

// fireColor maps a heat value (0-255) to a realistic fire palette.
// Gradient: transparent → dark red/brown → red → orange → gold → pale yellow
func fireColor(val uint8) (r, g, b, a uint8) {
	if val < 24 {
		return 0, 0, 0, 0
	}

	// Normalize to 0.0-1.0 range (24-255 → 0.0-1.0)
	t := float64(val-24) / 231.0

	// Piecewise palette for realistic fire
	switch {
	case t < 0.2:
		// Black → dark maroon/brown
		p := t / 0.2
		r = uint8(p * 80)
		g = uint8(p * 10)
		a = uint8(p * 180)
		return r, g, 0, a
	case t < 0.45:
		// Dark maroon → bright red
		p := (t - 0.2) / 0.25
		r = uint8(80 + p*175)
		g = uint8(10 + p*20)
		a = uint8(180 + p*75)
		return r, g, 0, a
	case t < 0.7:
		// Red → orange
		p := (t - 0.45) / 0.25
		r = 255
		g = uint8(30 + p*170)
		a = 255
		return r, g, 0, a
	case t < 0.9:
		// Orange → golden yellow
		p := (t - 0.7) / 0.2
		r = 255
		g = uint8(200 + p*55)
		b = uint8(p * 30)
		a = 255
		return r, g, b, a
	default:
		// Golden yellow → pale yellow/white tips
		p := (t - 0.9) / 0.1
		r = 255
		g = 255
		b = uint8(30 + p*120)
		a = uint8(255 - p*80)
		return r, g, b, a
	}
}
