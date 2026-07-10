//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"

	"github.com/strobotti/linkquisition"
)

// Celebratory fireworks bursting with colorful particle trails.
// --- Fireworks Effect ---

const (
	fireworksMaxRockets   = 5
	fireworksParticles    = 60
	fireworksLaunchChance = 8 // percent chance per frame to launch a new rocket
)

type fireworksParticle struct {
	x, y    float64
	vx, vy  float64
	life    float64
	r, g, b uint8
}

type fireworksRocket struct {
	particles []fireworksParticle
	exploded  bool
	// Pre-explosion rocket position
	x, y    float64
	vy      float64
	targetY float64
	color   [3]uint8
}

type fireworksState struct {
	rockets []fireworksRocket
	width   int
	height  int
}

var fireworksColors = [][3]uint8{
	{255, 200, 50},  // Gold
	{255, 80, 80},   // Red
	{80, 150, 255},  // Blue
	{80, 255, 80},   // Green
	{200, 100, 255}, // Purple
	{255, 150, 200}, // Pink
	{255, 255, 255}, // White
}

func (p *shenanigans) startFireworks(pc linkquisition.PickerCanvas) {
	p.startEffect(pc, effectConfig{
		state:         &fireworksState{},
		opacity:       0.2,
		frameInterval: frameInterval,
		skipInvert:    true,
	})
}

func (s *fireworksState) init(width, height int) {
	s.width = width
	s.height = height
	s.rockets = nil
}

func (s *fireworksState) update() {
	if s.width == 0 || s.height == 0 {
		return
	}

	// Chance to launch a new rocket
	if len(s.rockets) < fireworksMaxRockets && rand.IntN(100) < fireworksLaunchChance {
		color := fireworksColors[rand.IntN(len(fireworksColors))]
		s.rockets = append(s.rockets, fireworksRocket{
			x:       0.2 + rand.Float64()*0.6,
			y:       1.0,
			vy:      -0.025 - rand.Float64()*0.015,
			targetY: 0.15 + rand.Float64()*0.35,
			color:   color,
		})
	}

	// Update rockets
	alive := s.rockets[:0]
	for i := range s.rockets {
		r := &s.rockets[i]

		if !r.exploded {
			r.y += r.vy
			// Explode when reaching target height
			if r.y <= r.targetY {
				r.exploded = true
				r.particles = make([]fireworksParticle, fireworksParticles)
				for j := range r.particles {
					angle := rand.Float64() * 6.283
					speed := 0.005 + rand.Float64()*0.015
					r.particles[j] = fireworksParticle{
						x:    r.x,
						y:    r.y,
						vx:   sinApprox(angle) * speed,
						vy:   sinApprox(angle+1.5708) * speed,
						life: 1.0,
						r:    r.color[0],
						g:    r.color[1],
						b:    r.color[2],
					}
				}
			}
		} else {
			// Update particles
			allDead := true
			for j := range r.particles {
				p := &r.particles[j]
				if p.life <= 0 {
					continue
				}
				p.x += p.vx
				p.y += p.vy
				p.vy += 0.0004 // gravity
				p.vx *= 0.98   // drag
				p.vy *= 0.98
				p.life -= 0.015
				if p.life > 0 {
					allDead = false
				}
			}
			if allDead {
				continue // don't keep this rocket
			}
		}
		alive = append(alive, *r)
	}
	s.rockets = alive
}

func (s *fireworksState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	for _, rocket := range s.rockets {
		if !rocket.exploded {
			// Draw rising rocket as a small bright dot with trail
			px := int(rocket.x * float64(w))
			py := int(rocket.y * float64(h))
			s.drawDot(pixels, px, py, 2, 255, 220, 150, 255)
			// Trail
			s.drawDot(pixels, px, py+3, 1, 255, 150, 50, 150)
			s.drawDot(pixels, px, py+6, 1, 255, 100, 30, 80)
		} else {
			// Draw particles
			for _, p := range rocket.particles {
				if p.life <= 0 {
					continue
				}
				px := int(p.x * float64(w))
				py := int(p.y * float64(h))
				alpha := uint8(p.life * 255)
				size := 1
				if p.life > 0.7 {
					size = 2
				}
				s.drawDot(pixels, px, py, size, p.r, p.g, p.b, alpha)
			}
		}
	}

	return pixels
}

func (s *fireworksState) drawDot(pixels []uint8, cx, cy, radius int, r, g, b, a uint8) {
	w, h := s.width, s.height
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			px := cx + dx
			py := cy + dy
			if px < 0 || px >= w || py < 0 || py >= h {
				continue
			}
			if dx*dx+dy*dy > radius*radius {
				continue
			}
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
