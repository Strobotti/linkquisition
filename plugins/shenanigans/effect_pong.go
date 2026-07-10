//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Classic Pong game with two AI paddles and a bouncing ball.
// --- Pong Effect ---

const (
	pongPaddleWidth  = 6
	pongPaddleHeight = 40
	pongBallSize     = 6
	pongPaddleMargin = 12
	pongPaddleSpeed  = 3.5
	pongBallAlpha    = 140
	pongPaddleAlpha  = 120
	pongNetAlpha     = 50
	pongNetDash      = 8
	pongNetGap       = 6
)

type pongState struct {
	width, height int

	// Ball
	ballX, ballY   float64
	ballVX, ballVY float64

	// Paddles (y is the center)
	leftY, rightY float64

	// Score
	leftScore, rightScore int
}

func (p *shenanigans) startPong(pc linkquisition.PickerCanvas) {
	state := &pongState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}
	state.resetBall()

	pc.AddRasterOverlay(0.4, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
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

func (s *pongState) resetBall() {
	s.ballX = float64(s.width) / 2
	s.ballY = float64(s.height) / 2
	s.leftY = float64(s.height) / 2
	s.rightY = float64(s.height) / 2

	// Random initial direction
	s.ballVX = 3.0 + rand.Float64()*2.0
	if rand.IntN(2) == 0 {
		s.ballVX = -s.ballVX
	}
	s.ballVY = (rand.Float64() - 0.5) * 4.0
}

func (s *pongState) update() { //nolint:gocyclo
	h := float64(s.height)
	w := float64(s.width)

	// Move ball
	s.ballX += s.ballVX
	s.ballY += s.ballVY

	// Bounce off top/bottom walls
	if s.ballY <= 0 {
		s.ballY = -s.ballY
		s.ballVY = -s.ballVY
	} else if s.ballY >= h-1 {
		s.ballY = 2*(h-1) - s.ballY
		s.ballVY = -s.ballVY
	}

	// AI for paddles — follow the ball with imperfect tracking
	s.leftY += clampPaddleMove(s.ballY-s.leftY, pongPaddleSpeed*0.85)
	s.rightY += clampPaddleMove(s.ballY-s.rightY, pongPaddleSpeed*0.9)

	// Clamp paddles within bounds
	halfPaddle := float64(pongPaddleHeight) / 2
	s.leftY = clampFloat(s.leftY, halfPaddle, h-halfPaddle)
	s.rightY = clampFloat(s.rightY, halfPaddle, h-halfPaddle)

	// Left paddle collision
	leftPaddleX := float64(pongPaddleMargin + pongPaddleWidth)
	if s.ballX <= leftPaddleX && s.ballVX < 0 {
		if s.ballY >= s.leftY-halfPaddle && s.ballY <= s.leftY+halfPaddle {
			s.ballX = leftPaddleX
			s.ballVX = -s.ballVX * (1.0 + rand.Float64()*0.1)
			// Add spin based on where ball hits the paddle
			offset := (s.ballY - s.leftY) / halfPaddle
			s.ballVY += offset * 1.5
		}
	}

	// Right paddle collision
	rightPaddleX := w - float64(pongPaddleMargin+pongPaddleWidth)
	if s.ballX >= rightPaddleX && s.ballVX > 0 {
		if s.ballY >= s.rightY-halfPaddle && s.ballY <= s.rightY+halfPaddle {
			s.ballX = rightPaddleX
			s.ballVX = -s.ballVX * (1.0 + rand.Float64()*0.1)
			offset := (s.ballY - s.rightY) / halfPaddle
			s.ballVY += offset * 1.5
		}
	}

	// Cap ball speed to prevent it from getting too fast
	maxSpeed := 8.0
	if s.ballVX > maxSpeed {
		s.ballVX = maxSpeed
	} else if s.ballVX < -maxSpeed {
		s.ballVX = -maxSpeed
	}
	if s.ballVY > maxSpeed {
		s.ballVY = maxSpeed
	} else if s.ballVY < -maxSpeed {
		s.ballVY = -maxSpeed
	}

	// Score — ball leaves the field
	if s.ballX < 0 {
		s.rightScore++
		s.resetBall()
	} else if s.ballX > w {
		s.leftScore++
		s.resetBall()
	}
}

func (s *pongState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw center net (dashed line)
	centerX := w / 2
	for y := 0; y < h; y++ {
		segment := y % (pongNetDash + pongNetGap)
		if segment < pongNetDash {
			s.setPixel(pixels, centerX, y, 255, 255, 255, pongNetAlpha)
		}
	}

	// Draw paddles
	halfPaddle := pongPaddleHeight / 2
	// Left paddle
	for dy := -halfPaddle; dy <= halfPaddle; dy++ {
		for dx := 0; dx < pongPaddleWidth; dx++ {
			px := pongPaddleMargin + dx
			py := int(s.leftY) + dy
			s.setPixel(pixels, px, py, 255, 255, 255, pongPaddleAlpha)
		}
	}
	// Right paddle
	for dy := -halfPaddle; dy <= halfPaddle; dy++ {
		for dx := 0; dx < pongPaddleWidth; dx++ {
			px := w - pongPaddleMargin - pongPaddleWidth + dx
			py := int(s.rightY) + dy
			s.setPixel(pixels, px, py, 255, 255, 255, pongPaddleAlpha)
		}
	}

	// Draw ball
	halfBall := pongBallSize / 2
	for dy := -halfBall; dy <= halfBall; dy++ {
		for dx := -halfBall; dx <= halfBall; dx++ {
			px := int(s.ballX) + dx
			py := int(s.ballY) + dy
			s.setPixel(pixels, px, py, 255, 255, 255, pongBallAlpha)
		}
	}

	// Draw score (simple dot-based digits)
	s.drawScore(pixels)

	return pixels
}

func (s *pongState) setPixel(pixels []uint8, x, y int, r, g, b, a uint8) { //nolint:unparam
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

func (s *pongState) drawScore(pixels []uint8) {
	// Simple score display near the top center
	scoreAlpha := uint8(70)
	y := 15
	// Left score — draw dots on the left of center
	cx := s.width/2 - 20
	for i := range s.leftScore {
		s.drawDot(pixels, cx-i*8, y, scoreAlpha)
	}
	// Right score — draw dots on the right of center
	cx = s.width/2 + 20
	for i := range s.rightScore {
		s.drawDot(pixels, cx+i*8, y, scoreAlpha)
	}
}

func (s *pongState) drawDot(pixels []uint8, cx, cy int, alpha uint8) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			s.setPixel(pixels, cx+dx, cy+dy, 255, 255, 255, alpha)
		}
	}
}

func clampPaddleMove(delta, maxMove float64) float64 {
	if delta > maxMove {
		return maxMove
	}
	if delta < -maxMove {
		return -maxMove
	}
	return delta
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

