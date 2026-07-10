//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"

	"github.com/strobotti/linkquisition"
)

// Space Invaders arcade game with descending alien formations.
// --- Space Invaders Effect ---

const (
	invRows            = 4
	invCols            = 8
	invMoveInterval    = 12 // frames between invader movements
	invShootChance     = 0.015
	invBulletSpeed     = 4.0
	invPlayerBulletSpd = -5.0
	invBulletWidth     = 2
	invBulletHeight    = 5
	invAlpha           = 120
	invBulletAlpha     = 100
	invExplosionFrames = 8
)

// Classic 8x8 invader sprites (1 = filled pixel in the sprite grid)
var invaderSprites = [3][8]uint8{
	// Squid (top rows)
	{0x18, 0x3C, 0x7E, 0xDB, 0xFF, 0x24, 0x5A, 0xA5},
	// Crab (middle rows)
	{0x24, 0x3C, 0x7E, 0xDB, 0xFF, 0x7E, 0x24, 0x42},
	// Octopus (bottom rows)
	{0x18, 0x3C, 0x7E, 0xDB, 0xFF, 0xBD, 0x24, 0x42},
}

type invaderEntity struct {
	alive bool
	x, y  float64
	kind  int // 0=squid, 1=crab, 2=octopus
}

type invBullet struct {
	x, y float64
	vy   float64
}

type invExplosion struct {
	x, y   float64
	frames int
}

type invadersState struct {
	width, height int

	invaders   []invaderEntity
	bullets    []invBullet
	explosions []invExplosion

	// Movement
	moveDir    float64 // +1 right, -1 left
	moveTimer  int
	dropNext   bool
	moveAmount float64

	// Player
	playerX     float64
	playerPhase float64 // sine oscillation phase around invader center

	// Dynamic scale (computed from window size)
	pixelSize    int
	spacingX     int
	spacingY     int
	playerWidth  int
	playerHeight int

	// Game state
	frameCount int
}

func (p *shenanigans) startInvaders(pc linkquisition.PickerCanvas) {
	p.startEffect(pc, effectConfig{
		state:         &invadersState{},
		opacity:       0.4,
		frameInterval: frameInterval,
	})
}

func (s *invadersState) init(width, height int) {
	s.width = width
	s.height = height
	s.reset()
}

func (s *invadersState) reset() {
	// Compute scale from window dimensions — fill ~70% of width with the formation
	formationWidthCells := invCols * 10 // 8px sprite + 2px gap per invader in cell units
	s.pixelSize = s.width * 7 / (formationWidthCells * 10)
	if s.pixelSize < 2 {
		s.pixelSize = 2
	}
	if s.pixelSize > 6 {
		s.pixelSize = 6
	}
	s.spacingX = s.pixelSize*8 + s.pixelSize*2 // sprite width + gap
	s.spacingY = s.pixelSize*8 + s.pixelSize*2
	s.playerWidth = s.pixelSize * 6
	s.playerHeight = s.pixelSize * 3

	s.invaders = make([]invaderEntity, invRows*invCols)
	startX := float64(s.width)/2 - float64(invCols*s.spacingX)/2
	startY := float64(s.height) * 0.08

	for row := range invRows {
		for col := range invCols {
			idx := row*invCols + col
			s.invaders[idx] = invaderEntity{
				alive: true,
				x:     startX + float64(col*s.spacingX),
				y:     startY + float64(row*s.spacingY),
				kind:  row % 3,
			}
		}
	}

	s.bullets = nil
	s.explosions = nil
	s.moveDir = 1
	s.moveTimer = 0
	s.dropNext = false
	s.moveAmount = float64(s.pixelSize) * 0.7
	s.playerX = float64(s.width) / 2
	s.playerPhase = 0
	s.frameCount = 0
}

func (s *invadersState) update() {
	s.frameCount++

	// Move invaders
	s.moveTimer++
	if s.moveTimer >= invMoveInterval {
		s.moveTimer = 0
		s.moveInvaders()
	}

	// Move player (auto-AI: oscillate around the invader formation center)
	s.playerPhase += 0.04
	centerX := s.invadersCenterX()
	oscillation := sinApprox(s.playerPhase) * float64(s.width) * 0.12
	targetX := centerX + oscillation
	diff := targetX - s.playerX
	maxPlayerSpeed := 3.0
	s.playerX += clampPaddleMove(diff, maxPlayerSpeed)

	// Player shoots occasionally
	if s.frameCount%20 == 0 {
		s.bullets = append(s.bullets, invBullet{
			x: s.playerX, y: float64(s.height - s.playerHeight*2 - 2), vy: invPlayerBulletSpd,
		})
	}

	// Invaders shoot randomly
	for i := range s.invaders {
		if s.invaders[i].alive && rand.Float64() < invShootChance {
			spriteSize := float64(s.pixelSize * 8)
			s.bullets = append(s.bullets, invBullet{
				x: s.invaders[i].x + spriteSize/2, y: s.invaders[i].y + spriteSize, vy: invBulletSpeed,
			})
		}
	}

	// Update bullets
	s.updateBullets()

	// Update explosions
	alive := s.explosions[:0]
	for i := range s.explosions {
		s.explosions[i].frames--
		if s.explosions[i].frames > 0 {
			alive = append(alive, s.explosions[i])
		}
	}
	s.explosions = alive

	// Check if all invaders are dead — reset
	allDead := true
	for i := range s.invaders {
		if s.invaders[i].alive {
			allDead = false
			break
		}
	}
	if allDead {
		s.reset()
	}
}

func (s *invadersState) moveInvaders() {
	dropAmount := float64(s.pixelSize * 4)
	if s.dropNext {
		for i := range s.invaders {
			if s.invaders[i].alive {
				s.invaders[i].y += dropAmount
			}
		}
		s.dropNext = false
		s.moveDir = -s.moveDir
		return
	}

	// Check if any invader hits the edge
	spriteSize := float64(s.pixelSize * 8)
	margin := float64(s.pixelSize * 2)
	hitEdge := false
	for i := range s.invaders {
		if !s.invaders[i].alive {
			continue
		}
		newX := s.invaders[i].x + s.moveAmount*s.moveDir
		if newX < margin || newX+spriteSize > float64(s.width)-margin {
			hitEdge = true
			break
		}
	}

	if hitEdge {
		s.dropNext = true
	} else {
		for i := range s.invaders {
			if s.invaders[i].alive {
				s.invaders[i].x += s.moveAmount * s.moveDir
			}
		}
	}

	// If invaders reach the bottom, reset
	bottomLimit := float64(s.height - s.playerHeight*3)
	for i := range s.invaders {
		if s.invaders[i].alive && s.invaders[i].y > bottomLimit {
			s.reset()
			return
		}
	}
}

func (s *invadersState) updateBullets() {
	remaining := s.bullets[:0]

	for i := range s.bullets {
		s.bullets[i].y += s.bullets[i].vy

		// Remove off-screen bullets
		if s.bullets[i].y < -10 || s.bullets[i].y > float64(s.height)+10 {
			continue
		}

		// Player bullets hit invaders
		if s.bullets[i].vy < 0 {
			hit := false
			for j := range s.invaders {
				if !s.invaders[j].alive {
					continue
				}
				if s.bulletHitsInvader(s.bullets[i], s.invaders[j]) {
					s.invaders[j].alive = false
					s.explosions = append(s.explosions, invExplosion{
						x: s.invaders[j].x, y: s.invaders[j].y, frames: invExplosionFrames,
					})
					hit = true
					break
				}
			}
			if hit {
				continue
			}
		}

		remaining = append(remaining, s.bullets[i])
	}

	s.bullets = remaining
}

func (s *invadersState) bulletHitsInvader(b invBullet, inv invaderEntity) bool {
	spriteSize := float64(s.pixelSize * 8)
	return b.x >= inv.x-2 && b.x <= inv.x+spriteSize && b.y >= inv.y-2 && b.y <= inv.y+spriteSize
}

// invadersCenterX returns the average X position of all living invaders.
func (s *invadersState) invadersCenterX() float64 {
	sumX := 0.0
	count := 0
	for i := range s.invaders {
		if s.invaders[i].alive {
			sumX += s.invaders[i].x
			count++
		}
	}
	if count == 0 {
		return float64(s.width) / 2
	}
	return sumX / float64(count)
}

func (s *invadersState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw invaders
	for i := range s.invaders {
		if s.invaders[i].alive {
			s.drawInvader(pixels, s.invaders[i])
		}
	}

	// Draw explosions
	for i := range s.explosions {
		s.drawExplosion(pixels, s.explosions[i])
	}

	// Draw bullets
	for i := range s.bullets {
		s.drawBullet(pixels, s.bullets[i])
	}

	// Draw player ship
	s.drawPlayer(pixels)

	return pixels
}

func (s *invadersState) drawInvader(pixels []uint8, inv invaderEntity) {
	sprite := invaderSprites[inv.kind]
	for row := range 8 {
		for col := range 8 {
			if sprite[row]&(1<<(7-col)) != 0 {
				px := int(inv.x) + col*s.pixelSize
				py := int(inv.y) + row*s.pixelSize
				s.drawBlock(pixels, px, py, s.pixelSize, 255, 255, 255, invAlpha)
			}
		}
	}
}

func (s *invadersState) drawExplosion(pixels []uint8, exp invExplosion) {
	// Simple expanding cross pattern
	size := (invExplosionFrames - exp.frames) * s.pixelSize
	spriteCenter := s.pixelSize * 4
	cx, cy := int(exp.x)+spriteCenter, int(exp.y)+spriteCenter
	alpha := uint8(exp.frames * 15)

	for d := -size; d <= size; d++ {
		s.setInvPixel(pixels, cx+d, cy, 255, 200, 100, alpha)
		s.setInvPixel(pixels, cx, cy+d, 255, 200, 100, alpha)
		s.setInvPixel(pixels, cx+d, cy+d, 255, 150, 50, alpha/2)
		s.setInvPixel(pixels, cx+d, cy-d, 255, 150, 50, alpha/2)
	}
}

func (s *invadersState) drawBullet(pixels []uint8, b invBullet) {
	bx, by := int(b.x), int(b.y)
	for dy := 0; dy < invBulletHeight; dy++ {
		for dx := 0; dx < invBulletWidth; dx++ {
			s.setInvPixel(pixels, bx+dx, by+dy, 255, 255, 255, invBulletAlpha)
		}
	}
}

func (s *invadersState) drawPlayer(pixels []uint8) {
	px := int(s.playerX) - s.playerWidth/2
	py := s.height - s.playerHeight*2

	// Simple ship shape: flat base with a turret
	for dx := 0; dx < s.playerWidth; dx++ {
		for dy := 0; dy < s.playerHeight/2; dy++ {
			s.setInvPixel(pixels, px+dx, py+s.playerHeight/2+dy, 255, 255, 255, invAlpha)
		}
	}
	// Turret (narrower top)
	turretX := px + s.playerWidth/4
	turretW := s.playerWidth / 2
	for dx := 0; dx < turretW; dx++ {
		for dy := 0; dy < s.playerHeight/2; dy++ {
			s.setInvPixel(pixels, turretX+dx, py+dy, 255, 255, 255, invAlpha)
		}
	}
	// Nose
	noseX := px + s.playerWidth/2 - 1
	s.setInvPixel(pixels, noseX, py-1, 255, 255, 255, invAlpha)
	s.setInvPixel(pixels, noseX+1, py-1, 255, 255, 255, invAlpha)
}

func (s *invadersState) drawBlock(pixels []uint8, x, y, size int, r, g, b, a uint8) {
	for dy := range size {
		for dx := range size {
			s.setInvPixel(pixels, x+dx, y+dy, r, g, b, a)
		}
	}
}

func (s *invadersState) setInvPixel(pixels []uint8, x, y int, r, g, b, a uint8) {
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
