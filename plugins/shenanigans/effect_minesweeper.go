//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Minesweeper being auto-solved, revealing the grid progressively.
// --- Minesweeper Effect ---

const (
	minesweeperMineRatio     = 0.15
	minesweeperAlpha         = 60
	minesweeperRevealedAlpha = 50
	minesweeperNumberAlpha   = 70
	minesweeperFrameInterval = 120 * time.Millisecond
	minesweeperTargetCols    = 20
)

// Cell states for minesweeper.
const (
	msCellHidden   = 0
	msCellRevealed = 1
	msCellExploded = 3
)

type minesweeperCell struct {
	mine     bool
	state    int
	adjacent int
}

type minesweeperState struct {
	width, height int
	cols, rows    int
	cellSize      int
	offsetX       int
	offsetY       int
	grid          [][]minesweeperCell
	revealQueue   []msPoint
	gameOver      bool
	resetTimer    int
	frameCount    int
}

type msPoint struct {
	x, y int
}

func (p *shenanigans) startMinesweeper(pc linkquisition.PickerCanvas) {
	p.startEffect(pc, effectConfig{
		state:         &minesweeperState{},
		opacity:       0.6,
		frameInterval: minesweeperFrameInterval,
	})
}

func (s *minesweeperState) init(width, height int) {
	s.width = width
	s.height = height
	s.cellSize = s.width / minesweeperTargetCols
	if s.cellSize < 8 {
		s.cellSize = 8
	}
	if s.cellSize > 20 {
		s.cellSize = 20
	}
	s.cols = s.width / s.cellSize
	s.rows = s.height / s.cellSize
	s.offsetX = (s.width - s.cols*s.cellSize) / 2
	s.offsetY = (s.height - s.rows*s.cellSize) / 2
	s.reset()
}

func (s *minesweeperState) reset() {
	s.grid = make([][]minesweeperCell, s.rows)
	for r := range s.grid {
		s.grid[r] = make([]minesweeperCell, s.cols)
	}

	// Place mines
	totalCells := s.rows * s.cols
	mineCount := int(float64(totalCells) * minesweeperMineRatio)
	for placed := 0; placed < mineCount; {
		x := rand.IntN(s.cols)
		y := rand.IntN(s.rows)
		if !s.grid[y][x].mine {
			s.grid[y][x].mine = true
			placed++
		}
	}

	// Calculate adjacency numbers
	for r := range s.rows {
		for c := range s.cols {
			if s.grid[r][c].mine {
				continue
			}
			count := 0
			for dr := -1; dr <= 1; dr++ {
				for dc := -1; dc <= 1; dc++ {
					nr, nc := r+dr, c+dc
					if nr >= 0 && nr < s.rows && nc >= 0 && nc < s.cols && s.grid[nr][nc].mine {
						count++
					}
				}
			}
			s.grid[r][c].adjacent = count
		}
	}

	// Start by revealing a random safe cell (no adjacent mines) to seed the flood
	s.revealQueue = nil
	s.gameOver = false
	s.resetTimer = 0
	s.seedReveal()
}

func (s *minesweeperState) seedReveal() {
	// Find a cell with 0 adjacent mines to start a nice flood fill
	candidates := make([]msPoint, 0, 32)
	for r := range s.rows {
		for c := range s.cols {
			if !s.grid[r][c].mine && s.grid[r][c].adjacent == 0 {
				candidates = append(candidates, msPoint{c, r})
			}
		}
	}
	if len(candidates) > 0 {
		start := candidates[rand.IntN(len(candidates))]
		s.revealQueue = append(s.revealQueue, start)
	} else {
		// Fallback: pick any non-mine cell
		for r := range s.rows {
			for c := range s.cols {
				if !s.grid[r][c].mine {
					s.revealQueue = append(s.revealQueue, msPoint{c, r})
					return
				}
			}
		}
	}
}

func (s *minesweeperState) update() {
	s.frameCount++

	if s.gameOver {
		s.resetTimer--
		if s.resetTimer <= 0 {
			s.reset()
		}
		return
	}

	// Reveal cells from the queue (a few per frame for wave effect)
	revealPerFrame := max(1, (s.cols*s.rows)/80)
	for range revealPerFrame {
		if len(s.revealQueue) == 0 {
			break
		}
		// Pop from queue
		pt := s.revealQueue[0]
		s.revealQueue = s.revealQueue[1:]

		cell := &s.grid[pt.y][pt.x]
		if cell.state != msCellHidden {
			continue
		}

		if cell.mine {
			// Hit a mine — game over!
			cell.state = msCellExploded
			s.gameOver = true
			s.resetTimer = 15
			return
		}

		cell.state = msCellRevealed

		// If no adjacent mines, flood fill neighbors
		if cell.adjacent == 0 {
			for dr := -1; dr <= 1; dr++ {
				for dc := -1; dc <= 1; dc++ {
					nr, nc := pt.y+dr, pt.x+dc
					if nr >= 0 && nr < s.rows && nc >= 0 && nc < s.cols {
						if s.grid[nr][nc].state == msCellHidden {
							s.revealQueue = append(s.revealQueue, msPoint{nc, nr})
						}
					}
				}
			}
		}
	}

	// If queue is empty and game not over, pick a new unrevealed safe cell
	if len(s.revealQueue) == 0 && !s.gameOver {
		s.pickNextReveal()
	}
}

func (s *minesweeperState) pickNextReveal() {
	// Prefer cells adjacent to already-revealed cells
	for r := range s.rows {
		for c := range s.cols {
			if s.grid[r][c].state == msCellHidden && !s.grid[r][c].mine {
				if s.hasRevealedNeighbor(r, c) {
					s.revealQueue = append(s.revealQueue, msPoint{c, r})
					return
				}
			}
		}
	}
	// Fallback: pick any hidden non-mine cell
	for r := range s.rows {
		for c := range s.cols {
			if s.grid[r][c].state == msCellHidden && !s.grid[r][c].mine {
				s.revealQueue = append(s.revealQueue, msPoint{c, r})
				return
			}
		}
	}
	// All safe cells revealed — occasionally step on a mine for fun, then reset
	for r := range s.rows {
		for c := range s.cols {
			if s.grid[r][c].state == msCellHidden && s.grid[r][c].mine {
				s.revealQueue = append(s.revealQueue, msPoint{c, r})
				return
			}
		}
	}
}

func (s *minesweeperState) hasRevealedNeighbor(r, c int) bool {
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			nr, nc := r+dr, c+dc
			if nr >= 0 && nr < s.rows && nc >= 0 && nc < s.cols {
				if s.grid[nr][nc].state == msCellRevealed {
					return true
				}
			}
		}
	}
	return false
}

func (s *minesweeperState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	for r := range s.rows {
		for c := range s.cols {
			cell := s.grid[r][c]
			px := s.offsetX + c*s.cellSize
			py := s.offsetY + r*s.cellSize

			switch cell.state {
			case msCellHidden:
				// Raised button look
				s.drawMSCell(pixels, px, py, 130, 130, 130, minesweeperAlpha, true)
			case msCellRevealed:
				// Flat revealed cell
				s.drawMSCell(pixels, px, py, 180, 180, 180, minesweeperRevealedAlpha, false)
				// Draw number if > 0
				if cell.adjacent > 0 {
					s.drawMSNumber(pixels, px, py, cell.adjacent)
				}
			case msCellExploded:
				// Red explosion
				s.drawMSCell(pixels, px, py, 220, 50, 50, minesweeperAlpha+20, false)
				s.drawMSMine(pixels, px, py)
			}
		}
	}

	// If game over, reveal all mines
	if s.gameOver {
		for r := range s.rows {
			for c := range s.cols {
				if s.grid[r][c].mine && s.grid[r][c].state != msCellExploded {
					px := s.offsetX + c*s.cellSize
					py := s.offsetY + r*s.cellSize
					s.drawMSCell(pixels, px, py, 180, 180, 180, minesweeperRevealedAlpha, false)
					s.drawMSMine(pixels, px, py)
				}
			}
		}
	}

	return pixels
}

func (s *minesweeperState) drawMSCell(pixels []uint8, px, py int, r, g, b, a uint8, raised bool) {
	cs := s.cellSize
	for dy := 0; dy < cs; dy++ {
		for dx := 0; dx < cs; dx++ {
			x := px + dx
			y := py + dy
			if x < 0 || x >= s.width || y < 0 || y >= s.height {
				continue
			}
			cr, cg, cb := r, g, b
			// Border effect for raised cells
			if raised {
				if dx == 0 || dy == 0 {
					cr = min(r+40, 255)
					cg = min(g+40, 255)
					cb = min(b+40, 255)
				} else if dx == cs-1 || dy == cs-1 {
					cr = r - min(r, 40)
					cg = g - min(g, 40)
					cb = b - min(b, 40)
				}
			} else if dx == 0 || dy == 0 || dx == cs-1 || dy == cs-1 {
				// Flat border
				cr = r - min(r, 20)
				cg = g - min(g, 20)
				cb = b - min(b, 20)
			}
			offset := (y*s.width + x) * rgbaChannels
			if a > pixels[offset+3] {
				pixels[offset] = cr
				pixels[offset+1] = cg
				pixels[offset+2] = cb
				pixels[offset+3] = a
			}
		}
	}
}

// Minesweeper number colors (classic Windows style).
var msNumberColors = [8][3]uint8{
	{0, 0, 200},     // 1 - blue
	{0, 130, 0},     // 2 - green
	{200, 0, 0},     // 3 - red
	{0, 0, 130},     // 4 - dark blue
	{130, 0, 0},     // 5 - dark red
	{0, 130, 130},   // 6 - teal
	{100, 100, 100}, // 7 - gray
	{60, 60, 60},    // 8 - dark gray
}

func (s *minesweeperState) drawMSNumber(pixels []uint8, px, py, num int) {
	if num < 1 || num > 8 {
		return
	}
	color := msNumberColors[num-1]
	cs := s.cellSize

	// Draw a simple representation of the number using a small centered block pattern
	// Each digit is a 3x5 bitmap
	digitBitmaps := [9][5]uint8{
		{},                        // 0 (unused)
		{0x4, 0x4, 0x4, 0x4, 0x4}, // 1: center column
		{0x7, 0x1, 0x7, 0x4, 0x7}, // 2
		{0x7, 0x1, 0x7, 0x1, 0x7}, // 3
		{0x5, 0x5, 0x7, 0x1, 0x1}, // 4
		{0x7, 0x4, 0x7, 0x1, 0x7}, // 5
		{0x7, 0x4, 0x7, 0x5, 0x7}, // 6
		{0x7, 0x1, 0x1, 0x1, 0x1}, // 7
		{0x7, 0x5, 0x7, 0x5, 0x7}, // 8
	}

	bitmap := digitBitmaps[num]
	// Scale the 3x5 bitmap to fit within the cell
	pixSize := max(cs/6, 1)
	startX := px + (cs-3*pixSize)/2
	startY := py + (cs-5*pixSize)/2

	for row := range 5 {
		for col := range 3 {
			if bitmap[row]&(1<<(2-col)) != 0 {
				for dy := range pixSize {
					for dx := range pixSize {
						x := startX + col*pixSize + dx
						y := startY + row*pixSize + dy
						if x >= 0 && x < s.width && y >= 0 && y < s.height {
							offset := (y*s.width + x) * rgbaChannels
							if minesweeperNumberAlpha > pixels[offset+3] {
								pixels[offset] = color[0]
								pixels[offset+1] = color[1]
								pixels[offset+2] = color[2]
								pixels[offset+3] = minesweeperNumberAlpha
							}
						}
					}
				}
			}
		}
	}
}

func (s *minesweeperState) drawMSMine(pixels []uint8, px, py int) {
	cs := s.cellSize
	cx := px + cs/2
	cy := py + cs/2
	radius := cs / 4

	// Draw a small circle for the mine
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy > radius*radius {
				continue
			}
			x := cx + dx
			y := cy + dy
			if x >= 0 && x < s.width && y >= 0 && y < s.height {
				offset := (y*s.width + x) * rgbaChannels
				a := uint8(minesweeperAlpha + 20)
				if a > pixels[offset+3] {
					pixels[offset] = 30
					pixels[offset+1] = 30
					pixels[offset+2] = 30
					pixels[offset+3] = a
				}
			}
		}
	}

	// Draw spikes (4 lines)
	for d := -radius - 1; d <= radius+1; d++ {
		points := [][2]int{
			{cx + d, cy}, {cx, cy + d},
			{cx + d*7/10, cy + d*7/10}, {cx + d*7/10, cy - d*7/10},
		}
		for _, pt := range points {
			x, y := pt[0], pt[1]
			if x >= 0 && x < s.width && y >= 0 && y < s.height {
				offset := (y*s.width + x) * rgbaChannels
				a := uint8(minesweeperAlpha + 10)
				if a > pixels[offset+3] {
					pixels[offset] = 30
					pixels[offset+1] = 30
					pixels[offset+2] = 30
					pixels[offset+3] = a
				}
			}
		}
	}
}
