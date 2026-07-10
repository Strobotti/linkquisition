//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Breakout/Arkanoid arcade game with paddle, ball, and destructible bricks.
// --- Breakout Effect ---

const (
	breakoutRows        = 5
	breakoutCols        = 10
	breakoutBallSize    = 5
	breakoutPaddleH     = 6
	breakoutAlpha       = 120
	breakoutBallAlpha   = 140
	breakoutPaddleAlpha = 110
)

var breakoutColors = [5][3]uint8{
	{220, 50, 50},  // red
	{220, 150, 0},  // orange
	{220, 220, 0},  // yellow
	{50, 200, 50},  // green
	{50, 120, 220}, // blue
}

type breakoutBrick struct {
	alive bool
	color int
}

type breakoutState struct {
	width, height int

	// Grid dimensions (computed from window)
	brickW, brickH int
	marginTop      int

	// Bricks
	bricks []breakoutBrick

	// Ball
	ballX, ballY   float64
	ballVX, ballVY float64

	// Paddle
	paddleX     float64
	paddleW     int
	paddleY     int
	paddlePhase float64
}

func (p *shenanigans) startBreakout(pc linkquisition.PickerCanvas) {
	state := &breakoutState{
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

	pc.AddRasterOverlay(0.4, func(w, h int) []uint8 {
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

func (s *breakoutState) reset() {
	// Compute brick sizes to fill ~80% of width
	s.brickW = (s.width * 8) / (breakoutCols * 10)
	if s.brickW < 10 {
		s.brickW = 10
	}
	s.brickH = s.brickW / 3
	if s.brickH < 6 {
		s.brickH = 6
	}
	s.marginTop = s.height / 8
	s.paddleW = s.brickW * 2
	s.paddleY = s.height - s.height/8

	// Create bricks
	s.bricks = make([]breakoutBrick, breakoutRows*breakoutCols)
	for row := range breakoutRows {
		for col := range breakoutCols {
			s.bricks[row*breakoutCols+col] = breakoutBrick{
				alive: true,
				color: row % len(breakoutColors),
			}
		}
	}

	// Reset ball to center, random angle
	s.ballX = float64(s.width) / 2
	s.ballY = float64(s.height) * 0.6
	s.ballVX = 3.0 + rand.Float64()*2.0
	if rand.IntN(2) == 0 {
		s.ballVX = -s.ballVX
	}
	s.ballVY = -(3.0 + rand.Float64()*1.5)

	// Paddle starts centered
	s.paddleX = float64(s.width) / 2
	s.paddlePhase = rand.Float64() * 6.28
}

func (s *breakoutState) update() {
	// Move ball
	s.ballX += s.ballVX
	s.ballY += s.ballVY

	// Bounce off walls
	if s.ballX <= 0 || s.ballX >= float64(s.width)-1 {
		s.ballVX = -s.ballVX
		s.ballX = clampFloat(s.ballX, 1, float64(s.width)-2)
	}
	if s.ballY <= 0 {
		s.ballVY = -s.ballVY
		s.ballY = 1
	}

	// Ball falls below paddle — reset
	if s.ballY > float64(s.height) {
		s.reset()
		return
	}

	// Paddle AI: follow ball with oscillation
	s.paddlePhase += 0.03
	oscillation := sinApprox(s.paddlePhase) * float64(s.width) * 0.05
	target := s.ballX + oscillation
	diff := target - s.paddleX
	s.paddleX += clampPaddleMove(diff, 4.0)
	s.paddleX = clampFloat(s.paddleX, float64(s.paddleW/2), float64(s.width-s.paddleW/2))

	// Ball-paddle collision
	halfPaddle := float64(s.paddleW) / 2
	if s.ballVY > 0 && s.ballY >= float64(s.paddleY)-float64(breakoutPaddleH) &&
		s.ballY <= float64(s.paddleY) &&
		s.ballX >= s.paddleX-halfPaddle && s.ballX <= s.paddleX+halfPaddle {
		s.ballVY = -s.ballVY
		// Spin based on where it hit
		offset := (s.ballX - s.paddleX) / halfPaddle
		s.ballVX += offset * 2.0
		s.ballY = float64(s.paddleY) - float64(breakoutPaddleH) - 1
	}

	// Ball-brick collision
	s.checkBrickCollisions()

	// Speed cap
	maxSpd := 7.0
	s.ballVX = clampFloat(s.ballVX, -maxSpd, maxSpd)
	s.ballVY = clampFloat(s.ballVY, -maxSpd, maxSpd)

	// All bricks destroyed — reset
	allDead := true
	for i := range s.bricks {
		if s.bricks[i].alive {
			allDead = false
			break
		}
	}
	if allDead {
		s.reset()
	}
}

func (s *breakoutState) checkBrickCollisions() {
	startX := (s.width - breakoutCols*s.brickW) / 2

	for row := range breakoutRows {
		for col := range breakoutCols {
			idx := row*breakoutCols + col
			if !s.bricks[idx].alive {
				continue
			}

			bx := float64(startX + col*s.brickW)
			by := float64(s.marginTop + row*(s.brickH+2))

			// Simple AABB check
			if s.ballX >= bx-float64(breakoutBallSize) &&
				s.ballX <= bx+float64(s.brickW)+float64(breakoutBallSize) &&
				s.ballY >= by-float64(breakoutBallSize) &&
				s.ballY <= by+float64(s.brickH)+float64(breakoutBallSize) {
				s.bricks[idx].alive = false
				// Determine bounce direction
				centerX := bx + float64(s.brickW)/2
				centerY := by + float64(s.brickH)/2
				dx := s.ballX - centerX
				dy := s.ballY - centerY
				if abs(int(dx))*s.brickH > abs(int(dy))*s.brickW {
					s.ballVX = -s.ballVX
				} else {
					s.ballVY = -s.ballVY
				}
				return // only break one brick per frame
			}
		}
	}
}

func (s *breakoutState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	startX := (s.width - breakoutCols*s.brickW) / 2

	// Draw bricks
	for row := range breakoutRows {
		for col := range breakoutCols {
			idx := row*breakoutCols + col
			if !s.bricks[idx].alive {
				continue
			}
			c := breakoutColors[s.bricks[idx].color]
			bx := startX + col*s.brickW
			by := s.marginTop + row*(s.brickH+2)
			s.drawRect(pixels, bx+1, by+1, s.brickW-2, s.brickH-2, c[0], c[1], c[2], breakoutAlpha)
		}
	}

	// Draw paddle
	px := int(s.paddleX) - s.paddleW/2
	s.drawRect(pixels, px, s.paddleY, s.paddleW, breakoutPaddleH, 200, 200, 200, breakoutPaddleAlpha)

	// Draw ball
	bx := int(s.ballX) - breakoutBallSize/2
	by := int(s.ballY) - breakoutBallSize/2
	s.drawRect(pixels, bx, by, breakoutBallSize, breakoutBallSize, 255, 255, 255, breakoutBallAlpha)

	return pixels
}

func (s *breakoutState) drawRect(pixels []uint8, x, y, rw, rh int, r, g, b, a uint8) {
	for dy := 0; dy < rh; dy++ {
		for dx := 0; dx < rw; dx++ {
			px := x + dx
			py := y + dy
			if px < 0 || px >= s.width || py < 0 || py >= s.height {
				continue
			}
			offset := (py*s.width + px) * rgbaChannels
			if a > pixels[offset+3] {
				pixels[offset] = r
				pixels[offset+1] = g
				pixels[offset+2] = b
				pixels[offset+3] = a
			}
		}
	}
}

