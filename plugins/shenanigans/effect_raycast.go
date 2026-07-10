//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"time"

	"github.com/strobotti/linkquisition"
)

// First-person raycaster — a Wolfenstein 3D-style maze explorer using DDA ray casting.

// --- Raycaster Effect ---

const (
	raycastAlpha         = 55
	raycastFrameInterval = 30 * time.Millisecond
	raycastMapSize       = 16
	raycastFOV           = 1.0 // ~60 degrees
	raycastMaxDist       = 16.0
	raycastResDiv        = 3 // render at 1/3 horizontal resolution
	raycastTurnSpeed     = 0.04
	raycastMoveSpeed     = 0.03
	raycastWallBuffer    = 0.4
)

// Simple maze map (1 = wall, 0 = open).
var raycastMap = [raycastMapSize][raycastMapSize]uint8{
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 1, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 1, 0, 1},
	{1, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 1, 0, 1},
	{1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1},
	{1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 0, 1, 0, 1, 0, 1},
	{1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1},
	{1, 0, 1, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0, 1},
	{1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1},
	{1, 0, 1, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 1, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 1, 0, 1, 1, 0, 1, 1, 0, 1, 1, 0, 1, 0, 1},
	{1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1},
	{1, 0, 1, 1, 0, 1, 0, 1, 1, 0, 1, 0, 1, 1, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
}

type raycastState struct {
	width, height int
	posX, posY    float64
	dirX, dirY    float64
	time          float64
	turnCommit    float64 // committed turn direction: -1, 0, or +1
}

func (p *shenanigans) startRaycast(pc linkquisition.PickerCanvas) {
	state := &raycastState{
		width:  pc.Width(),
		height: pc.Height(),
		posX:   4.5,
		posY:   1.5,
		dirX:   1.0,
		dirY:   0.0,
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}

	pc.AddRasterOverlay(0.6, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(raycastFrameInterval)
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

func (s *raycastState) update() {
	s.time += 0.02

	// If we're committed to a turn, keep turning until the path ahead is fully clear
	if s.turnCommit != 0 {
		s.rotateDir(s.turnCommit * raycastTurnSpeed * 2.5)

		// Check if we've turned enough — path must be clear at a good distance
		checkX := s.posX + s.dirX*2.0
		checkY := s.posY + s.dirY*2.0
		nearX := s.posX + s.dirX*raycastWallBuffer
		nearY := s.posY + s.dirY*raycastWallBuffer
		if s.isOpen(checkX, checkY) && s.isOpen(nearX, nearY) {
			s.turnCommit = 0
		}
		return
	}

	// Not turning — check forward and move
	nearX := s.posX + s.dirX*raycastWallBuffer
	nearY := s.posY + s.dirY*raycastWallBuffer
	if !s.isOpen(nearX, nearY) {
		// Wall right in front — commit to a turn immediately
		s.turnCommit = s.chooseTurnDirection()
		return
	}

	farX := s.posX + s.dirX*1.2
	farY := s.posY + s.dirY*1.2
	if !s.isOpen(farX, farY) {
		// Wall approaching — commit to a turn, but also move a little
		s.posX += s.dirX * raycastMoveSpeed * 0.3
		s.posY += s.dirY * raycastMoveSpeed * 0.3
		s.turnCommit = s.chooseTurnDirection()
		return
	}

	// Path is clear — move forward
	s.posX += s.dirX * raycastMoveSpeed
	s.posY += s.dirY * raycastMoveSpeed

	// Periodically try to turn left to explore more of the maze
	if int(s.time*100)%250 == 0 {
		leftDirX := s.dirY
		leftDirY := -s.dirX
		leftX := s.posX + leftDirX*2.0
		leftY := s.posY + leftDirY*2.0
		if s.isOpen(leftX, leftY) {
			s.turnCommit = -1
		}
	}
}

// chooseTurnDirection probes left and right to decide which way has more room.
func (s *raycastState) chooseTurnDirection() float64 {
	// Check right side
	rightDirX := -s.dirY
	rightDirY := s.dirX
	rightClear := 0
	for i := 1; i <= 3; i++ {
		rx := s.posX + rightDirX*float64(i)*0.5
		ry := s.posY + rightDirY*float64(i)*0.5
		if s.isOpen(rx, ry) {
			rightClear++
		}
	}

	// Check left side
	leftDirX := s.dirY
	leftDirY := -s.dirX
	leftClear := 0
	for i := 1; i <= 3; i++ {
		lx := s.posX + leftDirX*float64(i)*0.5
		ly := s.posY + leftDirY*float64(i)*0.5
		if s.isOpen(lx, ly) {
			leftClear++
		}
	}

	if leftClear > rightClear {
		return -1.0 // turn left
	}
	return 1.0 // turn right (default)
}

func (s *raycastState) isOpen(x, y float64) bool {
	ix, iy := int(x), int(y)
	if ix < 0 || ix >= raycastMapSize || iy < 0 || iy >= raycastMapSize {
		return false
	}
	return raycastMap[iy][ix] == 0
}

func (s *raycastState) rotateDir(angle float64) {
	cosA := sinApprox(angle + 1.5708) // cos(a) ≡ sin(a + π/2)
	sinA := sinApprox(angle)
	oldDirX := s.dirX
	s.dirX = s.dirX*cosA - s.dirY*sinA
	s.dirY = oldDirX*sinA + s.dirY*cosA
}

func (s *raycastState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	step := raycastResDiv

	// Draw ceiling and floor for the entire frame first
	for py := 0; py < h; py++ {
		for px := 0; px < w; px += step {
			if py < h/2 {
				// Ceiling: darker at top, lighter toward horizon
				ceilFade := 1.0 - float64(py)/float64(h/2)
				ca := uint8(float64(raycastAlpha) * (0.1 + ceilFade*0.3))
				for dx := 0; dx < step && px+dx < w; dx++ {
					off := (py*w + px + dx) * rgbaChannels
					pixels[off] = 30
					pixels[off+1] = 30
					pixels[off+2] = 50
					pixels[off+3] = ca
				}
			} else {
				// Floor: lighter toward bottom
				floorFade := float64(py-h/2) / float64(h/2)
				fa := uint8(float64(raycastAlpha) * (0.1 + floorFade*0.4))
				for dx := 0; dx < step && px+dx < w; dx++ {
					off := (py*w + px + dx) * rgbaChannels
					pixels[off] = 50
					pixels[off+1] = 45
					pixels[off+2] = 35
					pixels[off+3] = fa
				}
			}
		}
	}

	// Camera plane (perpendicular to direction)
	planeX := -s.dirY * raycastFOV * 0.5
	planeY := s.dirX * raycastFOV * 0.5

	for x := 0; x < w; x += step {
		// Ray direction for this column
		cameraX := 2.0*float64(x)/float64(w) - 1.0
		rayDirX := s.dirX + planeX*cameraX
		rayDirY := s.dirY + planeY*cameraX

		// DDA raycasting
		dist, side := s.castRay(rayDirX, rayDirY)

		if dist <= 0 {
			continue
		}

		// Calculate wall height
		lineHeight := int(float64(h) / dist)
		if lineHeight > h*2 {
			lineHeight = h * 2
		}

		drawStart := h/2 - lineHeight/2
		drawEnd := h/2 + lineHeight/2

		// Wall color (darker on Y-side for depth)
		var r, g, b uint8
		if side == 0 {
			r, g, b = 160, 80, 80 // X-side: red-brown
		} else {
			r, g, b = 120, 60, 60 // Y-side: darker
		}

		// Fog: fade with distance
		fog := 1.0 - dist/raycastMaxDist
		if fog < 0.1 {
			fog = 0.1
		}
		r = uint8(float64(r) * fog)
		g = uint8(float64(g) * fog)
		b = uint8(float64(b) * fog)

		// Draw the column (walls only — ceiling/floor already drawn)
		for px := x; px < x+step && px < w; px++ {
			for py := max(drawStart, 0); py < min(drawEnd, h); py++ {
				offset := (py*w + px) * rgbaChannels
				if raycastAlpha > pixels[offset+3] {
					pixels[offset] = r
					pixels[offset+1] = g
					pixels[offset+2] = b
					pixels[offset+3] = raycastAlpha
				}
			}
		}
	}

	return pixels
}

// castRay performs DDA raycasting and returns distance to the nearest wall and which side was hit.
func (s *raycastState) castRay(rayDirX, rayDirY float64) (float64, int) {
	mapX := int(s.posX)
	mapY := int(s.posY)

	// Length of ray from one side to next
	var deltaDistX, deltaDistY float64
	if rayDirX == 0 {
		deltaDistX = 1e30
	} else {
		deltaDistX = absF(1.0 / rayDirX)
	}
	if rayDirY == 0 {
		deltaDistY = 1e30
	} else {
		deltaDistY = absF(1.0 / rayDirY)
	}

	var stepX, stepY int
	var sideDistX, sideDistY float64

	if rayDirX < 0 {
		stepX = -1
		sideDistX = (s.posX - float64(mapX)) * deltaDistX
	} else {
		stepX = 1
		sideDistX = (float64(mapX) + 1.0 - s.posX) * deltaDistX
	}
	if rayDirY < 0 {
		stepY = -1
		sideDistY = (s.posY - float64(mapY)) * deltaDistY
	} else {
		stepY = 1
		sideDistY = (float64(mapY) + 1.0 - s.posY) * deltaDistY
	}

	// DDA
	side := 0
	for range int(raycastMaxDist * 4) {
		if sideDistX < sideDistY {
			sideDistX += deltaDistX
			mapX += stepX
			side = 0
		} else {
			sideDistY += deltaDistY
			mapY += stepY
			side = 1
		}

		if mapX < 0 || mapX >= raycastMapSize || mapY < 0 || mapY >= raycastMapSize {
			return raycastMaxDist, side
		}
		if raycastMap[mapY][mapX] > 0 {
			// Calculate perpendicular distance
			var perpDist float64
			if side == 0 {
				perpDist = sideDistX - deltaDistX
			} else {
				perpDist = sideDistY - deltaDistY
			}
			return perpDist, side
		}
	}

	return raycastMaxDist, side
}
