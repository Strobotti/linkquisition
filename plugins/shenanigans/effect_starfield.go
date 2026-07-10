//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"

	"github.com/strobotti/linkquisition"
)

// 3D starfield flying through space with parallax depth layers.
// --- Starfield Effect ---

const starCount = 200

type star struct {
	x, y, z float64
}

type starfieldState struct {
	stars  []star
	width  int
	height int
}

func (p *shenanigans) startStarfield(pc linkquisition.PickerCanvas) {
	p.startEffect(pc, effectConfig{
		state:         &starfieldState{},
		opacity:       0.2,
		frameInterval: frameInterval,
	})
}

func (s *starfieldState) init(width, height int) {
	s.width = width
	s.height = height
	s.stars = make([]star, starCount)
	for i := range s.stars {
		s.stars[i] = s.newStar(true)
	}
}

func (s *starfieldState) newStar(randomDepth bool) star {
	z := 0.01 + rand.Float64()*0.99
	if randomDepth {
		z = 0.1 + rand.Float64()*0.9
	}

	return star{
		x: (rand.Float64() - 0.5) * 2.0,
		y: (rand.Float64() - 0.5) * 2.0,
		z: z,
	}
}

func (s *starfieldState) update() {
	for i := range s.stars {
		s.stars[i].z -= 0.015

		// Respawn stars that have passed the viewer
		if s.stars[i].z <= 0.001 {
			s.stars[i] = s.newStar(false)
			s.stars[i].z = 0.9 + rand.Float64()*0.1
		}
	}
}

func (s *starfieldState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	cx := float64(w) / 2.0
	cy := float64(h) / 2.0

	for _, st := range s.stars {
		// Perspective projection
		screenX := int(cx + (st.x/st.z)*cx)
		screenY := int(cy + (st.y/st.z)*cy)

		if screenX < 0 || screenX >= w || screenY < 0 || screenY >= h {
			continue
		}

		// Size and brightness increase as stars get closer (z → 0)
		brightness := uint8(min(int((1.0-st.z)*255), 255))
		size := int(1 + (1.0-st.z)*3)

		s.drawStar(pixels, screenX, screenY, size, brightness)
	}

	return pixels
}

func (s *starfieldState) drawStar(pixels []uint8, screenX, screenY, size int, brightness uint8) {
	w, h := s.width, s.height
	for dy := -size; dy <= size; dy++ {
		for dx := -size; dx <= size; dx++ {
			px := screenX + dx
			py := screenY + dy

			if px < 0 || px >= w || py < 0 || py >= h {
				continue
			}

			dist := dx*dx + dy*dy
			maxDist := size * size
			if dist > maxDist {
				continue
			}

			// Alpha falls off with distance from center
			falloff := 1.0 - float64(dist)/float64(maxDist+1)
			alpha := uint8(float64(brightness) * falloff)

			offset := (py*w + px) * rgbaChannels
			if alpha > pixels[offset+3] {
				pixels[offset] = brightness
				pixels[offset+1] = brightness
				pixels[offset+2] = 255 // slight blue tint
				pixels[offset+3] = alpha
			}
		}
	}
}
