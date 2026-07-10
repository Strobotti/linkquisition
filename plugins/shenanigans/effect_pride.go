//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"time"

	"github.com/strobotti/linkquisition"
)

// Animated rainbow pride flag with gentle wave motion.
// --- Pride Effect ---

type prideState struct {
	time float64
}

// Pride flag colors (6-stripe rainbow)
var prideColors = [][3]uint8{
	{228, 3, 3},   // Red
	{255, 140, 0}, // Orange
	{255, 237, 0}, // Yellow
	{0, 128, 38},  // Green
	{0, 77, 255},  // Blue
	{117, 7, 135}, // Purple
}

func (p *shenanigans) startPride(pc linkquisition.PickerCanvas) {
	state := &prideState{}

	pc.AddRasterOverlay(0.45, func(w, h int) []uint8 {
		return state.render(w, h)
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
		defer ticker.Stop()

		for range ticker.C {
			if p.stopped.Load() {
				return
			}
			state.time += 0.04
			pc.ScheduleRefresh()
		}
	}()
}

func (s *prideState) render(w, h int) []uint8 {
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	t := s.time
	stripeCount := float64(len(prideColors))

	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			fx := float64(px) / float64(w)
			fy := float64(py) / float64(h)

			// Flag wave: pinned on the left edge, wave amplitude increases to the right
			// This simulates fabric attached to a pole on the left side
			amplitude := fx * fx * 0.12
			wave := sinApprox((fx*2.0-t*1.5)*3.14159) * amplitude
			wave2 := sinApprox((fx*3.0-t*2.0)*3.14159) * amplitude * 0.5

			// Determine which stripe this pixel belongs to (with wave offset)
			stripePos := (fy + wave + wave2) * stripeCount
			stripeIdx := int(stripePos)

			if stripeIdx < 0 {
				stripeIdx = 0
			}
			if stripeIdx >= len(prideColors) {
				stripeIdx = len(prideColors) - 1
			}

			// Smooth blending between stripes
			blend := stripePos - float64(stripeIdx)
			nextIdx := stripeIdx + 1
			if nextIdx >= len(prideColors) {
				nextIdx = len(prideColors) - 1
			}

			c1 := prideColors[stripeIdx]
			c2 := prideColors[nextIdx]

			// Smooth interpolation (smoothstep-like)
			blend = blend * blend * (3 - 2*blend)

			r := uint8(float64(c1[0])*(1-blend) + float64(c2[0])*blend)
			g := uint8(float64(c1[1])*(1-blend) + float64(c2[1])*blend)
			b := uint8(float64(c1[2])*(1-blend) + float64(c2[2])*blend)

			// Subtle shading to simulate fabric folds (stronger toward free edge)
			foldDepth := fx * 0.2
			shade := 1.0 - foldDepth + foldDepth*sinApprox((fx*3-t*1.5)*3.14159)
			r = uint8(float64(r) * shade)
			g = uint8(float64(g) * shade)
			b = uint8(float64(b) * shade)

			offset := (py*w + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = 200
		}
	}

	return pixels
}

