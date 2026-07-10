//nolint:mnd // Visual effects plugin: magic numbers are by design.
package main

import (
	"github.com/strobotti/linkquisition"
)

// Football (soccer) pitch with a ball bouncing around and a spotlight.
// --- Football Effect ---

type footballState struct {
	time float64
}

func (p *shenanigans) startFootball(pc linkquisition.PickerCanvas) {
	p.startSimpleEffect(pc, simpleEffectConfig{
		state:         &footballState{},
		opacity:       0.4,
		frameInterval: frameInterval,
	})
}

func (s *footballState) update() {
	s.time += 0.03
}

func (s *footballState) render(w, h int) []uint8 {
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	t := s.time

	for py := 0; py < h; py++ {
		fy := float64(py) / float64(h)
		for px := 0; px < w; px++ {
			fx := float64(px) / float64(w)

			// Green pitch base with subtle stripe pattern (mowed grass look)
			stripeWidth := 0.08
			stripe := int(fx/stripeWidth) % 2
			var gr, gg, gb uint8
			if stripe == 0 {
				gr, gg, gb = 34, 139, 34
			} else {
				gr, gg, gb = 30, 124, 30
			}

			// Animated element: a "spotlight" sweeping across the pitch
			spotX := 0.5 + 0.4*sinApprox(t*1.5)
			spotY := 0.5 + 0.3*sinApprox(t*1.1+1.0)
			spotDist := (fx-spotX)*(fx-spotX) + (fy-spotY)*(fy-spotY)
			spotLight := fastExp(-spotDist*15) * 0.3

			var r, g, b uint8
			if isPitchLine(fx, fy) {
				r, g, b = 255, 255, 255
			} else {
				r = uint8(min(int(float64(gr)*(1+spotLight)), 255))
				g = uint8(min(int(float64(gg)*(1+spotLight)), 255))
				b = uint8(min(int(float64(gb)*(1+spotLight)), 255))
			}

			offset := (py*w + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = 180
		}
	}

	return pixels
}

// isPitchLine determines if a normalized coordinate is on a white pitch marking.
func isPitchLine(fx, fy float64) bool {
	return isPitchCenterMarkings(fx, fy) ||
		isPitchBoundary(fx, fy) ||
		isPitchPenaltyArea(fx, fy)
}

func isPitchCenterMarkings(fx, fy float64) bool {
	// Center line (vertical)
	if absF(fx-0.5) < 0.004 {
		return true
	}

	// Center circle
	cx, cy := 0.5, 0.5
	dist := (fx-cx)*(fx-cx)*1.5 + (fy-cy)*(fy-cy)
	if absF(dist-0.04) < 0.003 {
		return true
	}

	// Center dot
	return dist < 0.002
}

func isPitchBoundary(fx, fy float64) bool {
	if fx < 0.02 || fx > 0.98 || fy < 0.03 || fy > 0.97 {
		if fx > 0.015 && fx < 0.985 && fy > 0.025 && fy < 0.975 {
			return true
		}
	}
	return false
}

func isPitchPenaltyArea(fx, fy float64) bool {
	penaltyW := 0.15
	penaltyH := 0.35
	penaltyTop := 0.5 - penaltyH
	penaltyBot := 0.5 + penaltyH

	// Left penalty area
	if fx < penaltyW && fy > penaltyTop && fy < penaltyBot {
		if absF(fx-penaltyW) < 0.004 || absF(fy-penaltyTop) < 0.005 || absF(fy-penaltyBot) < 0.005 {
			return true
		}
	}

	// Right penalty area
	if fx > (1-penaltyW) && fy > penaltyTop && fy < penaltyBot {
		if absF(fx-(1-penaltyW)) < 0.004 || absF(fy-penaltyTop) < 0.005 || absF(fy-penaltyBot) < 0.005 {
			return true
		}
	}

	return false
}
