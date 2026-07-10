//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Pac-Man arcade game with ghosts chasing through a maze.
// --- Pac-Man Effect ---

const (
	pacCellSize      = 10
	pacAlpha         = 130
	pacGhostAlpha    = 110
	pacDotAlpha      = 80
	pacWallAlpha     = 60
	pacFrameInterval = 120 * time.Millisecond
	pacMazeW         = 21
	pacMazeH         = 11
)

// Simple maze layout: 1=wall, 0=path
var pacMaze = [pacMazeH][pacMazeW]uint8{
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 1, 1, 0, 1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 1, 0, 1, 1, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 1, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 1, 0, 1},
	{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1},
	{1, 0, 1, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 1, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 1, 1, 0, 1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 1, 0, 1, 1, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
}

type pacEntity struct {
	x, y int
	dir  int // 0=right, 1=down, 2=left, 3=up
}

type pacmanState struct {
	width, height int
	cellSize      int
	offsetX       int
	offsetY       int

	pacman     pacEntity
	ghosts     [4]pacEntity
	dots       [pacMazeH][pacMazeW]bool
	mouthOpen  bool
	frameCount int
}

func (p *shenanigans) startPacman(pc linkquisition.PickerCanvas) {
	state := &pacmanState{
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
		ticker := time.NewTicker(pacFrameInterval)
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

func (s *pacmanState) reset() {
	// Scale maze to fit window
	cellW := s.width / pacMazeW
	cellH := s.height / pacMazeH
	s.cellSize = cellW
	if cellH < cellW {
		s.cellSize = cellH
	}
	if s.cellSize < 6 {
		s.cellSize = 6
	}

	// Center the maze
	s.offsetX = (s.width - pacMazeW*s.cellSize) / 2
	s.offsetY = (s.height - pacMazeH*s.cellSize) / 2

	// Place pac-man
	s.pacman = pacEntity{x: 1, y: 1, dir: 0}

	// Place ghosts
	s.ghosts = [4]pacEntity{
		{x: 9, y: 5, dir: 0},
		{x: 10, y: 5, dir: 2},
		{x: 11, y: 5, dir: 0},
		{x: 10, y: 3, dir: 1},
	}

	// Fill dots on all path cells
	for row := range pacMazeH {
		for col := range pacMazeW {
			s.dots[row][col] = pacMaze[row][col] == 0
		}
	}
	s.frameCount = 0
}

func (s *pacmanState) update() {
	s.frameCount++
	s.mouthOpen = (s.frameCount/3)%2 == 0

	// Move pac-man (AI: prefer direction toward nearest dot)
	s.movePacman()

	// Eat dot
	s.dots[s.pacman.y][s.pacman.x] = false

	// Move ghosts
	for i := range s.ghosts {
		s.moveGhost(i)
	}

	// Check if all dots eaten — reset
	hasDots := false
	for row := range pacMazeH {
		for col := range pacMazeW {
			if s.dots[row][col] {
				hasDots = true
				break
			}
		}
		if hasDots {
			break
		}
	}
	if !hasDots {
		s.reset()
	}
}

func (s *pacmanState) movePacman() {
	// Find nearest dot
	bestDist := 9999
	bestDir := s.pacman.dir

	for _, d := range []int{0, 1, 2, 3} {
		nx, ny := s.nextCell(s.pacman.x, s.pacman.y, d)
		if pacMaze[ny][nx] == 1 {
			continue
		}
		// Don't reverse unless no other option
		if d == (s.pacman.dir+2)%4 {
			continue
		}
		dist := s.distToNearestDot(nx, ny)
		if dist < bestDist {
			bestDist = dist
			bestDir = d
		}
	}

	// If no forward option, allow reverse
	nx, ny := s.nextCell(s.pacman.x, s.pacman.y, bestDir)
	if pacMaze[ny][nx] == 1 {
		bestDir = (s.pacman.dir + 2) % 4
	}

	s.pacman.dir = bestDir
	nx, ny = s.nextCell(s.pacman.x, s.pacman.y, bestDir)
	if pacMaze[ny][nx] == 0 {
		s.pacman.x = nx
		s.pacman.y = ny
	}
}

func (s *pacmanState) moveGhost(idx int) {
	g := &s.ghosts[idx]

	// Simple AI: pick a random valid direction (not reverse) at intersections
	var options []int
	for _, d := range []int{0, 1, 2, 3} {
		if d == (g.dir+2)%4 {
			continue
		}
		nx, ny := s.nextCell(g.x, g.y, d)
		if pacMaze[ny][nx] == 0 {
			options = append(options, d)
		}
	}

	if len(options) == 0 {
		// Dead end, reverse
		g.dir = (g.dir + 2) % 4
	} else {
		g.dir = options[rand.IntN(len(options))]
	}

	nx, ny := s.nextCell(g.x, g.y, g.dir)
	if pacMaze[ny][nx] == 0 {
		g.x = nx
		g.y = ny
	}
}

func (s *pacmanState) nextCell(x, y, dir int) (nx, ny int) {
	nx, ny = x, y
	switch dir {
	case 0:
		nx++
	case 1:
		ny++
	case 2:
		nx--
	case 3:
		ny--
	}
	// Clamp to maze bounds
	if nx < 0 {
		nx = 0
	} else if nx >= pacMazeW {
		nx = pacMazeW - 1
	}
	if ny < 0 {
		ny = 0
	} else if ny >= pacMazeH {
		ny = pacMazeH - 1
	}
	return nx, ny
}

func (s *pacmanState) distToNearestDot(fromX, fromY int) int {
	best := 9999
	for row := range pacMazeH {
		for col := range pacMazeW {
			if s.dots[row][col] {
				d := abs(col-fromX) + abs(row-fromY)
				if d < best {
					best = d
				}
			}
		}
	}
	return best
}

func (s *pacmanState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw maze walls
	for row := range pacMazeH {
		for col := range pacMazeW {
			if pacMaze[row][col] == 1 {
				px := s.offsetX + col*s.cellSize
				py := s.offsetY + row*s.cellSize
				s.drawPacRect(pixels, px+1, py+1, s.cellSize-2, s.cellSize-2, 30, 30, 180, pacWallAlpha)
			}
		}
	}

	// Draw dots
	dotR := s.cellSize / 6
	if dotR < 1 {
		dotR = 1
	}
	for row := range pacMazeH {
		for col := range pacMazeW {
			if s.dots[row][col] {
				cx := s.offsetX + col*s.cellSize + s.cellSize/2
				cy := s.offsetY + row*s.cellSize + s.cellSize/2
				s.drawCircle(pixels, cx, cy, dotR, 255, 200, 150, pacDotAlpha)
			}
		}
	}

	// Draw pac-man
	s.drawPacman(pixels)

	// Draw ghosts
	ghostColors := [4][3]uint8{{255, 0, 0}, {255, 150, 200}, {0, 220, 220}, {255, 150, 0}}
	for i, g := range s.ghosts {
		cx := s.offsetX + g.x*s.cellSize + s.cellSize/2
		cy := s.offsetY + g.y*s.cellSize + s.cellSize/2
		r := s.cellSize * 2 / 5
		c := ghostColors[i]
		s.drawCircle(pixels, cx, cy, r, c[0], c[1], c[2], pacGhostAlpha)
		// Flat bottom for ghost shape
		s.drawPacRect(pixels, cx-r, cy, r*2, r, c[0], c[1], c[2], pacGhostAlpha)
	}

	return pixels
}

func (s *pacmanState) drawPacman(pixels []uint8) { //nolint:gocyclo
	cx := s.offsetX + s.pacman.x*s.cellSize + s.cellSize/2
	cy := s.offsetY + s.pacman.y*s.cellSize + s.cellSize/2
	r := s.cellSize * 2 / 5

	// Draw as circle with a mouth gap
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy > r*r {
				continue
			}
			// Mouth: skip pixels in a wedge in the facing direction
			if s.mouthOpen {
				skip := false
				switch s.pacman.dir {
				case 0: // right
					skip = dx > 0 && abs(dy) < dx
				case 2: // left
					skip = dx < 0 && abs(dy) < -dx
				case 1: // down
					skip = dy > 0 && abs(dx) < dy
				case 3: // up
					skip = dy < 0 && abs(dx) < -dy
				}
				if skip {
					continue
				}
			}
			px := cx + dx
			py := cy + dy
			if px >= 0 && px < s.width && py >= 0 && py < s.height {
				offset := (py*s.width + px) * rgbaChannels
				if pacAlpha > pixels[offset+3] {
					pixels[offset] = 255
					pixels[offset+1] = 220
					pixels[offset+2] = 0
					pixels[offset+3] = pacAlpha
				}
			}
		}
	}
}

func (s *pacmanState) drawCircle(pixels []uint8, cx, cy, r int, red, g, b, a uint8) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy > r*r {
				continue
			}
			px := cx + dx
			py := cy + dy
			if px >= 0 && px < s.width && py >= 0 && py < s.height {
				offset := (py*s.width + px) * rgbaChannels
				if a > pixels[offset+3] {
					pixels[offset] = red
					pixels[offset+1] = g
					pixels[offset+2] = b
					pixels[offset+3] = a
				}
			}
		}
	}
}

func (s *pacmanState) drawPacRect(pixels []uint8, x, y, rw, rh int, r, g, b, a uint8) {
	for dy := range rh {
		for dx := range rw {
			px := x + dx
			py := y + dy
			if px >= 0 && px < s.width && py >= 0 && py < s.height {
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
}

