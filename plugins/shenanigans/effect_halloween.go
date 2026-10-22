//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"math/rand/v2"
	"time"

	"github.com/strobotti/linkquisition"
)

// A spooky scene: a big orange moon, bats flapping across the sky, ghosts
// fading in and out, and a spider dangling from a thread next to cobwebs.
// --- Halloween Effect ---

const (
	halloweenBatCount      = 7
	halloweenGhostCount    = 4
	halloweenWebSpokes     = 7
	halloweenWebRings      = 6
	halloweenFrameInterval = 30 * time.Millisecond
	halloweenPi            = 3.14159265
)

type rgb [3]uint8

type halloweenBat struct {
	x, y      float64
	vx, vy    float64
	phase     float64
	flapSpeed float64
	size      float64
}

type halloweenGhost struct {
	x, y    float64
	vx, vy  float64
	phase   float64
	size    float64
	life    float64
	lifeMax float64
}

type halloweenWeb struct {
	cornerX, cornerY float64
	dirX             float64 // 1 = grows to the right, -1 = grows to the left
	length           float64
}

type halloweenPalette struct {
	bat, ghost, web, spider, thread rgb
	batAlpha, ghostAlpha, webAlpha  uint8
}

type halloweenState struct {
	width, height int
	palette       halloweenPalette
	bats          []halloweenBat
	ghosts        []halloweenGhost
	webs          []halloweenWeb
	spiderPhase   float64
}

func (p *shenanigans) startHalloween(pc linkquisition.PickerCanvas) {
	p.startEffect(pc, effectConfig{
		state:         newHalloweenState(p.lightMode),
		opacity:       0.75,
		frameInterval: halloweenFrameInterval,
		skipInvert:    true, // custom palette for both light and dark themes
	})
}

func newHalloweenState(light bool) *halloweenState {
	pal := halloweenPalette{
		bat: rgb{95, 55, 130}, ghost: rgb{235, 235, 255}, web: rgb{220, 220, 235},
		spider: rgb{150, 100, 200}, thread: rgb{200, 200, 215},
		batAlpha: 235, ghostAlpha: 120, webAlpha: 110,
	}
	if light {
		pal = halloweenPalette{
			bat: rgb{30, 20, 45}, ghost: rgb{140, 120, 200}, web: rgb{90, 90, 100},
			spider: rgb{35, 20, 50}, thread: rgb{80, 80, 90},
			batAlpha: 235, ghostAlpha: 130, webAlpha: 130,
		}
	}
	return &halloweenState{palette: pal}
}

func (s *halloweenState) init(width, height int) {
	s.width = width
	s.height = height

	s.bats = make([]halloweenBat, halloweenBatCount)
	for i := range s.bats {
		s.bats[i] = s.newBat(true)
	}

	s.ghosts = make([]halloweenGhost, halloweenGhostCount)
	for i := range s.ghosts {
		s.ghosts[i] = s.newGhost()
		// Stagger the life cycles so ghosts don't all appear at once.
		s.ghosts[i].life = rand.Float64() * s.ghosts[i].lifeMax
	}

	webLen := float64(min(width, height)) * 0.45
	s.webs = []halloweenWeb{
		{cornerX: 0, cornerY: 0, dirX: 1, length: webLen},
		{cornerX: float64(width), cornerY: 0, dirX: -1, length: webLen * 0.6},
	}
}

func (s *halloweenState) newBat(anywhere bool) halloweenBat {
	w, h := float64(s.width), float64(s.height)
	dir := 1.0
	if rand.IntN(2) == 0 {
		dir = -1.0
	}
	x := -30.0
	if dir < 0 {
		x = w + 30
	}
	if anywhere {
		x = rand.Float64() * w
	}
	return halloweenBat{
		x:         x,
		y:         h*0.05 + rand.Float64()*h*0.65,
		vx:        dir * (1.5 + rand.Float64()*2.0),
		phase:     rand.Float64() * 6.28,
		flapSpeed: 0.35 + rand.Float64()*0.2,
		size:      7 + rand.Float64()*6,
	}
}

func (s *halloweenState) newGhost() halloweenGhost {
	w, h := float64(s.width), float64(s.height)
	return halloweenGhost{
		x:       w*0.1 + rand.Float64()*w*0.8,
		y:       h*0.3 + rand.Float64()*h*0.6,
		vx:      (rand.Float64() - 0.5) * 0.4,
		vy:      -(0.15 + rand.Float64()*0.3),
		phase:   rand.Float64() * 6.28,
		size:    22 + rand.Float64()*16,
		lifeMax: 200 + rand.Float64()*200,
	}
}

func (s *halloweenState) update() {
	w := float64(s.width)

	for i := range s.bats {
		b := &s.bats[i]
		b.phase += b.flapSpeed
		// Erratic bat flight: random vertical kicks with damping plus a lazy bob.
		b.vy += (rand.Float64() - 0.5) * 0.5
		b.vy *= 0.92
		b.x += b.vx
		b.y += b.vy + sinApprox(b.phase*0.3)*0.6
		if b.y < 5 {
			b.vy += 0.3
		}
		if b.x < -40 || b.x > w+40 {
			s.bats[i] = s.newBat(false)
		}
	}

	for i := range s.ghosts {
		g := &s.ghosts[i]
		g.life++
		g.phase += 0.05
		g.x += g.vx + sinApprox(g.phase)*0.5
		g.y += g.vy
		if g.life >= g.lifeMax {
			s.ghosts[i] = s.newGhost()
		}
	}

	s.spiderPhase += 0.02
}

func (s *halloweenState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	s.drawMoon(pixels)
	for _, web := range s.webs {
		s.drawWeb(pixels, web)
	}
	s.drawSpider(pixels)
	for i := range s.ghosts {
		s.drawGhost(pixels, &s.ghosts[i])
	}
	for i := range s.bats {
		s.drawBat(pixels, &s.bats[i])
	}

	return pixels
}

// put writes a single pixel with bounds checking. Unless force is set, the pixel
// is only written if it is more opaque than what is already there.
func (s *halloweenState) put(pixels []uint8, x, y int, c rgb, a uint8, force bool) {
	if x < 0 || x >= s.width || y < 0 || y >= s.height {
		return
	}
	o := (y*s.width + x) * rgbaChannels
	if !force && a <= pixels[o+3] {
		return
	}
	pixels[o], pixels[o+1], pixels[o+2], pixels[o+3] = c[0], c[1], c[2], a
}

func (s *halloweenState) line(pixels []uint8, x0, y0, x1, y1 float64, c rgb, a uint8) {
	dx, dy := x1-x0, y1-y0
	steps := int(max(absF(dx), absF(dy))) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		s.put(pixels, int(x0+dx*t), int(y0+dy*t), c, a, false)
	}
}

func (s *halloweenState) ellipse(pixels []uint8, cx, cy, rx, ry float64, c rgb, a uint8, force bool) {
	for y := int(cy - ry); y <= int(cy+ry)+1; y++ {
		for x := int(cx - rx); x <= int(cx+rx)+1; x++ {
			nx := (float64(x) - cx) / rx
			ny := (float64(y) - cy) / ry
			if nx*nx+ny*ny <= 1 {
				s.put(pixels, x, y, c, a, force)
			}
		}
	}
}

func (s *halloweenState) drawMoon(pixels []uint8) {
	cx := float64(s.width) * 0.82
	cy := float64(s.height) * 0.22
	r := float64(min(s.width, s.height)) * 0.14
	glowR := r * 1.7
	// The overlay is drawn at reduced opacity, so use a bright, near-opaque disc;
	// otherwise the orange blends with the dark background into a muddy brown.
	moon := rgb{255, 214, 120}
	glow := rgb{255, 160, 50}

	for y := int(cy - glowR); y <= int(cy+glowR); y++ {
		for x := int(cx - glowR); x <= int(cx+glowR); x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			d2 := dx*dx + dy*dy
			switch {
			case d2 <= r*r:
				s.put(pixels, x, y, moon, 245, false)
			case d2 <= glowR*glowR:
				f := 1 - d2/(glowR*glowR)
				s.put(pixels, x, y, glow, uint8(30*f*f), false)
			}
		}
	}

	// Craters: subtle, slightly darker than the disc.
	crater := rgb{225, 175, 85}
	s.ellipse(pixels, cx-r*0.35, cy-r*0.2, r*0.22, r*0.22, crater, 245, true)
	s.ellipse(pixels, cx+r*0.3, cy+r*0.25, r*0.16, r*0.16, crater, 245, true)
	s.ellipse(pixels, cx+r*0.1, cy-r*0.5, r*0.1, r*0.1, crater, 245, true)
}

func (s *halloweenState) drawWeb(pixels []uint8, web halloweenWeb) {
	pal := s.palette
	point := func(spoke, ring int) (float64, float64) {
		a := (halloweenPi / 2) * float64(spoke) / float64(halloweenWebSpokes-1)
		d := web.length * float64(ring) / float64(halloweenWebRings)
		return web.cornerX + web.dirX*cosApprox(a)*d, web.cornerY + sinApprox(a)*d
	}

	for i := range halloweenWebSpokes {
		x, y := point(i, halloweenWebRings)
		s.line(pixels, web.cornerX, web.cornerY, x, y, pal.web, pal.webAlpha)
	}

	for k := 1; k <= halloweenWebRings; k++ {
		for i := 0; i < halloweenWebSpokes-1; i++ {
			x0, y0 := point(i, k)
			x1, y1 := point(i+1, k)
			// Sag the strand slightly towards the corner.
			mx := web.cornerX + ((x0+x1)/2-web.cornerX)*0.88
			my := web.cornerY + ((y0+y1)/2-web.cornerY)*0.88
			s.line(pixels, x0, y0, mx, my, pal.web, pal.webAlpha)
			s.line(pixels, mx, my, x1, y1, pal.web, pal.webAlpha)
		}
	}
}

func (s *halloweenState) drawSpider(pixels []uint8) {
	pal := s.palette
	sx := float64(s.width) * 0.18
	h := float64(s.height)

	// Descend, dangle, climb back up, dangle — with plateaus at both ends.
	e := min(max((sinNorm(s.spiderPhase)-0.15)/0.7, 0), 1)
	sy := h*0.08 + e*h*0.25

	s.line(pixels, sx, 0, sx, sy, pal.thread, 150)

	// Legs
	attachY := sy + 3
	for j := range 4 {
		wiggle := sinApprox(s.spiderPhase*8+float64(j)) * 1.5
		for _, sgn := range []float64{-1, 1} {
			kx := sx + sgn*10
			ky := attachY - 6 + float64(j)*4 + wiggle
			s.line(pixels, sx, attachY, kx, ky, pal.spider, 230)
			s.line(pixels, kx, ky, kx+sgn*7, ky+7, pal.spider, 230)
		}
	}

	// Body: head and abdomen
	s.ellipse(pixels, sx, sy, 3.5, 3.5, pal.spider, 240, false)
	s.ellipse(pixels, sx, sy+8, 5.5, 6.5, pal.spider, 240, false)

	red := rgb{255, 40, 40}
	s.put(pixels, int(sx)-2, int(sy)-1, red, 255, true)
	s.put(pixels, int(sx)+1, int(sy)-1, red, 255, true)
}

func (s *halloweenState) drawGhost(pixels []uint8, g *halloweenGhost) {
	pal := s.palette
	fade := max(sinApprox(halloweenPi*g.life/g.lifeMax), 0)
	alpha := uint8(float64(pal.ghostAlpha) * fade)
	if alpha < 3 {
		return
	}

	r := g.size / 2
	bodyH := r * 1.3

	for y := int(g.y - r); y <= int(g.y+bodyH)+4; y++ {
		for x := int(g.x - r); x <= int(g.x+r)+1; x++ {
			dx, dy := float64(x)-g.x, float64(y)-g.y
			inside := false
			if dy <= 0 {
				inside = dx*dx+dy*dy <= r*r
			} else if absF(dx) <= r {
				hem := bodyH + 3*sinApprox(dx*0.6+g.phase*3)
				inside = dy <= hem
			}
			if inside {
				s.put(pixels, x, y, pal.ghost, alpha, false)
			}
		}
	}

	// Face: darker, more opaque eyes and mouth.
	faceA := uint8(min(int(alpha)+70, 255))
	face := rgb{20, 20, 40}
	s.ellipse(pixels, g.x-r*0.35, g.y-r*0.15, r*0.14, r*0.22, face, faceA, true)
	s.ellipse(pixels, g.x+r*0.35, g.y-r*0.15, r*0.14, r*0.22, face, faceA, true)
	s.ellipse(pixels, g.x, g.y+r*0.35, r*0.17, r*0.22, face, faceA, true)
}

func (s *halloweenState) drawBat(pixels []uint8, b *halloweenBat) {
	pal := s.palette
	flap := sinApprox(b.phase)
	span := int(b.size * 1.8)

	for _, sgn := range []float64{-1, 1} {
		for i := 1; i <= span; i++ {
			frac := float64(i) / float64(span)
			fi := float64(i)
			top := b.y - (0.15+0.7*flap)*fi
			scallop := 2 * absF(sinApprox(fi*0.9)) * frac
			bottom := b.y + b.size*0.35*(1-frac) + scallop
			if top > bottom {
				top, bottom = bottom, top
			}
			x := int(b.x + sgn*fi)
			for y := int(top); y <= int(bottom); y++ {
				s.put(pixels, x, y, pal.bat, pal.batAlpha, false)
			}
		}
	}

	// Body and ears
	s.ellipse(pixels, b.x, b.y, b.size*0.28, b.size*0.42, pal.bat, pal.batAlpha, false)
	for _, sgn := range []float64{-1, 1} {
		ex := b.x + sgn*b.size*0.18
		s.line(pixels, ex, b.y-b.size*0.35, ex+sgn*1.5, b.y-b.size*0.65, pal.bat, pal.batAlpha)
	}

	red := rgb{255, 40, 40}
	s.put(pixels, int(b.x-b.size*0.12), int(b.y-b.size*0.15), red, 255, true)
	s.put(pixels, int(b.x+b.size*0.12), int(b.y-b.size*0.15), red, 255, true)
}
