//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Flappy Bird — a bird navigating through pipe gaps.
// --- Flappy Bird Effect ---

const (
	flappyGravity       = 0.35
	flappyJumpVelocity  = -5.0
	flappyPipeSpeed     = 2.0
	flappyPipeGap       = 0.28 // fraction of window height
	flappyPipeWidth     = 40
	flappyBirdSize      = 14
	flappyAlpha         = 65
	flappyBirdAlpha     = 80
	flappyFrameInterval = 25 * time.Millisecond
	flappyPipeSpacing   = 180 // pixels between pipe centers
)

type flappyPipe struct {
	x      float64
	gapY   float64 // center of the gap (0-1 normalized)
	scored bool
}

type flappyState struct {
	width, height int

	// Bird
	birdY  float64
	birdVY float64
	birdX  float64

	// Pipes
	pipes []flappyPipe

	// Game state
	score      int
	frameCount int
	dead       bool
	deadTimer  int
}

func (p *shenanigans) startFlappy(pc linkquisition.PickerCanvas) {
	state := &flappyState{
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

	pc.AddRasterOverlay(0.6, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.reset()
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(flappyFrameInterval)
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

func (s *flappyState) reset() {
	s.birdX = float64(s.width) * 0.25
	s.birdY = float64(s.height) * 0.5
	s.birdVY = 0
	s.score = 0
	s.dead = false
	s.deadTimer = 0
	s.frameCount = 0

	// Initialize pipes — enough to fill the screen plus buffer off the right edge
	s.pipes = nil
	pipeCount := s.width/flappyPipeSpacing + 3
	startX := float64(s.width) + float64(flappyPipeSpacing)/2
	for i := range pipeCount {
		s.pipes = append(s.pipes, flappyPipe{
			x:    startX + float64(i*flappyPipeSpacing),
			gapY: 0.3 + rand.Float64()*0.4,
		})
	}
}

func (s *flappyState) update() {
	s.frameCount++

	if s.dead {
		s.deadTimer--
		if s.deadTimer <= 0 {
			s.reset()
		}
		return
	}

	// Apply gravity
	s.birdVY += flappyGravity
	s.birdY += s.birdVY

	// AI: flap when needed
	s.flappyAI()

	// Move pipes
	for i := range s.pipes {
		s.pipes[i].x -= flappyPipeSpeed
	}

	// Recycle pipes that have scrolled off-screen
	if len(s.pipes) > 0 && s.pipes[0].x < -float64(flappyPipeWidth) {
		s.pipes = s.pipes[1:]
		// Add a new pipe — ensure it spawns at or beyond the right edge
		newX := s.pipes[len(s.pipes)-1].x + float64(flappyPipeSpacing)
		rightEdge := float64(s.width)
		if newX < rightEdge {
			newX = rightEdge
		}
		s.pipes = append(s.pipes, flappyPipe{
			x:    newX,
			gapY: 0.25 + rand.Float64()*0.5,
		})
	}

	// Check collisions
	if s.checkCollision() {
		s.dead = true
		s.deadTimer = 30
		return
	}

	// Score
	for i := range s.pipes {
		if !s.pipes[i].scored && s.pipes[i].x+float64(flappyPipeWidth) < s.birdX {
			s.pipes[i].scored = true
			s.score++
		}
	}
}

func (s *flappyState) flappyAI() {
	// Look at the next pipe ahead of the bird
	var nextPipe *flappyPipe
	for i := range s.pipes {
		if s.pipes[i].x+float64(flappyPipeWidth) > s.birdX {
			nextPipe = &s.pipes[i]
			break
		}
	}
	if nextPipe == nil {
		return
	}

	// Target the center of the gap
	gapCenterY := nextPipe.gapY * float64(s.height)
	gapHalf := flappyPipeGap * float64(s.height) / 2

	// Flap if bird is below the target zone or falling too fast toward it
	targetY := gapCenterY - gapHalf*0.2 // aim slightly above center
	if s.birdY > targetY && s.birdVY > -2.0 {
		s.birdVY = flappyJumpVelocity
	}

	// Don't let bird go above screen
	if s.birdY < float64(flappyBirdSize) {
		s.birdVY = 1.0
	}
}

func (s *flappyState) checkCollision() bool {
	h := float64(s.height)

	// Floor/ceiling
	if s.birdY > h-float64(flappyBirdSize) || s.birdY < 0 {
		return true
	}

	// Pipe collision
	birdLeft := s.birdX - float64(flappyBirdSize)/2
	birdRight := s.birdX + float64(flappyBirdSize)/2
	birdTop := s.birdY - float64(flappyBirdSize)/2
	birdBottom := s.birdY + float64(flappyBirdSize)/2

	for _, pipe := range s.pipes {
		pipeLeft := pipe.x
		pipeRight := pipe.x + float64(flappyPipeWidth)

		// Check horizontal overlap
		if birdRight > pipeLeft && birdLeft < pipeRight {
			gapTop := pipe.gapY*h - flappyPipeGap*h/2
			gapBottom := pipe.gapY*h + flappyPipeGap*h/2

			// Check if bird is outside the gap
			if birdTop < gapTop || birdBottom > gapBottom {
				return true
			}
		}
	}

	return false
}

func (s *flappyState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw pipes
	for _, pipe := range s.pipes {
		s.drawPipe(pixels, pipe)
	}

	// Draw bird
	s.drawBird(pixels)

	return pixels
}

func (s *flappyState) drawPipe(pixels []uint8, pipe flappyPipe) {
	w, h := s.width, s.height
	pipeLeft := int(pipe.x)
	pipeRight := pipeLeft + flappyPipeWidth

	gapTop := int(pipe.gapY*float64(h) - flappyPipeGap*float64(h)/2)
	gapBottom := int(pipe.gapY*float64(h) + flappyPipeGap*float64(h)/2)

	// Top pipe
	for py := 0; py < gapTop; py++ {
		for px := pipeLeft; px < pipeRight; px++ {
			if px < 0 || px >= w || py < 0 || py >= h {
				continue
			}
			offset := (py*w + px) * rgbaChannels
			// Green pipe with darker edges
			r, g, b := uint8(60), uint8(180), uint8(60)
			if px == pipeLeft || px == pipeRight-1 {
				r, g, b = 40, 120, 40
			}
			if py >= gapTop-4 && py < gapTop {
				// Lip at the bottom of top pipe
				r, g, b = 50, 150, 50
			}
			if flappyAlpha > pixels[offset+3] {
				pixels[offset] = r
				pixels[offset+1] = g
				pixels[offset+2] = b
				pixels[offset+3] = flappyAlpha
			}
		}
	}

	// Bottom pipe
	for py := gapBottom; py < h; py++ {
		for px := pipeLeft; px < pipeRight; px++ {
			if px < 0 || px >= w || py < 0 || py >= h {
				continue
			}
			offset := (py*w + px) * rgbaChannels
			r, g, b := uint8(60), uint8(180), uint8(60)
			if px == pipeLeft || px == pipeRight-1 {
				r, g, b = 40, 120, 40
			}
			if py >= gapBottom && py < gapBottom+4 {
				// Lip at the top of bottom pipe
				r, g, b = 50, 150, 50
			}
			if flappyAlpha > pixels[offset+3] {
				pixels[offset] = r
				pixels[offset+1] = g
				pixels[offset+2] = b
				pixels[offset+3] = flappyAlpha
			}
		}
	}
}

func (s *flappyState) drawBird(pixels []uint8) {
	w, h := s.width, s.height
	cx := int(s.birdX)
	cy := int(s.birdY)
	size := flappyBirdSize

	// Bird body (yellow circle-ish)
	for dy := -size / 2; dy <= size/2; dy++ {
		for dx := -size / 2; dx <= size/2; dx++ {
			if dx*dx+dy*dy > (size/2)*(size/2) {
				continue
			}
			px := cx + dx
			py := cy + dy
			if px < 0 || px >= w || py < 0 || py >= h {
				continue
			}
			offset := (py*w + px) * rgbaChannels
			if flappyBirdAlpha > pixels[offset+3] {
				pixels[offset] = 240
				pixels[offset+1] = 200
				pixels[offset+2] = 50
				pixels[offset+3] = flappyBirdAlpha
			}
		}
	}

	// Eye (white dot with black pupil)
	eyeX := cx + size/4
	eyeY := cy - size/6
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			px := eyeX + dx
			py := eyeY + dy
			if px >= 0 && px < w && py >= 0 && py < h {
				offset := (py*w + px) * rgbaChannels
				pixels[offset] = 255
				pixels[offset+1] = 255
				pixels[offset+2] = 255
				pixels[offset+3] = flappyBirdAlpha
			}
		}
	}
	// Pupil
	if eyeX+1 >= 0 && eyeX+1 < w && eyeY >= 0 && eyeY < h {
		offset := (eyeY*w + eyeX + 1) * rgbaChannels
		pixels[offset] = 0
		pixels[offset+1] = 0
		pixels[offset+2] = 0
		pixels[offset+3] = flappyBirdAlpha
	}

	// Beak (small orange triangle to the right)
	for dy := -1; dy <= 1; dy++ {
		for dx := 0; dx < 4-abs(dy); dx++ {
			px := cx + size/2 + dx
			py := cy + dy + 1
			if px >= 0 && px < w && py >= 0 && py < h {
				offset := (py*w + px) * rgbaChannels
				if flappyBirdAlpha > pixels[offset+3] {
					pixels[offset] = 240
					pixels[offset+1] = 120
					pixels[offset+2] = 30
					pixels[offset+3] = flappyBirdAlpha
				}
			}
		}
	}
}

