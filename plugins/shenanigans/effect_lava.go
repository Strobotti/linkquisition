//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Lava lamp with soft blobs rising and falling in viscous fluid.
// --- Lava Lamp Effect ---

const (
	lavaBlobCount     = 6
	lavaAlpha         = 60
	lavaFrameInterval = 40 * time.Millisecond
	lavaThreshold     = 1.0
	lavaResDiv        = 4 // render at 1/4 resolution for performance
)

type lavaBlob struct {
	x, y       float64
	radius     float64
	vy         float64
	phase      float64
	phaseSpeed float64
}

type lavaState struct {
	width, height int
	blobs         []lavaBlob
	time          float64
	hueShift      float64
}

func (p *shenanigans) startLava(pc linkquisition.PickerCanvas) {
	p.startEffect(pc, effectConfig{
		state:         &lavaState{},
		opacity:       0.6,
		frameInterval: lavaFrameInterval,
		skipInvert:    true,
	})
}

func (s *lavaState) init(width, height int) {
	s.width = width
	s.height = height
	s.blobs = make([]lavaBlob, lavaBlobCount)
	for i := range s.blobs {
		s.blobs[i] = lavaBlob{
			x:          0.2 + rand.Float64()*0.6,
			y:          rand.Float64(),
			radius:     0.06 + rand.Float64()*0.08,
			vy:         0.002 + rand.Float64()*0.003,
			phase:      rand.Float64() * 6.28,
			phaseSpeed: 0.01 + rand.Float64()*0.02,
		}
		if i%2 == 0 {
			s.blobs[i].vy = -s.blobs[i].vy
		}
	}
}

func (s *lavaState) update() {
	s.time += 0.02
	s.hueShift += 0.003

	for i := range s.blobs {
		b := &s.blobs[i]
		b.y += b.vy
		b.phase += b.phaseSpeed
		b.x += sinApprox(b.phase) * 0.002

		if b.y < -0.1 {
			b.vy = absF(b.vy)
		} else if b.y > 1.1 {
			b.vy = -absF(b.vy)
		}
		if b.x < 0.1 {
			b.x = 0.1
		} else if b.x > 0.9 {
			b.x = 0.9
		}
	}
}

func (s *lavaState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	step := lavaResDiv

	for py := 0; py < h; py += step {
		fy := float64(py) / float64(h)
		for px := 0; px < w; px += step {
			fx := float64(px) / float64(w)

			field := 0.0
			for _, b := range s.blobs {
				dx := fx - b.x
				dy := fy - b.y
				distSq := dx*dx + dy*dy
				if distSq < 0.0001 {
					distSq = 0.0001
				}
				field += (b.radius * b.radius) / distSq
			}

			if field < lavaThreshold {
				continue
			}

			intensity := (field - lavaThreshold) / lavaThreshold
			if intensity > 1.0 {
				intensity = 1.0
			}

			r, g, b := s.lavaColor(intensity, fy)
			a := uint8(float64(lavaAlpha) * (0.5 + intensity*0.5))

			for dy := 0; dy < step && py+dy < h; dy++ {
				for dx := 0; dx < step && px+dx < w; dx++ {
					offset := ((py+dy)*w + px + dx) * rgbaChannels
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

func (s *lavaState) lavaColor(intensity, fy float64) (rOut, gOut, bOut uint8) {
	hue := s.hueShift + fy*0.3
	h := hue - float64(int(hue))
	var r, g, b float64

	switch {
	case h < 0.25:
		t := h / 0.25
		r, g, b = 1.0, 0.2+t*0.4, 0.0
	case h < 0.5:
		t := (h - 0.25) / 0.25
		r, g, b = 1.0, 0.6-t*0.5, t*0.6
	case h < 0.75:
		t := (h - 0.5) / 0.25
		r, g, b = 1.0-t*0.3, 0.1, 0.6+t*0.3
	default:
		t := (h - 0.75) / 0.25
		r, g, b = 0.7+t*0.3, 0.1+t*0.1, 0.9-t*0.9
	}

	r = r*0.6 + r*0.4*intensity
	g = g*0.4 + g*0.6*intensity
	b = b*0.5 + b*0.5*intensity

	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}
