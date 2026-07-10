//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Tetris — falling tetrominoes stacking and clearing lines.
// --- Tetris Effect ---

const (
	tetrisRows          = 20
	tetrisCols          = 10
	tetrisAlpha         = 70
	tetrisGhostAlpha    = 20
	tetrisGridAlpha     = 20
	tetrisFrameInterval = 400 * time.Millisecond
)

// Standard Tetris piece shapes as 4x4 bitmaps (row-major).
// Each piece is defined in its spawn orientation.
var tetrisPieces = [7][4][4]bool{
	// I-piece
	{
		{false, false, false, false},
		{true, true, true, true},
		{false, false, false, false},
		{false, false, false, false},
	},
	// O-piece
	{
		{false, true, true, false},
		{false, true, true, false},
		{false, false, false, false},
		{false, false, false, false},
	},
	// T-piece
	{
		{false, true, false, false},
		{true, true, true, false},
		{false, false, false, false},
		{false, false, false, false},
	},
	// S-piece
	{
		{false, true, true, false},
		{true, true, false, false},
		{false, false, false, false},
		{false, false, false, false},
	},
	// Z-piece
	{
		{true, true, false, false},
		{false, true, true, false},
		{false, false, false, false},
		{false, false, false, false},
	},
	// J-piece
	{
		{true, false, false, false},
		{true, true, true, false},
		{false, false, false, false},
		{false, false, false, false},
	},
	// L-piece
	{
		{false, false, true, false},
		{true, true, true, false},
		{false, false, false, false},
		{false, false, false, false},
	},
}

// NES-style Tetris colors for each piece type.
var tetrisColors = [7][3]uint8{
	{0, 240, 240}, // I - cyan
	{240, 240, 0}, // O - yellow
	{160, 0, 240}, // T - purple
	{0, 240, 0},   // S - green
	{240, 0, 0},   // Z - red
	{0, 0, 240},   // J - blue
	{240, 160, 0}, // L - orange
}

type tetrisCell struct {
	filled bool
	color  int // piece type index for color lookup
}

type tetrisPiece struct {
	kind   int
	shape  [4][4]bool
	x, y   int
	ghostY int
}

type tetrisState struct {
	width, height int
	cellSize      int
	offsetX       int // pixel offset to center the grid
	offsetY       int
	grid          [tetrisRows][tetrisCols]tetrisCell
	current       tetrisPiece
	linesCleared  int
	flashRows     []int
	flashTimer    int
}

func (p *shenanigans) startTetris(pc linkquisition.PickerCanvas) {
	state := &tetrisState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}
	state.computeLayout()
	state.spawnPiece()

	pc.AddRasterOverlay(0.6, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.computeLayout()
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(tetrisFrameInterval)
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

func (s *tetrisState) computeLayout() {
	// Size cells to fill the window height, using most of the vertical space
	s.cellSize = (s.height - 4) / tetrisRows
	if s.cellSize < 4 {
		s.cellSize = 4
	}
	// Also cap by width so the grid doesn't overflow horizontally
	maxByWidth := (s.width * 3 / 4) / tetrisCols
	if s.cellSize > maxByWidth {
		s.cellSize = maxByWidth
	}
	// Center the grid horizontally
	gridW := tetrisCols * s.cellSize
	s.offsetX = (s.width - gridW) / 2
	// Center vertically with a small top margin
	gridH := tetrisRows * s.cellSize
	s.offsetY = (s.height - gridH) / 2
}

func (s *tetrisState) spawnPiece() {
	kind := rand.IntN(len(tetrisPieces))
	s.current = tetrisPiece{
		kind:  kind,
		shape: tetrisPieces[kind],
		x:     tetrisCols/2 - 2,
		y:     0,
	}
	s.computeGhost()
}

func (s *tetrisState) computeGhost() {
	// Drop the piece as far as it can go to find the ghost position
	ghost := s.current
	for s.validPosition(ghost.shape, ghost.x, ghost.y+1) {
		ghost.y++
	}
	s.current.ghostY = ghost.y
}

func (s *tetrisState) update() {
	// Handle line clear flash
	if len(s.flashRows) > 0 {
		s.flashTimer--
		if s.flashTimer <= 0 {
			s.clearFlashRows()
			s.flashRows = nil
		}
		return
	}

	// Try to move the piece down
	if s.validPosition(s.current.shape, s.current.x, s.current.y+1) {
		s.current.y++
	} else {
		// Lock the piece
		s.lockPiece()
		// Check for completed lines
		s.checkLines()
		// Spawn a new piece
		s.spawnPiece()
		// If the new piece immediately collides, game over — reset
		if !s.validPosition(s.current.shape, s.current.x, s.current.y) {
			s.reset()
		}
	}

	// AI: try to make smart moves
	s.aiMove()
}

func (s *tetrisState) aiMove() {
	// Simple AI: evaluate all positions and rotations, pick the best one.
	// We do one horizontal move or rotation per tick to make it look natural.
	bestX, bestRot := s.findBestPlacement()

	// Rotate toward the best rotation
	currentRot := 0
	testShape := tetrisPieces[s.current.kind]
	for r := range 4 {
		if testShape == s.current.shape {
			currentRot = r
			break
		}
		testShape = tetrisRotate(testShape)
	}

	if currentRot != bestRot {
		// Rotate once
		rotated := tetrisRotate(s.current.shape)
		if s.validPosition(rotated, s.current.x, s.current.y) {
			s.current.shape = rotated
			s.computeGhost()
		}
		return
	}

	// Move horizontally toward best position
	if s.current.x < bestX {
		if s.validPosition(s.current.shape, s.current.x+1, s.current.y) {
			s.current.x++
			s.computeGhost()
		}
	} else if s.current.x > bestX {
		if s.validPosition(s.current.shape, s.current.x-1, s.current.y) {
			s.current.x--
			s.computeGhost()
		}
	}
}

func (s *tetrisState) findBestPlacement() (bestX, bestRot int) {
	bestScore := -1000000
	bestX = s.current.x
	bestRot = 0

	shape := tetrisPieces[s.current.kind]
	for rot := range 4 {
		for x := -2; x < tetrisCols; x++ {
			// Drop piece to bottom
			y := s.current.y
			for s.validPositionShape(shape, x, y+1) {
				y++
			}
			if !s.validPositionShape(shape, x, y) {
				continue
			}

			// Score the placement
			score := s.scorePlacement(shape, x, y)
			if score > bestScore {
				bestScore = score
				bestX = x
				bestRot = rot
			}
		}
		shape = tetrisRotate(shape)
	}
	return bestX, bestRot
}

func (s *tetrisState) scorePlacement(shape [4][4]bool, px, py int) int {
	score := 0

	// Reward lower placement (more toward the bottom)
	score += py * 4

	// Simulate placing the piece and check for completed lines
	tempGrid := s.grid
	for row := range 4 {
		for col := range 4 {
			if shape[row][col] {
				gy := py + row
				gx := px + col
				if gy >= 0 && gy < tetrisRows && gx >= 0 && gx < tetrisCols {
					tempGrid[gy][gx] = tetrisCell{filled: true, color: 0}
				}
			}
		}
	}

	// Count complete lines
	lines := 0
	for row := range tetrisRows {
		full := true
		for col := range tetrisCols {
			if !tempGrid[row][col].filled {
				full = false
				break
			}
		}
		if full {
			lines++
		}
	}
	score += lines * 100

	// Penalize holes (empty cells with filled cells above them)
	for col := range tetrisCols {
		foundFilled := false
		for row := range tetrisRows {
			if tempGrid[row][col].filled {
				foundFilled = true
			} else if foundFilled {
				score -= 30
			}
		}
	}

	// Penalize height differences between adjacent columns
	for col := 0; col < tetrisCols-1; col++ {
		h1 := tetrisColumnHeight(&tempGrid, col)
		h2 := tetrisColumnHeight(&tempGrid, col+1)
		diff := h1 - h2
		if diff < 0 {
			diff = -diff
		}
		score -= diff * 3
	}

	return score
}

func tetrisColumnHeight(grid *[tetrisRows][tetrisCols]tetrisCell, col int) int {
	for row := range tetrisRows {
		if grid[row][col].filled {
			return tetrisRows - row
		}
	}
	return 0
}

func (s *tetrisState) validPosition(shape [4][4]bool, px, py int) bool {
	return s.validPositionShape(shape, px, py)
}

func (s *tetrisState) validPositionShape(shape [4][4]bool, px, py int) bool {
	for row := range 4 {
		for col := range 4 {
			if !shape[row][col] {
				continue
			}
			gx := px + col
			gy := py + row
			if gx < 0 || gx >= tetrisCols || gy >= tetrisRows {
				return false
			}
			if gy >= 0 && s.grid[gy][gx].filled {
				return false
			}
		}
	}
	return true
}

func (s *tetrisState) lockPiece() {
	for row := range 4 {
		for col := range 4 {
			if s.current.shape[row][col] {
				gy := s.current.y + row
				gx := s.current.x + col
				if gy >= 0 && gy < tetrisRows && gx >= 0 && gx < tetrisCols {
					s.grid[gy][gx] = tetrisCell{filled: true, color: s.current.kind}
				}
			}
		}
	}
}

func (s *tetrisState) checkLines() {
	var completed []int
	for row := range tetrisRows {
		full := true
		for col := range tetrisCols {
			if !s.grid[row][col].filled {
				full = false
				break
			}
		}
		if full {
			completed = append(completed, row)
		}
	}
	if len(completed) > 0 {
		s.flashRows = completed
		s.flashTimer = 3 // flash for 3 ticks
		s.linesCleared += len(completed)
	}
}

func (s *tetrisState) clearFlashRows() {
	for _, row := range s.flashRows {
		// Move everything above down by one
		for r := row; r > 0; r-- {
			s.grid[r] = s.grid[r-1]
		}
		// Clear top row
		for col := range tetrisCols {
			s.grid[0][col] = tetrisCell{}
		}
	}
}

func (s *tetrisState) reset() {
	s.grid = [tetrisRows][tetrisCols]tetrisCell{}
	s.linesCleared = 0
	s.flashRows = nil
	s.spawnPiece()
}

func tetrisRotate(shape [4][4]bool) [4][4]bool {
	var rotated [4][4]bool
	for row := range 4 {
		for col := range 4 {
			rotated[col][3-row] = shape[row][col]
		}
	}
	return rotated
}

func (s *tetrisState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw grid border
	s.drawGridBorder(pixels)

	// Draw locked cells
	for row := range tetrisRows {
		for col := range tetrisCols {
			cell := s.grid[row][col]
			if cell.filled {
				// Flash effect on completed rows
				if s.isFlashRow(row) && s.flashTimer%2 == 0 {
					s.drawTetrisCell(pixels, col, row, 255, 255, 255, tetrisAlpha+20)
				} else {
					c := tetrisColors[cell.color]
					s.drawTetrisCell(pixels, col, row, c[0], c[1], c[2], tetrisAlpha)
				}
			}
		}
	}

	// Draw ghost piece
	if len(s.flashRows) == 0 {
		c := tetrisColors[s.current.kind]
		for row := range 4 {
			for col := range 4 {
				if s.current.shape[row][col] {
					gy := s.current.ghostY + row
					gx := s.current.x + col
					if gy >= 0 && gy < tetrisRows && gx >= 0 && gx < tetrisCols {
						s.drawTetrisCell(pixels, gx, gy, c[0], c[1], c[2], tetrisGhostAlpha)
					}
				}
			}
		}

		// Draw current piece
		for row := range 4 {
			for col := range 4 {
				if s.current.shape[row][col] {
					gy := s.current.y + row
					gx := s.current.x + col
					if gy >= 0 && gy < tetrisRows && gx >= 0 && gx < tetrisCols {
						s.drawTetrisCell(pixels, gx, gy, c[0], c[1], c[2], tetrisAlpha)
					}
				}
			}
		}
	}

	return pixels
}

func (s *tetrisState) isFlashRow(row int) bool {
	for _, r := range s.flashRows {
		if r == row {
			return true
		}
	}
	return false
}

func (s *tetrisState) drawTetrisCell(pixels []uint8, col, row int, r, g, b, a uint8) {
	px := s.offsetX + col*s.cellSize
	py := s.offsetY + row*s.cellSize

	// Draw filled cell with 1px border for grid look
	for dy := 1; dy < s.cellSize-1; dy++ {
		for dx := 1; dx < s.cellSize-1; dx++ {
			x := px + dx
			y := py + dy
			if x >= 0 && x < s.width && y >= 0 && y < s.height {
				offset := (y*s.width + x) * rgbaChannels
				if a > pixels[offset+3] {
					pixels[offset] = r
					pixels[offset+1] = g
					pixels[offset+2] = b
					pixels[offset+3] = a
				}
			}
		}
	}

	// Draw highlight on top edge for subtle 3D bevel effect
	highlight := uint8(min(int(a)+20, 255))
	for dx := 1; dx < s.cellSize-1; dx++ {
		x := px + dx
		y := py + 1
		if x >= 0 && x < s.width && y >= 0 && y < s.height {
			offset := (y*s.width + x) * rgbaChannels
			if highlight > pixels[offset+3] {
				pixels[offset] = min(r+30, 255)
				pixels[offset+1] = min(g+30, 255)
				pixels[offset+2] = min(b+30, 255)
				pixels[offset+3] = highlight
			}
		}
	}
}

func (s *tetrisState) drawGridBorder(pixels []uint8) {
	// Draw a subtle border around the playing field
	left := s.offsetX - 1
	right := s.offsetX + tetrisCols*s.cellSize
	top := s.offsetY - 1
	bottom := s.offsetY + tetrisRows*s.cellSize

	for y := top; y <= bottom; y++ {
		if y < 0 || y >= s.height {
			continue
		}
		// Left border
		if left >= 0 && left < s.width {
			offset := (y*s.width + left) * rgbaChannels
			pixels[offset] = 100
			pixels[offset+1] = 100
			pixels[offset+2] = 100
			pixels[offset+3] = tetrisGridAlpha + 20
		}
		// Right border
		if right >= 0 && right < s.width {
			offset := (y*s.width + right) * rgbaChannels
			pixels[offset] = 100
			pixels[offset+1] = 100
			pixels[offset+2] = 100
			pixels[offset+3] = tetrisGridAlpha + 20
		}
	}
	for x := left; x <= right; x++ {
		if x < 0 || x >= s.width {
			continue
		}
		// Bottom border
		if bottom >= 0 && bottom < s.height {
			offset := (bottom*s.width + x) * rgbaChannels
			pixels[offset] = 100
			pixels[offset+1] = 100
			pixels[offset+2] = 100
			pixels[offset+3] = tetrisGridAlpha + 20
		}
	}
}
