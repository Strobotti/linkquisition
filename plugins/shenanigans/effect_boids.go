//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Boids flocking simulation — birds following separation, alignment, and cohesion rules.
// --- Boids Effect ---

const (
	boidCount         = 60
	boidAlpha         = 60
	boidFrameInterval = 25 * time.Millisecond
	boidMaxSpeed      = 3.5
	boidVisualRange   = 60.0
	boidSeparationDist = 20.0
)

type boid struct {
	x, y   float64
	vx, vy float64
	hue    float64
}

type boidsState struct {
	width, height int
	flock         []boid
}

func (p *shenanigans) startBoids(pc linkquisition.PickerCanvas) {
	state := &boidsState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}
	state.init()

	pc.AddRasterOverlay(0.6, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.init()
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(boidFrameInterval)
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

func (s *boidsState) init() {
	s.flock = make([]boid, boidCount)
	for i := range s.flock {
		s.flock[i] = boid{
			x:   rand.Float64() * float64(s.width),
			y:   rand.Float64() * float64(s.height),
			vx:  (rand.Float64() - 0.5) * boidMaxSpeed * 2,
			vy:  (rand.Float64() - 0.5) * boidMaxSpeed * 2,
			hue: 0.55 + rand.Float64()*0.15, // blue-teal range
		}
	}
}

func (s *boidsState) update() {
	w := float64(s.width)
	h := float64(s.height)

	for i := range s.flock {
		b := &s.flock[i]

		// Flocking forces
		var sepX, sepY float64 // separation
		var alignVX, alignVY float64 // alignment
		var cohX, cohY float64 // cohesion
		neighbors := 0

		for j := range s.flock {
			if i == j {
				continue
			}
			other := &s.flock[j]
			dx := other.x - b.x
			dy := other.y - b.y
			dist := dx*dx + dy*dy

			if dist < boidVisualRange*boidVisualRange {
				neighbors++
				alignVX += other.vx
				alignVY += other.vy
				cohX += other.x
				cohY += other.y

				// Separation: push away from very close boids
				if dist < boidSeparationDist*boidSeparationDist && dist > 0.1 {
					sepX -= dx
					sepY -= dy
				}
			}
		}

		if neighbors > 0 {
			nf := float64(neighbors)

			// Alignment: steer toward average velocity
			alignVX /= nf
			alignVY /= nf
			b.vx += (alignVX - b.vx) * 0.05

			b.vy += (alignVY - b.vy) * 0.05

			// Cohesion: steer toward average position
			cohX /= nf
			cohY /= nf
			b.vx += (cohX - b.x) * 0.003
			b.vy += (cohY - b.y) * 0.003
		}

		// Separation
		b.vx += sepX * 0.05
		b.vy += sepY * 0.05

		// Edge avoidance (soft turn)
		margin := 50.0
		turnForce := 0.3
		if b.x < margin {
			b.vx += turnForce
		} else if b.x > w-margin {
			b.vx -= turnForce
		}
		if b.y < margin {
			b.vy += turnForce
		} else if b.y > h-margin {
			b.vy -= turnForce
		}

		// Speed limit
		speed := b.vx*b.vx + b.vy*b.vy
		if speed > boidMaxSpeed*boidMaxSpeed {
			scale := boidMaxSpeed / (absF(b.vx) + absF(b.vy))
			b.vx *= scale * 1.4
			b.vy *= scale * 1.4
		}

		// Move
		b.x += b.vx
		b.y += b.vy
	}
}

func (s *boidsState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	for _, b := range s.flock {
		cx, cy := int(b.x), int(b.y)
		if cx < 0 || cx >= w || cy < 0 || cy >= h {
			continue
		}

		// Draw as a small triangle pointing in movement direction
		s.drawBoid(pixels, b)
	}

	return pixels
}

func (s *boidsState) drawBoid(pixels []uint8, b boid) {
	w, h := s.width, s.height

	// Normalize velocity for direction
	speed := absF(b.vx) + absF(b.vy)
	if speed < 0.1 {
		speed = 0.1
	}
	dirX := b.vx / speed
	dirY := b.vy / speed

	// Triangle points: nose in direction of travel, tail behind
	size := 5.0
	noseX := b.x + dirX*size
	noseY := b.y + dirY*size
	// Perpendicular for tail width
	perpX := -dirY * size * 0.5
	perpY := dirX * size * 0.5
	tailX1 := b.x - dirX*size*0.3 + perpX
	tailY1 := b.y - dirY*size*0.3 + perpY
	tailX2 := b.x - dirX*size*0.3 - perpX
	tailY2 := b.y - dirY*size*0.3 - perpY

	// Color from hue
	r, g, bl := boidColor(b.hue)

	// Draw filled triangle using scanline
	s.drawBoidTriangle(pixels, w, h,
		int(noseX), int(noseY),
		int(tailX1), int(tailY1),
		int(tailX2), int(tailY2),
		r, g, bl)
}

func (s *boidsState) drawBoidTriangle(pixels []uint8, w, h, x0, y0, x1, y1, x2, y2 int, r, g, b uint8) {
	// Simple: draw lines between vertices
	s.drawBoidLine(pixels, w, h, x0, y0, x1, y1, r, g, b)
	s.drawBoidLine(pixels, w, h, x1, y1, x2, y2, r, g, b)
	s.drawBoidLine(pixels, w, h, x2, y2, x0, y0, r, g, b)
	// Fill center
	cx := (x0 + x1 + x2) / 3
	cy := (y0 + y1 + y2) / 3
	if cx >= 0 && cx < w && cy >= 0 && cy < h {
		offset := (cy*w + cx) * rgbaChannels
		if boidAlpha > pixels[offset+3] {
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = boidAlpha
		}
	}
}

func (s *boidsState) drawBoidLine(pixels []uint8, w, h, x0, y0, x1, y1 int, r, g, b uint8) {
	steps := abs(x1-x0) + abs(y1-y0)
	if steps == 0 {
		return
	}
	if steps > 30 {
		steps = 30
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		px := x0 + int(float64(x1-x0)*t)
		py := y0 + int(float64(y1-y0)*t)
		if px >= 0 && px < w && py >= 0 && py < h {
			offset := (py*w + px) * rgbaChannels
			if boidAlpha > pixels[offset+3] {
				pixels[offset] = r
				pixels[offset+1] = g
				pixels[offset+2] = b
				pixels[offset+3] = boidAlpha
			}
		}
	}
}

func boidColor(hue float64) (uint8, uint8, uint8) {
	// Map hue to a nice color (blue-teal-cyan range)
	h := hue - float64(int(hue))
	r := uint8(sinNorm(h*6.28+4.19) * 200)
	g := uint8(sinNorm(h*6.28+2.09) * 230)
	b := uint8(sinNorm(h*6.28) * 255)
	return r, g, b
}

