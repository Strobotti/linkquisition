//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Conway's Game of Life cellular automaton simulation.
// --- Game of Life Effect ---

const (
	lifeCellSize      = 4
	lifeAlpha         = 100
	lifeFadeAlpha     = 40
	lifeFrameInterval = 80 * time.Millisecond
	lifeInitDensity   = 0.35
)

type lifeState struct {
	width, height int
	cols, rows    int
	cells         []bool
	prev          []bool
	generation    int
	staleCount    int
}

func (p *shenanigans) startLife(pc linkquisition.PickerCanvas) {
	p.startEffect(pc, effectConfig{
		state:         &lifeState{},
		opacity:       0.5,
		frameInterval: lifeFrameInterval,
	})
}

func (s *lifeState) init(width, height int) {
	s.width = width
	s.height = height
	s.cols = s.width / lifeCellSize
	s.rows = s.height / lifeCellSize
	s.randomize()
}

func (s *lifeState) update() {
	s.step()
}

func (s *lifeState) randomize() {
	size := s.cols * s.rows
	s.cells = make([]bool, size)
	s.prev = make([]bool, size)
	s.generation = 0
	s.staleCount = 0

	for i := range s.cells {
		s.cells[i] = rand.Float64() < lifeInitDensity
	}
}

func (s *lifeState) step() {
	size := s.cols * s.rows
	if size == 0 {
		return
	}

	next := make([]bool, size)
	for row := range s.rows {
		for col := range s.cols {
			neighbors := s.countNeighbors(row, col)
			idx := row*s.cols + col
			alive := s.cells[idx]

			if alive {
				next[idx] = neighbors == 2 || neighbors == 3
			} else {
				next[idx] = neighbors == 3
			}
		}
	}

	// Detect stale patterns (identical to previous generation = oscillator)
	if s.cellsEqual(next, s.prev) || s.cellsEqual(next, s.cells) {
		s.staleCount++
	} else {
		s.staleCount = 0
	}

	copy(s.prev, s.cells)
	s.cells = next
	s.generation++

	// Re-seed if the pattern has become static or died out
	if s.staleCount > 3 || s.countAlive() < size/20 {
		s.randomize()
	}
}

func (s *lifeState) countNeighbors(row, col int) int {
	count := 0
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			if dr == 0 && dc == 0 {
				continue
			}
			// Wrap around edges (toroidal grid)
			r := (row + dr + s.rows) % s.rows
			c := (col + dc + s.cols) % s.cols
			if s.cells[r*s.cols+c] {
				count++
			}
		}
	}
	return count
}

func (s *lifeState) countAlive() int {
	count := 0
	for _, alive := range s.cells {
		if alive {
			count++
		}
	}
	return count
}

func (s *lifeState) cellsEqual(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *lifeState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	for row := range s.rows {
		for col := range s.cols {
			idx := row*s.cols + col
			alive := s.cells[idx]
			wasAlive := s.prev[idx]

			if alive {
				s.drawCell(pixels, col, row, lifeAlpha)
			} else if wasAlive {
				// Fading ghost of recently dead cells
				s.drawCell(pixels, col, row, lifeFadeAlpha)
			}
		}
	}

	return pixels
}

func (s *lifeState) drawCell(pixels []uint8, col, row int, alpha uint8) {
	startX := col * lifeCellSize
	startY := row * lifeCellSize

	// Draw cell with 1px gap for grid appearance
	for dy := 0; dy < lifeCellSize-1; dy++ {
		for dx := 0; dx < lifeCellSize-1; dx++ {
			px := startX + dx
			py := startY + dy
			if px >= s.width || py >= s.height {
				continue
			}
			offset := (py*s.width + px) * rgbaChannels
			if alpha > pixels[offset+3] {
				pixels[offset] = 180   // R — slight green tint
				pixels[offset+1] = 255 // G
				pixels[offset+2] = 180 // B
				pixels[offset+3] = alpha
			}
		}
	}
}
