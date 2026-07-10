//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// Frogger arcade game — a frog dodging traffic and hopping on logs.
// --- Frogger Effect ---

const (
	froggerLanes         = 10
	froggerAlpha         = 60
	froggerFrogAlpha     = 80
	froggerFrameInterval = 50 * time.Millisecond
	froggerHopInterval   = 12 // frames between frog hops
)

// Lane types for the frogger grid.
const (
	froggerLaneRoad  = 0
	froggerLaneRiver = 1
	froggerLaneSafe  = 2
)

type froggerVehicle struct {
	x     float64
	speed float64
	width int
	color [3]uint8
}

type froggerLog struct {
	x     float64
	speed float64
	width int
}

type froggerLane struct {
	kind     int
	vehicles []froggerVehicle
	logs     []froggerLog
}

type froggerState struct {
	width, height int
	laneHeight    int
	offsetY       int
	lanes         []froggerLane
	frogX, frogY  int // grid positions (lane index + column)
	frogPixelX    float64
	hopTimer      int
	frameCount    int
	frogSize      int
	colWidth      int
	gridCols      int
}

func (p *shenanigans) startFrogger(pc linkquisition.PickerCanvas) {
	p.startEffect(pc, effectConfig{
		state:         &froggerState{},
		opacity:       0.6,
		frameInterval: froggerFrameInterval,
	})
}

func (s *froggerState) init(width, height int) {
	s.width = width
	s.height = height
	// Layout: lanes fill the window vertically
	totalLanes := froggerLanes + 2 // +2 for start and goal safe zones
	s.laneHeight = s.height / totalLanes
	if s.laneHeight < 10 {
		s.laneHeight = 10
	}
	s.offsetY = (s.height - totalLanes*s.laneHeight) / 2
	s.frogSize = s.laneHeight * 2 / 3
	s.gridCols = 13
	s.colWidth = s.width / s.gridCols

	// Build lanes: safe, 4 road, safe (median), 4 river, safe (goal)
	s.lanes = make([]froggerLane, totalLanes)
	s.lanes[0] = froggerLane{kind: froggerLaneSafe}                // start
	s.lanes[totalLanes-1] = froggerLane{kind: froggerLaneSafe}     // goal
	s.lanes[froggerLanes/2+1] = froggerLane{kind: froggerLaneSafe} // median

	// Road lanes (bottom half)
	roadColors := [][3]uint8{
		{200, 60, 60},  // red car
		{60, 60, 200},  // blue truck
		{200, 200, 60}, // yellow car
		{200, 100, 60}, // orange van
	}
	for i := 1; i <= froggerLanes/2; i++ {
		dir := 1.0
		if i%2 == 0 {
			dir = -1.0
		}
		speed := (0.8 + rand.Float64()*1.2) * dir
		count := 2 + rand.IntN(2)
		vehicles := make([]froggerVehicle, count)
		spacing := float64(s.width) / float64(count)
		vWidth := s.colWidth + rand.IntN(s.colWidth)
		for j := range vehicles {
			vehicles[j] = froggerVehicle{
				x:     float64(j) * spacing,
				speed: speed,
				width: vWidth,
				color: roadColors[(i+j)%len(roadColors)],
			}
		}
		s.lanes[i] = froggerLane{kind: froggerLaneRoad, vehicles: vehicles}
	}

	// River lanes (top half)
	for i := froggerLanes/2 + 2; i < froggerLanes+1; i++ {
		dir := 1.0
		if i%2 == 0 {
			dir = -1.0
		}
		speed := (0.5 + rand.Float64()*0.8) * dir
		count := 2 + rand.IntN(2)
		logs := make([]froggerLog, count)
		spacing := float64(s.width) / float64(count)
		logWidth := s.colWidth*2 + rand.IntN(s.colWidth)
		for j := range logs {
			logs[j] = froggerLog{
				x:     float64(j) * spacing,
				speed: speed,
				width: logWidth,
			}
		}
		s.lanes[i] = froggerLane{kind: froggerLaneRiver, logs: logs}
	}

	s.resetFrog()
}

func (s *froggerState) resetFrog() {
	s.frogX = s.gridCols / 2
	s.frogY = 0
	s.frogPixelX = float64(s.frogX * s.colWidth)
	s.hopTimer = 0
}

func (s *froggerState) update() {
	s.frameCount++

	// Move vehicles and logs
	for i := range s.lanes {
		lane := &s.lanes[i]
		for j := range lane.vehicles {
			lane.vehicles[j].x += lane.vehicles[j].speed
			// Wrap around
			if lane.vehicles[j].x > float64(s.width) {
				lane.vehicles[j].x = -float64(lane.vehicles[j].width)
			} else if lane.vehicles[j].x < -float64(lane.vehicles[j].width) {
				lane.vehicles[j].x = float64(s.width)
			}
		}
		for j := range lane.logs {
			lane.logs[j].x += lane.logs[j].speed
			if lane.logs[j].x > float64(s.width) {
				lane.logs[j].x = -float64(lane.logs[j].width)
			} else if lane.logs[j].x < -float64(lane.logs[j].width) {
				lane.logs[j].x = float64(s.width)
			}
		}
	}

	// If frog is on a log, move with it
	if s.frogY > 0 && s.frogY < len(s.lanes) {
		lane := &s.lanes[s.frogY]
		if lane.kind == froggerLaneRiver {
			for _, log := range lane.logs {
				if s.frogOnLog(log) {
					s.frogPixelX += log.speed
					break
				}
			}
		}
	}

	// Frog AI: hop forward periodically
	s.hopTimer++
	if s.hopTimer >= froggerHopInterval {
		s.hopTimer = 0
		s.frogAI()
	}

	// Check if frog reached the goal
	if s.frogY >= len(s.lanes)-1 {
		s.resetFrog()
	}

	// Check if frog is off-screen or hit
	if s.frogPixelX < -float64(s.frogSize) || s.frogPixelX > float64(s.width) {
		s.resetFrog()
		return
	}

	// Check collision with vehicles
	if s.frogY > 0 && s.frogY < len(s.lanes) {
		lane := &s.lanes[s.frogY]
		if lane.kind == froggerLaneRoad {
			for _, v := range lane.vehicles {
				if s.frogHitsVehicle(v) {
					s.resetFrog()
					return
				}
			}
		}
		// Check if frog is in river but not on a log
		if lane.kind == froggerLaneRiver {
			onLog := false
			for _, log := range lane.logs {
				if s.frogOnLog(log) {
					onLog = true
					break
				}
			}
			if !onLog {
				s.resetFrog()
				return
			}
		}
	}
}

func (s *froggerState) frogAI() {
	// Try to move forward (up), with some lateral dodging
	nextY := s.frogY + 1
	if nextY >= len(s.lanes) {
		s.frogY = nextY
		return
	}

	// Check if moving straight forward is safe
	if s.isSafePosition(s.frogPixelX, nextY) {
		s.frogY = nextY
		return
	}

	// Try left or right
	leftX := s.frogPixelX - float64(s.colWidth)
	rightX := s.frogPixelX + float64(s.colWidth)

	if s.isSafePosition(leftX, nextY) {
		s.frogPixelX = leftX
		s.frogY = nextY
	} else if s.isSafePosition(rightX, nextY) {
		s.frogPixelX = rightX
		s.frogY = nextY
	} else if s.isSafePosition(leftX, s.frogY) {
		// Dodge sideways in current lane
		s.frogPixelX = leftX
	} else if s.isSafePosition(rightX, s.frogY) {
		s.frogPixelX = rightX
	}
	// Otherwise stay put and wait
}

func (s *froggerState) isSafePosition(px float64, laneIdx int) bool {
	if laneIdx < 0 || laneIdx >= len(s.lanes) {
		return false
	}
	if px < 0 || px > float64(s.width-s.frogSize) {
		return false
	}
	lane := &s.lanes[laneIdx]
	frogCenter := px + float64(s.frogSize)/2

	if lane.kind == froggerLaneRoad {
		for _, v := range lane.vehicles {
			vLeft := v.x
			vRight := v.x + float64(v.width)
			if frogCenter > vLeft && frogCenter < vRight {
				return false
			}
		}
		return true
	}
	if lane.kind == froggerLaneRiver {
		for _, log := range lane.logs {
			logLeft := log.x
			logRight := log.x + float64(log.width)
			if frogCenter > logLeft && frogCenter < logRight {
				return true
			}
		}
		return false // not on any log
	}
	return true // safe lane
}

func (s *froggerState) frogOnLog(log froggerLog) bool {
	frogCenter := s.frogPixelX + float64(s.frogSize)/2
	return frogCenter > log.x && frogCenter < log.x+float64(log.width)
}

func (s *froggerState) frogHitsVehicle(v froggerVehicle) bool {
	frogCenter := s.frogPixelX + float64(s.frogSize)/2
	return frogCenter > v.x && frogCenter < v.x+float64(v.width)
}

func (s *froggerState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw lanes
	for i, lane := range s.lanes {
		laneTop := s.offsetY + (len(s.lanes)-1-i)*s.laneHeight

		switch lane.kind {
		case froggerLaneSafe:
			drawRect(pixels, s.width, s.height, 0, laneTop, w, s.laneHeight, 60, 140, 60, froggerAlpha/2)
		case froggerLaneRoad:
			drawRect(pixels, s.width, s.height, 0, laneTop, w, s.laneHeight, 50, 50, 50, froggerAlpha/2)
			// Lane markings
			for mx := 0; mx < w; mx += s.colWidth {
				markY := laneTop + s.laneHeight/2
				drawRect(pixels, s.width, s.height, mx+s.colWidth/4, markY, s.colWidth/2, 2, 200, 200, 100, froggerAlpha/3)
			}
			// Draw vehicles
			for _, v := range lane.vehicles {
				vy := laneTop + (s.laneHeight-s.frogSize)/2
				drawRect(pixels, s.width, s.height, int(v.x), vy, v.width, s.frogSize, v.color[0], v.color[1], v.color[2], froggerAlpha)
			}
		case froggerLaneRiver:
			drawRect(pixels, s.width, s.height, 0, laneTop, w, s.laneHeight, 30, 60, 150, froggerAlpha/2)
			// Draw logs
			for _, log := range lane.logs {
				ly := laneTop + (s.laneHeight-s.frogSize*2/3)/2
				drawRect(pixels, s.width, s.height, int(log.x), ly, log.width, s.frogSize*2/3, 140, 90, 40, froggerAlpha)
			}
		}
	}

	// Draw frog
	frogScreenY := s.offsetY + (len(s.lanes)-1-s.frogY)*s.laneHeight + (s.laneHeight-s.frogSize)/2
	s.drawFrog(pixels, int(s.frogPixelX), frogScreenY)

	return pixels
}

func (s *froggerState) drawFrog(pixels []uint8, fx, fy int) {
	// Simple frog shape: a green square with eyes
	size := s.frogSize
	// Body
	for dy := 2; dy < size-2; dy++ {
		for dx := 2; dx < size-2; dx++ {
			px := fx + dx
			py := fy + dy
			if px >= 0 && px < s.width && py >= 0 && py < s.height {
				offset := (py*s.width + px) * rgbaChannels
				if froggerFrogAlpha > pixels[offset+3] {
					pixels[offset] = 50
					pixels[offset+1] = 200
					pixels[offset+2] = 50
					pixels[offset+3] = froggerFrogAlpha
				}
			}
		}
	}
	// Eyes (two small white dots near the top)
	eyeY := fy + size/4
	eyeSize := max(size/6, 2)
	s.drawFrogEye(pixels, fx+size/4, eyeY, eyeSize)
	s.drawFrogEye(pixels, fx+size*3/4-eyeSize, eyeY, eyeSize)
}

func (s *froggerState) drawFrogEye(pixels []uint8, ex, ey, size int) {
	for dy := range size {
		for dx := range size {
			px := ex + dx
			py := ey + dy
			if px >= 0 && px < s.width && py >= 0 && py < s.height {
				offset := (py*s.width + px) * rgbaChannels
				a := uint8(froggerFrogAlpha + 20)
				if a > pixels[offset+3] {
					pixels[offset] = 255
					pixels[offset+1] = 255
					pixels[offset+2] = 255
					pixels[offset+3] = a
				}
			}
		}
	}
}
