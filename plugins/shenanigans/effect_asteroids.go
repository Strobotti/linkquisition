//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Atari Asteroids arcade game with a rotating ship and floating rocks.
// --- Asteroids Effect ---

const (
	astShipSize      = 10
	astAlpha         = 120
	astBulletAlpha   = 140
	astBulletSpeed   = 6.0
	astBulletLife    = 40
	astMaxAsteroids  = 8
	astShootInterval = 15
	astTurnSpeed     = 0.05
	astThrust        = 0.12
	astFriction      = 0.98
)

type astVec struct{ x, y float64 }

type asteroid struct {
	pos    astVec
	vel    astVec
	radius float64
	edges  int // number of vertices (6-10)
}

type astBullet struct {
	pos  astVec
	vel  astVec
	life int
}

type asteroidsState struct {
	width, height int
	scale         float64

	// Ship
	shipPos   astVec
	shipVel   astVec
	shipAngle float64

	// Objects
	asteroids  []asteroid
	bullets    []astBullet
	shootTimer int
	frameCount int
}

func (p *shenanigans) startAsteroids(pc linkquisition.PickerCanvas) {
	state := &asteroidsState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}
	state.reset()

	pc.AddRasterOverlay(0.45, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.reset()
		}
		return p.invertForLight(state.render())
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

func (s *asteroidsState) reset() {
	s.scale = float64(s.width) / 600.0
	if s.scale < 0.5 {
		s.scale = 0.5
	}

	s.shipPos = astVec{float64(s.width) / 2, float64(s.height) / 2}
	s.shipVel = astVec{0, 0}
	s.shipAngle = 0
	s.bullets = nil
	s.shootTimer = 0
	s.frameCount = 0
	s.spawnAsteroids()
}

func (s *asteroidsState) spawnAsteroids() {
	s.asteroids = make([]asteroid, astMaxAsteroids)
	for i := range s.asteroids {
		// Spawn at edges
		var pos astVec
		if rand.IntN(2) == 0 {
			pos.x = rand.Float64() * float64(s.width)
			if rand.IntN(2) == 0 {
				pos.y = 0
			} else {
				pos.y = float64(s.height)
			}
		} else {
			pos.y = rand.Float64() * float64(s.height)
			if rand.IntN(2) == 0 {
				pos.x = 0
			} else {
				pos.x = float64(s.width)
			}
		}
		s.asteroids[i] = asteroid{
			pos:    pos,
			vel:    astVec{(rand.Float64() - 0.5) * 2, (rand.Float64() - 0.5) * 2},
			radius: (20 + rand.Float64()*15) * s.scale,
			edges:  6 + rand.IntN(4),
		}
	}
}

func (s *asteroidsState) update() {
	s.frameCount++

	// Ship AI: rotate toward nearest asteroid, thrust toward it, shoot
	nearest := s.nearestAsteroid()
	if nearest != nil {
		// Angle to nearest asteroid
		dx := nearest.pos.x - s.shipPos.x
		dy := nearest.pos.y - s.shipPos.y
		targetAngle := atan2Approx(dy, dx)

		// Rotate toward target
		diff := targetAngle - s.shipAngle
		for diff > 3.14159 {
			diff -= 6.28318
		}
		for diff < -3.14159 {
			diff += 6.28318
		}
		if diff > astTurnSpeed {
			s.shipAngle += astTurnSpeed
		} else if diff < -astTurnSpeed {
			s.shipAngle -= astTurnSpeed
		}

		// Thrust
		s.shipVel.x += cosApprox(s.shipAngle) * astThrust
		s.shipVel.y += sinApprox(s.shipAngle) * astThrust
	}

	// Friction
	s.shipVel.x *= astFriction
	s.shipVel.y *= astFriction

	// Move ship
	s.shipPos.x += s.shipVel.x
	s.shipPos.y += s.shipVel.y

	// Wrap ship
	s.shipPos.x = s.wrapX(s.shipPos.x)
	s.shipPos.y = s.wrapY(s.shipPos.y)

	// Shoot
	s.shootTimer++
	if s.shootTimer >= astShootInterval && nearest != nil {
		s.shootTimer = 0
		s.bullets = append(s.bullets, astBullet{
			pos:  s.shipPos,
			vel:  astVec{cosApprox(s.shipAngle) * astBulletSpeed, sinApprox(s.shipAngle) * astBulletSpeed},
			life: astBulletLife,
		})
	}

	// Move bullets
	alive := s.bullets[:0]
	for i := range s.bullets {
		s.bullets[i].pos.x += s.bullets[i].vel.x
		s.bullets[i].pos.y += s.bullets[i].vel.y
		s.bullets[i].life--
		if s.bullets[i].life > 0 {
			alive = append(alive, s.bullets[i])
		}
	}
	s.bullets = alive

	// Move asteroids
	for i := range s.asteroids {
		s.asteroids[i].pos.x = s.wrapX(s.asteroids[i].pos.x + s.asteroids[i].vel.x)
		s.asteroids[i].pos.y = s.wrapY(s.asteroids[i].pos.y + s.asteroids[i].vel.y)
	}

	// Bullet-asteroid collision
	s.checkBulletHits()

	// Respawn if all destroyed
	if len(s.asteroids) == 0 {
		s.spawnAsteroids()
	}
}

func (s *asteroidsState) checkBulletHits() {
	var newBullets []astBullet
	for _, b := range s.bullets {
		hit := false
		for i := range s.asteroids {
			dx := b.pos.x - s.asteroids[i].pos.x
			dy := b.pos.y - s.asteroids[i].pos.y
			dist := dx*dx + dy*dy
			r := s.asteroids[i].radius
			if dist < r*r {
				// Split asteroid
				if s.asteroids[i].radius > 12*s.scale {
					newR := s.asteroids[i].radius * 0.55
					for range 2 {
						s.asteroids = append(s.asteroids, asteroid{
							pos:    s.asteroids[i].pos,
							vel:    astVec{(rand.Float64() - 0.5) * 3, (rand.Float64() - 0.5) * 3},
							radius: newR,
							edges:  5 + rand.IntN(4),
						})
					}
				}
				// Remove hit asteroid
				s.asteroids = append(s.asteroids[:i], s.asteroids[i+1:]...)
				hit = true
				break
			}
		}
		if !hit {
			newBullets = append(newBullets, b)
		}
	}
	s.bullets = newBullets
}

func (s *asteroidsState) nearestAsteroid() *asteroid {
	if len(s.asteroids) == 0 {
		return nil
	}
	best := 0
	bestDist := 1e18
	for i := range s.asteroids {
		dx := s.asteroids[i].pos.x - s.shipPos.x
		dy := s.asteroids[i].pos.y - s.shipPos.y
		d := dx*dx + dy*dy
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	return &s.asteroids[best]
}

func (s *asteroidsState) wrapX(x float64) float64 {
	w := float64(s.width)
	for x < 0 {
		x += w
	}
	for x >= w {
		x -= w
	}
	return x
}

func (s *asteroidsState) wrapY(y float64) float64 {
	h := float64(s.height)
	for y < 0 {
		y += h
	}
	for y >= h {
		y -= h
	}
	return y
}

func (s *asteroidsState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw asteroids as wireframe polygons
	for i := range s.asteroids {
		s.drawAsteroid(pixels, s.asteroids[i])
	}

	// Draw bullets
	for _, b := range s.bullets {
		bx, by := int(b.pos.x), int(b.pos.y)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				s.setAstPixel(pixels, bx+dx, by+dy, 255, 255, 255, astBulletAlpha)
			}
		}
	}

	// Draw ship (triangle)
	s.drawShip(pixels)

	return pixels
}

func (s *asteroidsState) drawAsteroid(pixels []uint8, a asteroid) {
	cx, cy := a.pos.x, a.pos.y
	n := a.edges

	for i := range n {
		angle1 := float64(i) * 6.28318 / float64(n)
		angle2 := float64(i+1) * 6.28318 / float64(n)
		x1 := cx + cosApprox(angle1)*a.radius
		y1 := cy + sinApprox(angle1)*a.radius
		x2 := cx + cosApprox(angle2)*a.radius
		y2 := cy + sinApprox(angle2)*a.radius
		s.drawLine(pixels, int(x1), int(y1), int(x2), int(y2), 200, 200, 200, astAlpha)
	}
}

func (s *asteroidsState) drawShip(pixels []uint8) {
	// Triangle pointing in shipAngle direction
	size := float64(astShipSize) * s.scale
	cx, cy := s.shipPos.x, s.shipPos.y

	// Nose
	nx := cx + cosApprox(s.shipAngle)*size
	ny := cy + sinApprox(s.shipAngle)*size
	// Left wing
	lx := cx + cosApprox(s.shipAngle+2.5)*size*0.6
	ly := cy + sinApprox(s.shipAngle+2.5)*size*0.6
	// Right wing
	rx := cx + cosApprox(s.shipAngle-2.5)*size*0.6
	ry := cy + sinApprox(s.shipAngle-2.5)*size*0.6

	s.drawLine(pixels, int(nx), int(ny), int(lx), int(ly), 255, 255, 255, astAlpha)
	s.drawLine(pixels, int(nx), int(ny), int(rx), int(ry), 255, 255, 255, astAlpha)
	s.drawLine(pixels, int(lx), int(ly), int(rx), int(ry), 255, 255, 255, astAlpha)
}

// drawLine draws a line using Bresenham's algorithm.
func (s *asteroidsState) drawLine(pixels []uint8, x0, y0, x1, y1 int, r, g, b, a uint8) { //nolint:unparam
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy

	for {
		s.setAstPixel(pixels, x0, y0, r, g, b, a)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (s *asteroidsState) setAstPixel(pixels []uint8, x, y int, r, g, b, a uint8) {
	if x < 0 || x >= s.width || y < 0 || y >= s.height {
		return
	}
	offset := (y*s.width + x) * rgbaChannels
	if a > pixels[offset+3] {
		pixels[offset] = r
		pixels[offset+1] = g
		pixels[offset+2] = b
		pixels[offset+3] = a
	}
}
