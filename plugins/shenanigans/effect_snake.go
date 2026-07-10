//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Classic Snake game — the snake grows as it eats food pellets.
// --- Snake Effect ---

const (
	snakeInitLength    = 5
	snakeAlpha         = 110
	snakeFoodAlpha     = 130
	snakeFrameInterval = 90 * time.Millisecond
	snakeTargetCols    = 40
)

type snakeDir int

const (
	snakeDirUp snakeDir = iota
	snakeDirDown
	snakeDirRight
	snakeDirLeft
)

type snakePoint struct {
	x, y int
}

type snakeState struct {
	width, height int
	cols, rows    int
	cellSize      int
	body          []snakePoint
	dir           snakeDir
	food          snakePoint
	frameCount    int
}

func (p *shenanigans) startSnake(pc linkquisition.PickerCanvas) {
	state := &snakeState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}
	state.computeGrid()
	state.reset()

	pc.AddRasterOverlay(0.5, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.computeGrid()
			state.reset()
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(snakeFrameInterval)
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

func (s *snakeState) computeGrid() {
	s.cellSize = s.width / snakeTargetCols
	if s.cellSize < 6 {
		s.cellSize = 6
	}
	if s.cellSize > 16 {
		s.cellSize = 16
	}
	s.cols = s.width / s.cellSize
	s.rows = s.height / s.cellSize
}
func (s *snakeState) reset() {
	// Start in the center going right
	cx := s.cols / 2
	cy := s.rows / 2
	s.body = make([]snakePoint, snakeInitLength)
	for i := range s.body {
		s.body[i] = snakePoint{x: cx - i, y: cy}
	}
	s.dir = snakeDirRight
	s.placeFood()
}

func (s *snakeState) placeFood() {
	// Place food at a random empty cell
	for range 100 {
		p := snakePoint{x: rand.IntN(s.cols), y: rand.IntN(s.rows)}
		if !s.isBody(p) {
			s.food = p
			return
		}
	}
	// Fallback: just place it anywhere
	s.food = snakePoint{x: rand.IntN(s.cols), y: rand.IntN(s.rows)}
}

func (s *snakeState) isBody(p snakePoint) bool {
	for _, seg := range s.body {
		if seg == p {
			return true
		}
	}
	return false
}

func (s *snakeState) update() {
	s.frameCount++

	// AI: decide direction every frame
	s.chooseDirection()

	// Move head
	head := s.body[0]
	switch s.dir {
	case snakeDirUp:
		head.y--
	case snakeDirDown:
		head.y++
	case snakeDirRight:
		head.x++
	case snakeDirLeft:
		head.x--
	}

	// Wrap around edges
	head.x = (head.x + s.cols) % s.cols
	head.y = (head.y + s.rows) % s.rows

	// Check self-collision — reset if hit
	if s.isBody(head) {
		s.reset()
		return
	}

	// Grow or move
	s.body = append([]snakePoint{head}, s.body...)
	if head == s.food {
		s.placeFood()
	} else {
		s.body = s.body[:len(s.body)-1]
	}
}

func (s *snakeState) chooseDirection() {
	// Score each valid direction by distance to food after moving
	type option struct {
		dir  snakeDir
		dist int
	}

	var options []option
	for _, d := range []snakeDir{snakeDirUp, snakeDirDown, snakeDirLeft, snakeDirRight} {
		if d == s.oppositeDir() {
			continue
		}
		next := s.nextHead(d)
		if s.isBody(next) {
			continue
		}
		dist := s.wrapDist(next, s.food)
		options = append(options, option{d, dist})
	}

	if len(options) == 0 {
		return // no safe move, will hit self and reset
	}

	// Pick the direction that minimizes distance to food
	best := options[0]
	for _, o := range options[1:] {
		if o.dist < best.dist {
			best = o
		}
	}

	s.dir = best.dir
}

// wrapDist returns the Manhattan distance accounting for toroidal wrapping.
func (s *snakeState) wrapDist(a, b snakePoint) int {
	dx := abs(a.x - b.x)
	dy := abs(a.y - b.y)
	if dx > s.cols/2 {
		dx = s.cols - dx
	}
	if dy > s.rows/2 {
		dy = s.rows - dy
	}
	return dx + dy
}

func (s *snakeState) nextHead(d snakeDir) snakePoint {
	head := s.body[0]
	switch d {
	case snakeDirUp:
		head.y--
	case snakeDirDown:
		head.y++
	case snakeDirRight:
		head.x++
	case snakeDirLeft:
		head.x--
	}
	head.x = (head.x + s.cols) % s.cols
	head.y = (head.y + s.rows) % s.rows
	return head
}

func (s *snakeState) oppositeDir() snakeDir {
	switch s.dir {
	case snakeDirUp:
		return snakeDirDown
	case snakeDirDown:
		return snakeDirUp
	case snakeDirLeft:
		return snakeDirRight
	case snakeDirRight:
		return snakeDirLeft
	}
	return snakeDirLeft
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (s *snakeState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw body with gradient (head brighter, tail dimmer)
	for i, seg := range s.body {
		fade := float64(i) / float64(len(s.body))
		alpha := uint8(float64(snakeAlpha) * (1.0 - fade*0.6))
		s.drawSnakeCell(pixels, seg.x, seg.y, 150, 255, 150, alpha)
	}

	// Draw food (pulsing)
	pulse := uint8(sinNorm(float64(s.frameCount)*0.15) * 60)
	s.drawSnakeCell(pixels, s.food.x, s.food.y, 255, 100, 100, snakeFoodAlpha+pulse)

	return pixels
}

func (s *snakeState) drawSnakeCell(pixels []uint8, cx, cy int, r, g, b, a uint8) {
	startX := cx * s.cellSize
	startY := cy * s.cellSize

	// Draw with 1px padding for grid look
	for dy := 1; dy < s.cellSize-1; dy++ {
		for dx := 1; dx < s.cellSize-1; dx++ {
			px := startX + dx
			py := startY + dy
			if px >= s.width || py >= s.height {
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

