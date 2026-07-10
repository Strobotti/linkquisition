//nolint:mnd,gosec // Visual effects plugin: magic numbers and weak random are by design.
package main

import (
	"time"

	"github.com/strobotti/linkquisition"
)

// Retro demoscene sine-wave text scroller with color cycling.
// --- Sine Scroller Effect ---

const (
	sineScrollAlpha         = 55
	sineScrollBarAlpha      = 35
	sineScrollFrameInterval = 25 * time.Millisecond
	sineScrollMessage       = "  NOBODY EXPECTS THE LINKQUISITION!   ...GREETINGS TO ALL DEMOSCENE ENTHUSIASTS...   "
)

type sineScrollState struct {
	width, height int
	scrollX       float64
	time          float64
}

func (p *shenanigans) startSineScroll(pc linkquisition.PickerCanvas) {
	state := &sineScrollState{
		width:  pc.Width(),
		height: pc.Height(),
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
		ticker := time.NewTicker(sineScrollFrameInterval)
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

func (s *sineScrollState) update() {
	s.scrollX += 2.5
	s.time += 0.04
}

func (s *sineScrollState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw copper bars (horizontal color gradient bands that move)
	s.drawCopperBars(pixels)

	// Draw the sine-scrolling text
	s.drawSineText(pixels)

	return pixels
}

func (s *sineScrollState) drawCopperBars(pixels []uint8) {
	w, h := s.width, s.height
	barCount := 5
	barHeight := h / 8

	for i := range barCount {
		// Each bar oscillates vertically at different speeds
		centerY := float64(h)/2 + sinApprox(s.time*0.8+float64(i)*1.3)*float64(h)*0.3
		barTop := int(centerY) - barHeight/2

		for py := barTop; py < barTop+barHeight; py++ {
			if py < 0 || py >= h {
				continue
			}
			// Distance from bar center for smooth edges
			dist := absF(float64(py)-centerY) / float64(barHeight/2)
			if dist > 1.0 {
				continue
			}
			fade := (1.0 - dist) * (1.0 - dist)

			// Color based on bar index and time
			phase := s.time*0.5 + float64(i)*0.8
			r := uint8(sinNorm(phase) * 200 * fade)
			g := uint8(sinNorm(phase+2.09) * 200 * fade)
			b := uint8(sinNorm(phase+4.19) * 200 * fade)
			a := uint8(float64(sineScrollBarAlpha) * fade)

			for px := 0; px < w; px++ {
				offset := (py*w + px) * rgbaChannels
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

func (s *sineScrollState) drawSineText(pixels []uint8) {
	w, h := s.width, s.height
	msg := sineScrollMessage
	charHeight := 14

	// Pixel scale for the font
	pixScale := max(h/120, 1)
	scaledCharW := 10 * pixScale
	scaledCharH := charHeight * pixScale
	totalScrollW := len(msg) * scaledCharW

	// Current scroll offset within the repeating message
	scrollOffset := int(s.scrollX) % totalScrollW

	for i, ch := range msg {
		// Character x position (scrolling)
		cx := i*scaledCharW - scrollOffset

		// Wrap: if character scrolled off left, wrap to right
		if cx < -scaledCharW {
			cx += totalScrollW
		}
		if cx > w {
			continue
		}

		// Sine wave Y offset per character
		wave := sinApprox((float64(cx)/float64(w)*4.0+s.time)*3.14159) * float64(h) * 0.15
		cy := h/2 - scaledCharH/2 + int(wave)

		// Color: rainbow based on position
		hue := float64(i)/float64(len(msg)) + s.time*0.1
		r, g, b := sineScrollColor(hue)

		// Draw the character glyph
		s.drawChar(pixels, ch, cx, cy, pixScale, r, g, b)
	}
}

func (s *sineScrollState) drawChar(pixels []uint8, ch rune, cx, cy, scale int, r, g, b uint8) {
	w, h := s.width, s.height
	glyph := sineScrollGlyph(ch)

	for row := range 7 {
		for col := range 5 {
			if glyph[row]&(1<<(4-col)) == 0 {
				continue
			}
			// Draw scaled pixel
			for sy := range scale {
				for sx := range scale {
					px := cx + col*scale + sx + scale
					py := cy + row*scale*2 + sy
					if px >= 0 && px < w && py >= 0 && py < h {
						offset := (py*w + px) * rgbaChannels
						if sineScrollAlpha > pixels[offset+3] {
							pixels[offset] = r
							pixels[offset+1] = g
							pixels[offset+2] = b
							pixels[offset+3] = sineScrollAlpha
						}
					}
				}
			}
		}
	}
}

// sineScrollColor returns a rainbow color for a given hue [0,1).
func sineScrollColor(hue float64) (uint8, uint8, uint8) {
	h := hue - float64(int(hue))
	r := sinNorm(h*6.28) * 255
	g := sinNorm(h*6.28+2.09) * 255
	b := sinNorm(h*6.28+4.19) * 255
	return uint8(r), uint8(g), uint8(b)
}

// Simple 5x7 font glyphs for uppercase + space/punctuation.
func sineScrollGlyph(ch rune) [7]uint8 {
	if ch >= 'a' && ch <= 'z' {
		ch = ch - 'a' + 'A'
	}
	if g, ok := sineScrollFont[ch]; ok {
		return g
	}
	return [7]uint8{}
}

var sineScrollFont = map[rune][7]uint8{
	'A': {0x0E, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'B': {0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E},
	'C': {0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E},
	'D': {0x1E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x1E},
	'E': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F},
	'F': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x10},
	'G': {0x0E, 0x11, 0x10, 0x17, 0x11, 0x11, 0x0E},
	'H': {0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'I': {0x0E, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'J': {0x07, 0x02, 0x02, 0x02, 0x02, 0x12, 0x0C},
	'K': {0x11, 0x12, 0x14, 0x18, 0x14, 0x12, 0x11},
	'L': {0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1F},
	'M': {0x11, 0x1B, 0x15, 0x15, 0x11, 0x11, 0x11},
	'N': {0x11, 0x19, 0x15, 0x13, 0x11, 0x11, 0x11},
	'O': {0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'P': {0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10},
	'Q': {0x0E, 0x11, 0x11, 0x11, 0x15, 0x12, 0x0D},
	'R': {0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11},
	'S': {0x0E, 0x11, 0x10, 0x0E, 0x01, 0x11, 0x0E},
	'T': {0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	'U': {0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'V': {0x11, 0x11, 0x11, 0x11, 0x0A, 0x0A, 0x04},
	'W': {0x11, 0x11, 0x11, 0x15, 0x15, 0x1B, 0x11},
	'X': {0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11},
	'Y': {0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04},
	'Z': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x10, 0x1F},
	'!': {0x04, 0x04, 0x04, 0x04, 0x04, 0x00, 0x04},
	'.': {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
	' ': {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
}

