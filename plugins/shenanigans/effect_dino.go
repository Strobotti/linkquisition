//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"

	"github.com/strobotti/linkquisition"
)

// Chrome dinosaur runner game — jumping over cacti in a desert.
// --- Dino Runner Effect ---

const (
	dinoGroundY     = 0.82 // fraction of height for ground line
	dinoGravity     = 0.6
	dinoJumpForce   = -10.0
	dinoRunSpeed    = 4.0
	dinoAlpha       = 130
	dinoCactusAlpha = 110
	dinoGroundAlpha = 60
)

// Dino sprites — two leg frames for running animation (16 wide x 16 tall)
// Designed to look like the Chrome offline T-rex
var dinoSpriteRun1 = [16]uint16{
	0x0000, // ................
	0x03FC, // ......XXXXXXXX..
	0x07FE, // .....XXXXXXXXX..
	0x07DE, // .....XXXXX.XXXX.
	0x07FE, // .....XXXXXXXXX..
	0x07F0, // .....XXXXXXX....
	0x7FE0, // .XXXXXXXXXX.....
	0xFFF0, // XXXXXXXXXXXX....
	0xFFF0, // XXXXXXXXXXXX....
	0x7FE0, // .XXXXXXXXXX.....
	0x3FC0, // ..XXXXXXXX......
	0x1F80, // ...XXXXXX.......
	0x0F00, // ....XXXX........
	0x0600, // .....XX.........
	0x0300, // ......XX........
	0x0000, // ................
}

var dinoSpriteRun2 = [16]uint16{
	0x0000, // ................
	0x03FC, // ......XXXXXXXX..
	0x07FE, // .....XXXXXXXXX..
	0x07DE, // .....XXXXX.XXXX.
	0x07FE, // .....XXXXXXXXX..
	0x07F0, // .....XXXXXXX....
	0x7FE0, // .XXXXXXXXXX.....
	0xFFF0, // XXXXXXXXXXXX....
	0xFFF0, // XXXXXXXXXXXX....
	0x7FE0, // .XXXXXXXXXX.....
	0x3FC0, // ..XXXXXXXX......
	0x1F80, // ...XXXXXX.......
	0x0F00, // ....XXXX........
	0x0300, // ......XX........
	0x0600, // .....XX.........
	0x0000, // ................
}

// Dead dino (eyes become X)
var dinoSpriteDead = [16]uint16{
	0x0000, // ................
	0x03FC, // ......XXXXXXXX..
	0x05FA, // .....X.XXXXX.X..
	0x07FE, // .....XXXXXXXXX..
	0x07FE, // .....XXXXXXXXX..
	0x07F0, // .....XXXXXXX....
	0x7FE0, // .XXXXXXXXXX.....
	0xFFF0, // XXXXXXXXXXXX....
	0xFFF0, // XXXXXXXXXXXX....
	0x7FE0, // .XXXXXXXXXX.....
	0x3FC0, // ..XXXXXXXX......
	0x1F80, // ...XXXXXX.......
	0x0F00, // ....XXXX........
	0x0900, // ....X..X........
	0x0900, // ....X..X........
	0x0000, // ................
}

type dinoCactus struct {
	x      float64
	height int // 1=small, 2=medium, 3=tall
}

type dinoState struct {
	width, height int
	scale         int

	// Dino
	dinoY      float64
	dinoVY     float64
	groundY    int
	dead       bool
	deathTimer int

	// Obstacles
	cacti      []dinoCactus
	spawnTimer int
	spawnDelay int

	// Game
	speed      float64
	frameCount int
}

func (p *shenanigans) startDino(pc linkquisition.PickerCanvas) {
	p.startEffect(pc, effectConfig{
		state:         &dinoState{},
		opacity:       0.45,
		frameInterval: frameInterval,
	})
}

func (s *dinoState) init(width, height int) {
	s.width = width
	s.height = height
	s.reset()
}

func (s *dinoState) reset() {
	s.scale = s.width / 200
	if s.scale < 2 {
		s.scale = 2
	}
	if s.scale > 5 {
		s.scale = 5
	}

	s.groundY = int(float64(s.height) * dinoGroundY)
	s.dinoY = float64(s.groundY)
	s.dinoVY = 0
	s.dead = false
	s.deathTimer = 0
	s.cacti = nil
	s.spawnTimer = 0
	s.spawnDelay = 40 + rand.IntN(30)
	s.speed = dinoRunSpeed
	s.frameCount = 0
}

func (s *dinoState) update() {
	s.frameCount++

	// If dead, show death sprite for a bit then reset
	if s.dead {
		s.deathTimer++
		if s.deathTimer > 40 {
			s.reset()
		}
		return
	}

	// Gradually increase speed
	if s.frameCount%100 == 0 && s.speed < 9.0 {
		s.speed += 0.2
	}

	// Jump AI: jump when a cactus is close
	onGround := s.dinoY >= float64(s.groundY)
	if onGround {
		for _, c := range s.cacti {
			dinoRight := float64(s.scale*8 + s.scale*16) // dino x + sprite width
			dist := c.x - dinoRight
			if dist > 0 && dist < s.speed*12 {
				s.dinoVY = dinoJumpForce * float64(s.scale) / 3.0
				break
			}
		}
	}

	// Apply gravity
	s.dinoVY += dinoGravity
	s.dinoY += s.dinoVY
	if s.dinoY > float64(s.groundY) {
		s.dinoY = float64(s.groundY)
		s.dinoVY = 0
	}

	// Move cacti
	for i := range s.cacti {
		s.cacti[i].x -= s.speed
	}

	// Check collision
	s.checkCollision()

	// Remove off-screen cacti
	alive := s.cacti[:0]
	for _, c := range s.cacti {
		if c.x > -float64(s.scale*20) {
			alive = append(alive, c)
		}
	}
	s.cacti = alive

	// Spawn new cacti
	s.spawnTimer++
	if s.spawnTimer >= s.spawnDelay {
		s.spawnTimer = 0
		s.spawnDelay = 30 + rand.IntN(40)
		s.cacti = append(s.cacti, dinoCactus{
			x:      float64(s.width + s.scale*5),
			height: 1 + rand.IntN(3),
		})
	}
}

func (s *dinoState) checkCollision() {
	dinoX := s.scale * 8
	dinoW := s.scale * 12 // effective body width (not full 16, trimmed)
	dinoH := s.scale * 14
	dinoTop := int(s.dinoY) + s.scale*2 // skip top empty row

	for _, c := range s.cacti {
		cx := int(c.x)
		cactusW := s.scale * 3
		cactusH := s.scale * (5 + c.height*3)
		cactusTop := s.groundY + s.scale*16 - cactusH

		// AABB overlap check
		if dinoX+dinoW > cx && dinoX < cx+cactusW &&
			dinoTop+dinoH > cactusTop && dinoTop < cactusTop+cactusH {
			s.dead = true
			s.deathTimer = 0
			return
		}
	}
}

func (s *dinoState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw ground line
	for x := 0; x < w; x++ {
		for dy := 0; dy < s.scale; dy++ {
			py := s.groundY + s.scale*16 + dy
			if py < h {
				offset := (py*w + x) * rgbaChannels
				pixels[offset] = 180
				pixels[offset+1] = 180
				pixels[offset+2] = 180
				pixels[offset+3] = dinoGroundAlpha
			}
		}
	}

	// Draw dino
	s.drawDino(pixels)

	// Draw cacti
	for _, c := range s.cacti {
		s.drawCactus(pixels, c)
	}

	return pixels
}

func (s *dinoState) drawDino(pixels []uint8) {
	dinoX := s.scale * 8 // fixed X position from left edge
	baseY := int(s.dinoY)

	// Choose sprite based on state
	var sprite *[16]uint16
	if s.dead {
		sprite = &dinoSpriteDead
	} else if s.dinoY < float64(s.groundY) {
		// In the air — use run1 (legs together)
		sprite = &dinoSpriteRun1
	} else {
		// On ground — alternate legs every 4 frames
		if (s.frameCount/4)%2 == 0 {
			sprite = &dinoSpriteRun1
		} else {
			sprite = &dinoSpriteRun2
		}
	}

	for row := range 16 {
		bits := sprite[row]
		for col := range 16 {
			if bits&(1<<(15-col)) != 0 {
				px := dinoX + col*s.scale
				py := baseY + row*s.scale
				s.drawBlock(pixels, px, py, s.scale, 255, 255, 255, dinoAlpha)
			}
		}
	}
}

func (s *dinoState) drawCactus(pixels []uint8, c dinoCactus) {
	cx := int(c.x)
	cactusW := s.scale * 3
	cactusH := s.scale * (5 + c.height*3)
	cy := s.groundY + s.scale*16 - cactusH

	// Main stem
	s.drawRect(pixels, cx, cy, cactusW, cactusH, 100, 200, 100, dinoCactusAlpha)

	// Arms for taller cacti
	if c.height >= 2 {
		armY := cy + cactusH/3
		// Left arm
		s.drawRect(pixels, cx-s.scale*2, armY, s.scale*2, s.scale*2, 100, 200, 100, dinoCactusAlpha)
		s.drawRect(pixels, cx-s.scale*2, armY-s.scale*2, s.scale, s.scale*2, 100, 200, 100, dinoCactusAlpha)
	}
	if c.height >= 3 {
		armY := cy + cactusH*2/3
		// Right arm
		s.drawRect(pixels, cx+cactusW, armY, s.scale*2, s.scale*2, 100, 200, 100, dinoCactusAlpha)
		s.drawRect(pixels, cx+cactusW+s.scale, armY-s.scale*2, s.scale, s.scale*2, 100, 200, 100, dinoCactusAlpha)
	}
}

func (s *dinoState) drawRect(pixels []uint8, x, y, rw, rh int, r, g, b, a uint8) { //nolint:unparam
	for dy := range rh {
		for dx := range rw {
			px := x + dx
			py := y + dy
			if px < 0 || px >= s.width || py < 0 || py >= s.height {
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

func (s *dinoState) drawBlock(pixels []uint8, x, y, size int, r, g, b, a uint8) {
	for dy := range size {
		for dx := range size {
			px := x + dx
			py := y + dy
			if px < 0 || px >= s.width || py < 0 || py >= s.height {
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
