//nolint:mnd // Visual effects plugin: magic numbers are by design.
package main

import (
	"github.com/strobotti/linkquisition"
)

// Northern lights (aurora borealis) with shimmering curtain waves.
// --- Aurora Effect ---

const auroraLayers = 4

type auroraState struct {
	time float64
}

func (p *shenanigans) startAurora(pc linkquisition.PickerCanvas) {
	p.startSimpleEffect(pc, simpleEffectConfig{
		state:         &auroraState{},
		opacity:       0.4,
		frameInterval: frameInterval,
	})
}

func (s *auroraState) update() {
	s.time += 0.03
}

func (s *auroraState) render(w, h int) []uint8 {
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	t := s.time

	// Aurora occupies the top 2/3 of the window
	auroraEndY := h * 2 / 3

	for py := 0; py < auroraEndY; py++ {
		// Vertical position normalized (0 at top, 1 at aurora bottom)
		fy := float64(py) / float64(auroraEndY)

		for px := 0; px < w; px++ {
			fx := float64(px) / float64(w)

			// Combine multiple curtain layers
			var intensity float64
			for layer := range auroraLayers {
				fl := float64(layer)
				// Each layer has different frequency, speed, and phase
				wave := sinApprox((fx*(3+fl) + t*(0.4+fl*0.15) + fl*1.7) * 3.14159)
				wave2 := sinApprox((fx*(2+fl*0.7) - t*(0.3+fl*0.1) + fl*2.3) * 3.14159)

				// Curtain shape: thin band that undulates
				curtainCenter := 0.2 + 0.15*fl + 0.1*(wave*0.5+0.5)
				curtainWidth := 0.08 + 0.04*wave2

				// Gaussian-like falloff from the curtain center
				dist := (fy - curtainCenter) / curtainWidth
				layerIntensity := fastExp(-dist * dist * 0.5)

				intensity += layerIntensity * (0.6 + 0.4/(fl+1))
			}

			if intensity < 0.01 {
				continue
			}
			if intensity > 1.0 {
				intensity = 1.0
			}

			// Aurora color: shift from green to purple/blue based on position and time
			colorPhase := fx*0.5 + fy*0.3 + t*0.1
			r, g, b := auroraColor(colorPhase, intensity)

			alpha := uint8(intensity * 180)

			offset := (py*w + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = alpha
		}
	}

	return pixels
}

// auroraColor maps a phase and intensity to northern lights colors.
// Shifts between green, teal, blue, and purple.
func auroraColor(phase, intensity float64) (r, g, b uint8) {
	// Cycle through aurora palette
	p := sinApprox(phase * 3.14159 * 2)
	p = (p + 1.0) / 2.0 // normalize to 0-1

	// Blend between green-dominant and purple-dominant
	var rf, gf, bf float64
	switch {
	case p < 0.33:
		// Green to teal
		t := p / 0.33
		rf = 0.1 * t
		gf = 0.8 + 0.2*t
		bf = 0.2 + 0.5*t
	case p < 0.66:
		// Teal to purple
		t := (p - 0.33) / 0.33
		rf = 0.1 + 0.5*t
		gf = 1.0 - 0.6*t
		bf = 0.7 + 0.3*t
	default:
		// Purple back to green
		t := (p - 0.66) / 0.34
		rf = 0.6 - 0.5*t
		gf = 0.4 + 0.4*t
		bf = 1.0 - 0.8*t
	}

	r = uint8(rf * intensity * 255)
	g = uint8(gf * intensity * 255)
	b = uint8(bf * intensity * 255)
	return r, g, b
}

// fastExp approximates e^x for negative x values (used for Gaussian falloff).
func fastExp(x float64) float64 {
	if x < -6 {
		return 0
	}
	// Padé approximation: (1 + x/n)^n for small |x|
	// Using n=8 for reasonable accuracy
	t := 1.0 + x/8.0
	t *= t // ^2
	t *= t // ^4
	t *= t // ^8
	if t < 0 {
		return 0
	}
	return t
}
