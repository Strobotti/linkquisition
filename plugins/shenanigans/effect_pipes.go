//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Windows 3D Pipes screensaver — pipes grow in random directions, turning at joints.

// --- Pipes Effect ---

const (
	pipesAlpha         = 180
	pipesFrameInterval = 40 * time.Millisecond
	pipesMaxSegments   = 400
	pipesSegmentLength = 8
	pipesThickness     = 6
	pipesMaxPipes      = 5
	pipesJointRadius   = 5
)

// Direction constants for pipe movement.
const (
	dirRight = iota
	dirLeft
	dirUp
	dirDown
)

type pipeSegment struct {
	x, y int
	dir  int
}

type pipe struct {
	segments []pipeSegment
	r, g, b  uint8
	headX    int
	headY    int
	dir      int
	alive    bool
}

type pipesState struct {
	width, height int
	pipes         []pipe
	totalSegments int
}

func (p *shenanigans) startPipes(pc linkquisition.PickerCanvas) {
	p.startEffect(pc, effectConfig{
		state:         &pipesState{},
		opacity:       0.6,
		frameInterval: pipesFrameInterval,
	})
}

func (s *pipesState) init(width, height int) {
	s.width = width
	s.height = height
	s.pipes = nil
	s.totalSegments = 0
	s.spawnPipe()
}

func (s *pipesState) spawnPipe() {
	// Random bright color for the pipe
	colors := [][3]uint8{
		{220, 60, 60},  // red
		{60, 200, 60},  // green
		{60, 100, 220}, // blue
		{220, 180, 40}, // yellow
		{200, 60, 200}, // magenta
		{60, 200, 200}, // cyan
		{220, 120, 40}, // orange
		{160, 80, 220}, // purple
	}
	c := colors[rand.IntN(len(colors))]

	// Start from a random edge position
	var startX, startY, dir int
	switch rand.IntN(4) {
	case 0: // left edge
		startX = 0
		startY = rand.IntN(s.height)
		dir = dirRight
	case 1: // right edge
		startX = s.width - 1
		startY = rand.IntN(s.height)
		dir = dirLeft
	case 2: // top edge
		startX = rand.IntN(s.width)
		startY = 0
		dir = dirDown
	default: // bottom edge
		startX = rand.IntN(s.width)
		startY = s.height - 1
		dir = dirUp
	}

	s.pipes = append(s.pipes, pipe{
		segments: nil,
		r:        c[0],
		g:        c[1],
		b:        c[2],
		headX:    startX,
		headY:    startY,
		dir:      dir,
		alive:    true,
	})

	// Grow a few initial segments so the pipe is visible immediately
	newPipe := &s.pipes[len(s.pipes)-1]
	for range 3 {
		s.growPipe(newPipe)
	}
}

func (s *pipesState) update() {
	// Grow each alive pipe
	for i := range s.pipes {
		if !s.pipes[i].alive {
			continue
		}
		s.growPipe(&s.pipes[i])
	}

	// Spawn new pipes periodically
	aliveCount := 0
	for i := range s.pipes {
		if s.pipes[i].alive {
			aliveCount++
		}
	}
	if aliveCount < pipesMaxPipes && rand.IntN(20) == 0 {
		s.spawnPipe()
	}

	// If we have too many segments, reset
	if s.totalSegments > pipesMaxSegments {
		s.pipes = nil
		s.totalSegments = 0
		s.spawnPipe()
	}
}

func (s *pipesState) growPipe(p *pipe) {
	// Maybe change direction (turn at a joint)
	if rand.IntN(8) == 0 {
		p.dir = s.chooseTurn(p.dir)
	}

	// Calculate new head position
	dx, dy := dirDelta(p.dir)
	newX := p.headX + dx*pipesSegmentLength
	newY := p.headY + dy*pipesSegmentLength

	// Check bounds — if out of bounds, either turn or die
	if newX < 0 || newX >= s.width || newY < 0 || newY >= s.height {
		// Try turning
		p.dir = s.chooseTurn(p.dir)
		dx, dy = dirDelta(p.dir)
		newX = p.headX + dx*pipesSegmentLength
		newY = p.headY + dy*pipesSegmentLength

		if newX < 0 || newX >= s.width || newY < 0 || newY >= s.height {
			p.alive = false
			return
		}
	}

	// Add segment
	p.segments = append(p.segments, pipeSegment{
		x:   p.headX,
		y:   p.headY,
		dir: p.dir,
	})
	p.headX = newX
	p.headY = newY
	s.totalSegments++
}

func (s *pipesState) chooseTurn(currentDir int) int {
	// Pick a perpendicular direction
	switch currentDir {
	case dirRight, dirLeft:
		if rand.IntN(2) == 0 {
			return dirUp
		}
		return dirDown
	default:
		if rand.IntN(2) == 0 {
			return dirLeft
		}
		return dirRight
	}
}

func dirDelta(dir int) (dx, dy int) {
	switch dir {
	case dirRight:
		return 1, 0
	case dirLeft:
		return -1, 0
	case dirUp:
		return 0, -1
	case dirDown:
		return 0, 1
	default:
		return 1, 0
	}
}

func (s *pipesState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw all pipe segments and joints
	for pi := range s.pipes {
		pp := &s.pipes[pi]
		for si, seg := range pp.segments {
			s.drawSegment(pixels, w, h, seg, pp.r, pp.g, pp.b)

			// Draw joint (ball) at each turn
			if si > 0 && pp.segments[si-1].dir != seg.dir {
				s.drawJoint(pixels, w, h, seg.x, seg.y, pp.r, pp.g, pp.b)
			}
		}

		// Draw the pipe head as a brighter joint
		if pp.alive {
			s.drawJoint(pixels, w, h, pp.headX, pp.headY, pp.r, pp.g, pp.b)
		}
	}

	return pixels
}

func (s *pipesState) drawSegment(pixels []uint8, w, h int, seg pipeSegment, r, g, b uint8) {
	dx, dy := dirDelta(seg.dir)
	half := pipesThickness / 2

	for step := 0; step < pipesSegmentLength; step++ {
		cx := seg.x + dx*step
		cy := seg.y + dy*step

		// Draw a thick line perpendicular to the direction
		for t := -half; t <= half; t++ {
			var px, py int
			if dx != 0 {
				// Horizontal pipe — thickness in Y
				px = cx
				py = cy + t
			} else {
				// Vertical pipe — thickness in X
				px = cx + t
				py = cy
			}

			if px >= 0 && px < w && py >= 0 && py < h {
				off := (py*w + px) * rgbaChannels
				// Shading: brighter in center, darker at edges
				shade := 1.0 - absF(float64(t))/float64(half+1)*0.4
				pixels[off] = uint8(float64(r) * shade)
				pixels[off+1] = uint8(float64(g) * shade)
				pixels[off+2] = uint8(float64(b) * shade)
				pixels[off+3] = pipesAlpha
			}
		}
	}
}

func (s *pipesState) drawJoint(pixels []uint8, w, h, cx, cy int, r, g, b uint8) {
	rad := pipesJointRadius
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			if dx*dx+dy*dy > rad*rad {
				continue
			}
			px := cx + dx
			py := cy + dy
			if px >= 0 && px < w && py >= 0 && py < h {
				off := (py*w + px) * rgbaChannels
				// Spherical shading
				dist := float64(dx*dx+dy*dy) / float64(rad*rad)
				shade := 1.0 - dist*0.5
				pixels[off] = uint8(float64(r) * shade)
				pixels[off+1] = uint8(float64(g) * shade)
				pixels[off+2] = uint8(float64(b) * shade)
				pixels[off+3] = pipesAlpha + 10
			}
		}
	}
}
