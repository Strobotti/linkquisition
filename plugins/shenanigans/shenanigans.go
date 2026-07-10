//nolint:mnd,gosec,dupl // Visual effects plugin: magic numbers (colors, speeds, sizes) and weak random are by design.
package main

import (
	"context"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/strobotti/linkquisition"
)

const (
	effectMatrix      = "matrix"
	effectFire        = "fire"
	effectSnow        = "snow"
	effectPlasma      = "plasma"
	effectStarfield   = "starfield"
	effectAurora      = "aurora"
	effectGlitch      = "glitch"
	effectPride       = "pride"
	effectFootball    = "football"
	effectFireworks   = "fireworks"
	effectPong        = "pong"
	effectLife        = "life"
	effectInvaders    = "invaders"
	effectSnake       = "snake"
	effectRain        = "rain"
	effectBreakout    = "breakout"
	effectDino        = "dino"
	effectAsteroids   = "asteroids"
	effectPacman      = "pacman"
	effectRandom      = "random"
	frameInterval     = 30 * time.Millisecond
	fireFrameInterval = 25 * time.Millisecond
	fireWidth         = 200
	fireHeight        = 100
	matrixColumns     = 40
	rgbaChannels      = 4

	settingKeyEffect = "effect"
)

// Compile-time interface checks.
var _ linkquisition.Plugin = (*shenanigans)(nil)
var _ linkquisition.PluginUIHook = (*shenanigans)(nil)

type shenanigans struct {
	serviceProvider linkquisition.PluginServiceProvider
	effect          string
	stopped         atomic.Bool
	lightMode       bool
}

func (p *shenanigans) Metadata() linkquisition.PluginMetadata {
	return linkquisition.PluginMetadata{
		Name:        "Shenanigans",
		Description: "Adds completely useless but entertaining visual effects to the browser picker window",
		Author:      "Strobotti",
		Version:     "1.0.0",
		URL:         "https://github.com/Strobotti/linkquisition",
		Settings: []linkquisition.PluginSettingDescriptor{
			{
				Key:         settingKeyEffect,
				Label:       "Effect",
				Description: "Which visual effect to show on the picker window",
				Type:        linkquisition.SettingTypeChoice,
				Default:     effectRandom,
				// Options: "random" stays on top as the default; the rest MUST be
				// kept in alphabetical order for a consistent UI.
				Options: []string{
					effectRandom,
					effectAsteroids, effectAurora, effectBreakout, effectDino,
					effectFire, effectFireworks, effectFootball, effectGlitch,
					effectInvaders, effectLife, effectMatrix, effectPacman,
					effectPlasma, effectPong, effectPride, effectRain,
					effectSnake, effectSnow, effectStarfield,
				},
			},
		},
	}
}

func (p *shenanigans) Setup(
	serviceProvider linkquisition.PluginServiceProvider, config map[string]interface{},
) error {
	p.serviceProvider = serviceProvider

	if effectVal, ok := config[settingKeyEffect]; ok {
		if s, isStr := effectVal.(string); isStr {
			p.effect = s
		}
	}

	if p.effect == "" {
		p.effect = effectRandom
	}

	return nil
}

func (p *shenanigans) ProcessURL(_ context.Context, url string) linkquisition.PluginResult {
	return linkquisition.PluginResult{URL: url, Action: linkquisition.ActionContinue, ContinueChain: true}
}

func (p *shenanigans) Shutdown(_ context.Context) {
	p.stopped.Store(true)
}

func (p *shenanigans) OnPickerShown(canvas linkquisition.PickerCanvas) { //nolint:gocyclo
	p.lightMode = canvas.IsLightTheme()
	effect := p.effect

	allEffects := []string{
		effectMatrix, effectFire, effectSnow, effectPlasma,
		effectStarfield, effectAurora, effectGlitch, effectPride,
		effectFootball, effectFireworks, effectPong, effectLife,
		effectInvaders, effectSnake, effectRain, effectBreakout, effectDino,
		effectAsteroids, effectPacman,
	}

	if effect == effectRandom || !isKnownEffect(effect, allEffects) {
		effect = allEffects[rand.IntN(len(allEffects))]
	}

	p.serviceProvider.GetLogger().Debug("Shenanigans activating", "effect", effect)

	switch effect {
	case effectMatrix:
		p.startMatrixRain(canvas)
	case effectFire:
		p.startFire(canvas)
	case effectSnow:
		p.startSnow(canvas)
	case effectPlasma:
		p.startPlasma(canvas)
	case effectStarfield:
		p.startStarfield(canvas)
	case effectAurora:
		p.startAurora(canvas)
	case effectGlitch:
		p.startGlitch(canvas)
	case effectPride:
		p.startPride(canvas)
	case effectFootball:
		p.startFootball(canvas)
	case effectFireworks:
		p.startFireworks(canvas)
	case effectPong:
		p.startPong(canvas)
	case effectLife:
		p.startLife(canvas)
	case effectInvaders:
		p.startInvaders(canvas)
	case effectSnake:
		p.startSnake(canvas)
	case effectRain:
		p.startRain(canvas)
	case effectBreakout:
		p.startBreakout(canvas)
	case effectDino:
		p.startDino(canvas)
	case effectAsteroids:
		p.startAsteroids(canvas)
	case effectPacman:
		p.startPacman(canvas)
	}
}

// isKnownEffect returns true if the given effect name is in the list of known effects.
func isKnownEffect(effect string, known []string) bool {
	for _, e := range known {
		if e == effect {
			return true
		}
	}
	return false
}

// invertForLight adjusts pixel colors for light-theme visibility.
// In dark mode the buffer is returned unchanged.
// In light mode, bright (white/near-white) foreground pixels are darkened
// so they contrast against the light picker background. The background
// remains transparent, preserving full readability of the picker UI beneath.
func (p *shenanigans) invertForLight(pixels []uint8) []uint8 {
	if !p.lightMode {
		return pixels
	}
	for i := 0; i < len(pixels); i += rgbaChannels {
		if pixels[i+3] == 0 {
			continue // transparent — leave alone
		}
		// Invert RGB channels so white becomes dark and colors stay recognizable
		pixels[i] = 255 - pixels[i]
		pixels[i+1] = 255 - pixels[i+1]
		pixels[i+2] = 255 - pixels[i+2]
	}
	return pixels
}

// --- Matrix Rain Effect ---

type matrixState struct {
	columns []matrixColumn
	width   int
	height  int
}

type matrixColumn struct {
	y     float64
	speed float64
	chars []rune
}

func (p *shenanigans) startMatrixRain(pc linkquisition.PickerCanvas) {
	state := &matrixState{}
	state.width = pc.Width()
	state.height = pc.Height()
	state.initColumns()

	pc.AddRasterOverlay(0.6, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.initColumns()
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
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

func (s *matrixState) initColumns() {
	colCount := matrixColumns
	if s.width > 0 {
		colCount = s.width / 14
		if colCount < 1 {
			colCount = 1
		}
	}

	s.columns = make([]matrixColumn, colCount)
	for i := range s.columns {
		s.columns[i] = matrixColumn{
			y:     -rand.Float64() * float64(s.height),
			speed: 2 + rand.Float64()*6,
			chars: generateMatrixChars(20 + rand.IntN(15)),
		}
	}
}

func (s *matrixState) update() {
	h := float64(s.height)
	if h == 0 {
		h = 400
	}

	for i := range s.columns {
		s.columns[i].y += s.columns[i].speed
		if s.columns[i].y > h+float64(len(s.columns[i].chars)*16) {
			s.columns[i].y = -float64(len(s.columns[i].chars) * 16)
			s.columns[i].speed = 2 + rand.Float64()*6
			s.columns[i].chars = generateMatrixChars(20 + rand.IntN(15))
		}
	}
}

func (s *matrixState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	colWidth := w / len(s.columns)
	if colWidth < 1 {
		colWidth = 1
	}

	for i, col := range s.columns {
		x := i * colWidth
		charHeight := 16

		for j := range col.chars {
			cy := int(col.y) - j*charHeight
			if cy >= h {
				continue
			}
			if cy+charHeight <= 0 {
				break // this and all subsequent trail chars are above viewport
			}

			// Fade out older characters
			brightness := uint8(255 - min(j*12, 230))

			// Render the character glyph
			glyph := matrixGlyph(col.chars[j])
			glyphW := 8
			glyphH := 12
			// Center glyph within the column cell
			offsetX := (colWidth - glyphW) / 2
			offsetY := (charHeight - glyphH) / 2

			for gy := range glyphH {
				for gx := range glyphW {
					if glyph[gy]&(1<<(7-gx)) == 0 {
						continue
					}
					px := x + offsetX + gx
					py := cy + offsetY + gy
					if px >= 0 && px < w && py >= 0 && py < h {
						offset := (py*w + px) * rgbaChannels
						pixels[offset] = 0                // R
						pixels[offset+1] = brightness     // G
						pixels[offset+2] = brightness / 4 // B
						pixels[offset+3] = brightness     // A
					}
				}
			}
		}
	}

	return pixels
}

func generateMatrixChars(length int) []rune {
	chars := []rune("アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモ0123456789")
	result := make([]rune, length)
	for i := range result {
		result[i] = chars[rand.IntN(len(chars))]
	}
	return result
}

// matrixGlyphs contains 8x12 bitmap glyphs for the Matrix rain characters.
// Each glyph is 12 rows of uint8 where bits represent pixels (MSB = leftmost).
var matrixGlyphs = map[rune][12]uint8{
	// Katakana
	'ア': {0x00, 0x7E, 0x02, 0x02, 0x3E, 0x20, 0x20, 0x10, 0x08, 0x04, 0x02, 0x00},
	'イ': {0x00, 0x02, 0x04, 0x08, 0x18, 0x28, 0x08, 0x08, 0x08, 0x08, 0x08, 0x00},
	'ウ': {0x00, 0x10, 0x7E, 0x42, 0x42, 0x42, 0x22, 0x22, 0x14, 0x08, 0x04, 0x00},
	'エ': {0x00, 0x7E, 0x00, 0x00, 0x18, 0x18, 0x18, 0x18, 0x00, 0x00, 0x7E, 0x00},
	'オ': {0x00, 0x08, 0x7E, 0x08, 0x08, 0x1C, 0x2A, 0x4A, 0x08, 0x08, 0x08, 0x00},
	'カ': {0x00, 0x10, 0x7E, 0x12, 0x12, 0x22, 0x22, 0x22, 0x42, 0x42, 0x02, 0x00},
	'キ': {0x00, 0x08, 0x08, 0x7E, 0x08, 0x08, 0x7E, 0x08, 0x08, 0x08, 0x08, 0x00},
	'ク': {0x00, 0x20, 0x3E, 0x22, 0x22, 0x22, 0x14, 0x14, 0x08, 0x04, 0x02, 0x00},
	'ケ': {0x00, 0x20, 0x3E, 0x22, 0x04, 0x04, 0x08, 0x08, 0x10, 0x20, 0x40, 0x00},
	'コ': {0x00, 0x7E, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x7E, 0x00, 0x00},
	'サ': {0x00, 0x24, 0x24, 0x24, 0x7E, 0x24, 0x24, 0x04, 0x04, 0x08, 0x10, 0x00},
	'シ': {0x00, 0x42, 0x22, 0x02, 0x02, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x00},
	'ス': {0x00, 0x7E, 0x02, 0x04, 0x08, 0x0C, 0x14, 0x22, 0x42, 0x02, 0x02, 0x00},
	'セ': {0x00, 0x10, 0x10, 0x7E, 0x12, 0x12, 0x12, 0x22, 0x22, 0x42, 0x02, 0x00},
	'ソ': {0x00, 0x42, 0x22, 0x22, 0x12, 0x04, 0x04, 0x08, 0x08, 0x10, 0x20, 0x00},
	'タ': {0x00, 0x10, 0x3E, 0x22, 0x22, 0x3E, 0x10, 0x08, 0x04, 0x02, 0x02, 0x00},
	'チ': {0x00, 0x08, 0x7E, 0x08, 0x08, 0x3C, 0x08, 0x08, 0x10, 0x20, 0x40, 0x00},
	'ツ': {0x00, 0x44, 0x24, 0x24, 0x04, 0x04, 0x04, 0x08, 0x08, 0x10, 0x20, 0x00},
	'テ': {0x00, 0x7E, 0x00, 0x08, 0x08, 0x08, 0x08, 0x10, 0x10, 0x20, 0x40, 0x00},
	'ト': {0x00, 0x20, 0x20, 0x20, 0x38, 0x24, 0x20, 0x20, 0x20, 0x20, 0x20, 0x00},
	'ナ': {0x00, 0x08, 0x08, 0x7E, 0x08, 0x08, 0x10, 0x10, 0x20, 0x20, 0x40, 0x00},
	'ニ': {0x00, 0x00, 0x3C, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x7E, 0x00, 0x00},
	'ヌ': {0x00, 0x7E, 0x02, 0x04, 0x0C, 0x14, 0x24, 0x04, 0x02, 0x02, 0x02, 0x00},
	'ネ': {0x00, 0x08, 0x7E, 0x08, 0x08, 0x3E, 0x49, 0x08, 0x08, 0x08, 0x08, 0x00},
	'ノ': {0x00, 0x02, 0x02, 0x04, 0x04, 0x08, 0x08, 0x10, 0x10, 0x20, 0x40, 0x00},
	'ハ': {0x00, 0x00, 0x10, 0x10, 0x28, 0x28, 0x44, 0x44, 0x82, 0x02, 0x02, 0x00},
	'ヒ': {0x00, 0x20, 0x20, 0x20, 0x3E, 0x20, 0x20, 0x20, 0x20, 0x1E, 0x00, 0x00},
	'フ': {0x00, 0x7E, 0x42, 0x02, 0x02, 0x04, 0x04, 0x08, 0x10, 0x20, 0x40, 0x00},
	'ヘ': {0x00, 0x00, 0x00, 0x10, 0x28, 0x44, 0x82, 0x02, 0x00, 0x00, 0x00, 0x00},
	'ホ': {0x00, 0x08, 0x7E, 0x08, 0x08, 0x2A, 0x2A, 0x4A, 0x08, 0x08, 0x08, 0x00},
	'マ': {0x00, 0x7E, 0x02, 0x02, 0x04, 0x08, 0x10, 0x10, 0x08, 0x04, 0x02, 0x00},
	'ミ': {0x00, 0x00, 0x1E, 0x00, 0x00, 0x3C, 0x00, 0x00, 0x7E, 0x00, 0x00, 0x00},
	'ム': {0x00, 0x10, 0x10, 0x10, 0x18, 0x14, 0x14, 0x22, 0x42, 0x42, 0x7E, 0x00},
	'メ': {0x00, 0x02, 0x04, 0x08, 0x14, 0x24, 0x14, 0x08, 0x10, 0x20, 0x40, 0x00},
	'モ': {0x00, 0x7E, 0x08, 0x08, 0x7E, 0x08, 0x08, 0x08, 0x10, 0x20, 0x40, 0x00},
	// Digits
	'0': {0x00, 0x3C, 0x42, 0x46, 0x4A, 0x52, 0x62, 0x42, 0x42, 0x3C, 0x00, 0x00},
	'1': {0x00, 0x08, 0x18, 0x28, 0x08, 0x08, 0x08, 0x08, 0x08, 0x3E, 0x00, 0x00},
	'2': {0x00, 0x3C, 0x42, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x7E, 0x00, 0x00},
	'3': {0x00, 0x3C, 0x42, 0x02, 0x1C, 0x02, 0x02, 0x02, 0x42, 0x3C, 0x00, 0x00},
	'4': {0x00, 0x04, 0x0C, 0x14, 0x24, 0x44, 0x7E, 0x04, 0x04, 0x04, 0x00, 0x00},
	'5': {0x00, 0x7E, 0x40, 0x40, 0x7C, 0x02, 0x02, 0x02, 0x42, 0x3C, 0x00, 0x00},
	'6': {0x00, 0x1C, 0x20, 0x40, 0x7C, 0x42, 0x42, 0x42, 0x42, 0x3C, 0x00, 0x00},
	'7': {0x00, 0x7E, 0x02, 0x04, 0x04, 0x08, 0x08, 0x10, 0x10, 0x10, 0x00, 0x00},
	'8': {0x00, 0x3C, 0x42, 0x42, 0x3C, 0x42, 0x42, 0x42, 0x42, 0x3C, 0x00, 0x00},
	'9': {0x00, 0x3C, 0x42, 0x42, 0x42, 0x3E, 0x02, 0x02, 0x04, 0x38, 0x00, 0x00},
}

// matrixGlyph returns the 8x12 bitmap for a rune, falling back to a random glyph.
func matrixGlyph(r rune) [12]uint8 {
	if g, ok := matrixGlyphs[r]; ok {
		return g
	}
	// Fallback: generate a pseudo-random glyph pattern from the rune value
	var g [12]uint8
	seed := uint64(r)
	for i := range g {
		seed = seed*6364136223846793005 + 1
		g[i] = uint8((seed >> 33) & 0x7E) // keep within 8px width, avoid edges
	}
	return g
}

// --- Fire Effect ---

type fireState struct {
	grid   [][]uint8
	width  int
	height int
}

func (p *shenanigans) startFire(pc linkquisition.PickerCanvas) {
	state := &fireState{
		width:  fireWidth,
		height: fireHeight,
	}
	state.init()

	pc.AddRasterOverlay(0.4, func(w, h int) []uint8 {
		return state.render(w, h)
	})

	go func() {
		ticker := time.NewTicker(fireFrameInterval)
		defer ticker.Stop()

		for range ticker.C {
			if p.stopped.Load() {
				return
			}
			state.update()
			state.update() // double-step for faster movement
			pc.ScheduleRefresh()
		}
	}()
}

func (s *fireState) init() {
	s.grid = make([][]uint8, s.height)
	for i := range s.grid {
		s.grid[i] = make([]uint8, s.width)
	}
}

func (s *fireState) update() {
	// Set the bottom two rows to random hot values (wider fuel source)
	for x := range s.width {
		s.grid[s.height-1][x] = uint8(180 + rand.IntN(76))
		s.grid[s.height-2][x] = uint8(150 + rand.IntN(106))
	}

	// Propagate fire upward with averaging and cooling
	for y := range s.height - 2 {
		for x := range s.width {
			// Sample a wider neighborhood for smoother spread
			l2 := max(x-2, 0)
			l1 := max(x-1, 0)
			r1 := min(x+1, s.width-1)
			r2 := min(x+2, s.width-1)

			// Weighted average: center-heavy for more vertical flames
			sum := int(s.grid[y+1][l1]) +
				int(s.grid[y+1][x])*3 +
				int(s.grid[y+1][r1]) +
				int(s.grid[y+2][l2]) +
				int(s.grid[y+2][x])*2 +
				int(s.grid[y+2][r2])

			avg := sum / 9

			// Cooling increases toward the top for natural fadeout
			coolBase := 2 + (s.height-y)/15
			cooling := rand.IntN(coolBase + 1)
			val := avg - cooling

			if val < 0 {
				val = 0
			}
			s.grid[y][x] = uint8(val)
		}
	}
}

func (s *fireState) render(targetW, targetH int) []uint8 {
	if targetW == 0 || targetH == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, targetW*targetH*rgbaChannels)

	// Fire occupies the bottom 2/3 of the window
	fireStartY := targetH / 3

	fireH := targetH - fireStartY
	if fireH <= 0 {
		return pixels
	}

	for py := fireStartY; py < targetH; py++ {
		for px := 0; px < targetW; px++ {
			// Map to fire grid with bilinear interpolation
			fy := float64(py-fireStartY) * float64(s.height-1) / float64(fireH-1)
			fx := float64(px) * float64(s.width-1) / float64(targetW-1)

			val := s.sampleBilinear(fx, fy)

			r, g, b, a := fireColor(val)
			offset := (py*targetW + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = a
		}
	}

	return pixels
}

// sampleBilinear performs bilinear interpolation on the fire grid for smooth scaling.
func (s *fireState) sampleBilinear(fx, fy float64) uint8 {
	x0 := int(fx)
	y0 := int(fy)
	x1 := min(x0+1, s.width-1)
	y1 := min(y0+1, s.height-1)

	xFrac := fx - float64(x0)
	yFrac := fy - float64(y0)

	// Four corners
	v00 := float64(s.grid[y0][x0])
	v10 := float64(s.grid[y0][x1])
	v01 := float64(s.grid[y1][x0])
	v11 := float64(s.grid[y1][x1])

	// Interpolate
	top := v00*(1-xFrac) + v10*xFrac
	bottom := v01*(1-xFrac) + v11*xFrac
	val := top*(1-yFrac) + bottom*yFrac

	return uint8(min(max(int(val), 0), 255))
}

// fireColor maps a heat value (0-255) to a realistic fire palette.
// Gradient: transparent → dark red/brown → red → orange → gold → pale yellow
func fireColor(val uint8) (r, g, b, a uint8) {
	if val < 24 {
		return 0, 0, 0, 0
	}

	// Normalize to 0.0-1.0 range (24-255 → 0.0-1.0)
	t := float64(val-24) / 231.0

	// Piecewise palette for realistic fire
	switch {
	case t < 0.2:
		// Black → dark maroon/brown
		p := t / 0.2
		r = uint8(p * 80)
		g = uint8(p * 10)
		a = uint8(p * 180)
		return r, g, 0, a
	case t < 0.45:
		// Dark maroon → bright red
		p := (t - 0.2) / 0.25
		r = uint8(80 + p*175)
		g = uint8(10 + p*20)
		a = uint8(180 + p*75)
		return r, g, 0, a
	case t < 0.7:
		// Red → orange
		p := (t - 0.45) / 0.25
		r = 255
		g = uint8(30 + p*170)
		a = 255
		return r, g, 0, a
	case t < 0.9:
		// Orange → golden yellow
		p := (t - 0.7) / 0.2
		r = 255
		g = uint8(200 + p*55)
		b = uint8(p * 30)
		a = 255
		return r, g, b, a
	default:
		// Golden yellow → pale yellow/white tips
		p := (t - 0.9) / 0.1
		r = 255
		g = 255
		b = uint8(30 + p*120)
		a = uint8(255 - p*80)
		return r, g, b, a
	}
}

// --- Snow Effect ---

const (
	snowFlakeCount = 150
	snowMaxSize    = 4
)

type snowflake struct {
	x, y   float64
	size   float64
	speed  float64
	drift  float64
	wobble float64
	phase  float64
}

type snowState struct {
	flakes      []snowflake
	width       int
	height      int
	initialized bool
}

func (p *shenanigans) startSnow(pc linkquisition.PickerCanvas) {
	state := &snowState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}

	pc.AddRasterOverlay(0.3, func(w, h int) []uint8 {
		if !state.initialized || w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.init()
			state.initialized = true
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
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

func (s *snowState) init() {
	s.flakes = make([]snowflake, snowFlakeCount)
	for i := range s.flakes {
		// Start most flakes above the window so they drift in gradually.
		// A few start on-screen so it's not completely empty at first.
		onScreen := i < snowFlakeCount/5
		s.flakes[i] = s.newFlake(onScreen)
	}
}

func (s *snowState) newFlake(onScreen bool) snowflake {
	// Default: spawn above the viewport at varying distances
	y := -(rand.Float64() * float64(s.height))
	if onScreen {
		y = rand.Float64() * float64(s.height)
	}

	return snowflake{
		x:      rand.Float64() * float64(s.width),
		y:      y,
		size:   1 + rand.Float64()*float64(snowMaxSize-1),
		speed:  0.5 + rand.Float64()*2.0,
		drift:  (rand.Float64() - 0.5) * 0.3,
		wobble: 0.3 + rand.Float64()*0.7,
		phase:  rand.Float64() * 6.28,
	}
}

func (s *snowState) update() {
	for i := range s.flakes {
		f := &s.flakes[i]
		f.y += f.speed
		f.phase += 0.05

		// Gentle sine-wave wobble for horizontal drift
		f.x += f.drift + f.wobble*sinApprox(f.phase)*0.3

		// Respawn at top if fallen below window
		if f.y > float64(s.height)+10 {
			*f = s.newFlake(false)
		}

		// Wrap horizontally
		if f.x < 0 {
			f.x += float64(s.width)
		} else if f.x >= float64(s.width) {
			f.x -= float64(s.width)
		}
	}
}

func (s *snowState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	for _, f := range s.flakes {
		s.drawFlake(pixels, f)
	}

	return pixels
}

func (s *snowState) drawFlake(pixels []uint8, f snowflake) {
	w, h := s.width, s.height
	cx, cy := int(f.x), int(f.y)
	radius := int(f.size)

	// Draw a soft circle with alpha falloff
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			px := cx + dx
			py := cy + dy

			if px < 0 || px >= w || py < 0 || py >= h {
				continue
			}

			// Distance from center (0.0 to 1.0+)
			dist := float64(dx*dx+dy*dy) / float64(radius*radius+1)
			if dist > 1.0 {
				continue
			}

			// Soft edge: alpha falls off near the border
			alpha := uint8((1.0 - dist*dist) * 220)

			offset := (py*w + px) * rgbaChannels
			// Blend: white snowflake with alpha
			existing := pixels[offset+3]
			if alpha > existing {
				pixels[offset] = 255   // R
				pixels[offset+1] = 255 // G
				pixels[offset+2] = 255 // B
				pixels[offset+3] = alpha
			}
		}
	}
}

// sinApprox is a fast sine approximation (Bhaskara I's formula) avoiding math import.
func sinApprox(x float64) float64 {
	// Normalize to [0, 2π)
	const twoPi = 6.283185307
	const pi = 3.141592654

	for x < 0 {
		x += twoPi
	}
	for x >= twoPi {
		x -= twoPi
	}

	// Map to [0, π] with sign
	sign := 1.0
	if x > pi {
		x -= pi
		sign = -1.0
	}

	// Bhaskara I's approximation: sin(x) ≈ 16x(π-x) / (5π²-4x(π-x))
	num := 16 * x * (pi - x)
	den := 5*pi*pi - 4*x*(pi-x)
	return sign * num / den
}

// --- Plasma Effect ---

type plasmaState struct {
	time float64
}

func (p *shenanigans) startPlasma(pc linkquisition.PickerCanvas) {
	state := &plasmaState{}

	pc.AddRasterOverlay(0.5, func(w, h int) []uint8 {
		return state.render(w, h)
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
		defer ticker.Stop()

		for range ticker.C {
			if p.stopped.Load() {
				return
			}
			state.time += 0.06
			pc.ScheduleRefresh()
		}
	}()
}

func (s *plasmaState) render(w, h int) []uint8 {
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	t := s.time

	for py := 0; py < h; py++ {
		fy := float64(py) / float64(h)
		for px := 0; px < w; px++ {
			fx := float64(px) / float64(w)

			// Overlapping sine waves at different frequencies and phases
			v1 := sinApprox((fx*4 + t) * 3.14159)
			v2 := sinApprox((fy*4 + t*0.7) * 3.14159)
			v3 := sinApprox(((fx+fy)*3 + t*1.3) * 3.14159)
			v4 := sinApprox(((fx-fy)*2 + t*0.5) * 3.14159)
			v5 := sinApprox(((fx*fx+fy*fy)*4 - t*0.9) * 3.14159)

			// Combine waves (result in -1 to 1 range, normalize to 0-1)
			val := (v1 + v2 + v3 + v4 + v5) / 5.0
			val = (val + 1.0) / 2.0

			// Map to color using three phase-shifted sine waves for RGB
			r := uint8(sinNorm(val*3.14159*2+t*0.3) * 255)
			g := uint8(sinNorm(val*3.14159*2+t*0.3+2.09) * 255)
			b := uint8(sinNorm(val*3.14159*2+t*0.3+4.19) * 255)

			offset := (py*w + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = 200
		}
	}

	return pixels
}

// --- Starfield Effect ---

const starCount = 200

type star struct {
	x, y, z float64
}

type starfieldState struct {
	stars  []star
	width  int
	height int
}

func (p *shenanigans) startStarfield(pc linkquisition.PickerCanvas) {
	state := &starfieldState{}

	pc.AddRasterOverlay(0.2, func(w, h int) []uint8 {
		if !state.isInitialized() || w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.init()
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
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

func (s *starfieldState) isInitialized() bool {
	return len(s.stars) > 0
}

func (s *starfieldState) init() {
	s.stars = make([]star, starCount)
	for i := range s.stars {
		s.stars[i] = s.newStar(true)
	}
}

func (s *starfieldState) newStar(randomDepth bool) star {
	z := 0.01 + rand.Float64()*0.99
	if randomDepth {
		z = 0.1 + rand.Float64()*0.9
	}

	return star{
		x: (rand.Float64() - 0.5) * 2.0,
		y: (rand.Float64() - 0.5) * 2.0,
		z: z,
	}
}

func (s *starfieldState) update() {
	for i := range s.stars {
		s.stars[i].z -= 0.015

		// Respawn stars that have passed the viewer
		if s.stars[i].z <= 0.001 {
			s.stars[i] = s.newStar(false)
			s.stars[i].z = 0.9 + rand.Float64()*0.1
		}
	}
}

func (s *starfieldState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	cx := float64(w) / 2.0
	cy := float64(h) / 2.0

	for _, st := range s.stars {
		// Perspective projection
		screenX := int(cx + (st.x/st.z)*cx)
		screenY := int(cy + (st.y/st.z)*cy)

		if screenX < 0 || screenX >= w || screenY < 0 || screenY >= h {
			continue
		}

		// Size and brightness increase as stars get closer (z → 0)
		brightness := uint8(min(int((1.0-st.z)*255), 255))
		size := int(1 + (1.0-st.z)*3)

		s.drawStar(pixels, screenX, screenY, size, brightness)
	}

	return pixels
}

func (s *starfieldState) drawStar(pixels []uint8, screenX, screenY, size int, brightness uint8) {
	w, h := s.width, s.height
	for dy := -size; dy <= size; dy++ {
		for dx := -size; dx <= size; dx++ {
			px := screenX + dx
			py := screenY + dy

			if px < 0 || px >= w || py < 0 || py >= h {
				continue
			}

			dist := dx*dx + dy*dy
			maxDist := size * size
			if dist > maxDist {
				continue
			}

			// Alpha falls off with distance from center
			falloff := 1.0 - float64(dist)/float64(maxDist+1)
			alpha := uint8(float64(brightness) * falloff)

			offset := (py*w + px) * rgbaChannels
			if alpha > pixels[offset+3] {
				pixels[offset] = brightness
				pixels[offset+1] = brightness
				pixels[offset+2] = 255 // slight blue tint
				pixels[offset+3] = alpha
			}
		}
	}
}

// --- Aurora Effect ---

const auroraLayers = 4

type auroraState struct {
	time float64
}

func (p *shenanigans) startAurora(pc linkquisition.PickerCanvas) {
	state := &auroraState{}

	pc.AddRasterOverlay(0.4, func(w, h int) []uint8 {
		return state.render(w, h)
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
		defer ticker.Stop()

		for range ticker.C {
			if p.stopped.Load() {
				return
			}
			state.time += 0.03
			pc.ScheduleRefresh()
		}
	}()
}

func (s *auroraState) render(w, h int) []uint8 {
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	t := s.time

	// Aurora occupies the top 2/3 of the window
	auroraEndY := h * 2 / 3

	for py := 0; py < auroraEndY; py++ {
		// Vertical position normalized (0 at top, 1 at aurora bottom)
		fy := float64(py) / float64(auroraEndY)

		for px := 0; px < w; px++ {
			fx := float64(px) / float64(w)

			// Combine multiple curtain layers
			var intensity float64
			for layer := range auroraLayers {
				fl := float64(layer)
				// Each layer has different frequency, speed, and phase
				wave := sinApprox((fx*(3+fl) + t*(0.4+fl*0.15) + fl*1.7) * 3.14159)
				wave2 := sinApprox((fx*(2+fl*0.7) - t*(0.3+fl*0.1) + fl*2.3) * 3.14159)

				// Curtain shape: thin band that undulates
				curtainCenter := 0.2 + 0.15*fl + 0.1*(wave*0.5+0.5)
				curtainWidth := 0.08 + 0.04*wave2

				// Gaussian-like falloff from the curtain center
				dist := (fy - curtainCenter) / curtainWidth
				layerIntensity := fastExp(-dist * dist * 0.5)

				intensity += layerIntensity * (0.6 + 0.4/(fl+1))
			}

			if intensity < 0.01 {
				continue
			}
			if intensity > 1.0 {
				intensity = 1.0
			}

			// Aurora color: shift from green to purple/blue based on position and time
			colorPhase := fx*0.5 + fy*0.3 + t*0.1
			r, g, b := auroraColor(colorPhase, intensity)

			alpha := uint8(intensity * 180)

			offset := (py*w + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = alpha
		}
	}

	return pixels
}

// auroraColor maps a phase and intensity to northern lights colors.
// Shifts between green, teal, blue, and purple.
func auroraColor(phase, intensity float64) (r, g, b uint8) {
	// Cycle through aurora palette
	p := sinApprox(phase * 3.14159 * 2)
	p = (p + 1.0) / 2.0 // normalize to 0-1

	// Blend between green-dominant and purple-dominant
	var rf, gf, bf float64
	switch {
	case p < 0.33:
		// Green to teal
		t := p / 0.33
		rf = 0.1 * t
		gf = 0.8 + 0.2*t
		bf = 0.2 + 0.5*t
	case p < 0.66:
		// Teal to purple
		t := (p - 0.33) / 0.33
		rf = 0.1 + 0.5*t
		gf = 1.0 - 0.6*t
		bf = 0.7 + 0.3*t
	default:
		// Purple back to green
		t := (p - 0.66) / 0.34
		rf = 0.6 - 0.5*t
		gf = 0.4 + 0.4*t
		bf = 1.0 - 0.8*t
	}

	r = uint8(rf * intensity * 255)
	g = uint8(gf * intensity * 255)
	b = uint8(bf * intensity * 255)
	return r, g, b
}

// fastExp approximates e^x for negative x values (used for Gaussian falloff).
func fastExp(x float64) float64 {
	if x < -6 {
		return 0
	}
	// Padé approximation: (1 + x/n)^n for small |x|
	// Using n=8 for reasonable accuracy
	t := 1.0 + x/8.0
	t *= t // ^2
	t *= t // ^4
	t *= t // ^8
	if t < 0 {
		return 0
	}
	return t
}

// --- Glitch Effect ---

type glitchState struct {
	frame      int
	slices     []glitchSlice
	burstTimer int
	isBursting bool
}

type glitchSlice struct {
	y, height int
	offsetX   int
	channel   int // 0=R shift, 1=G shift, 2=B shift
	alpha     uint8
}

func (p *shenanigans) startGlitch(pc linkquisition.PickerCanvas) {
	state := &glitchState{}

	pc.AddRasterOverlay(0.0, func(w, h int) []uint8 {
		return state.render(w, h)
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
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

func (s *glitchState) update() {
	s.frame++
	s.burstTimer--

	// Trigger glitch bursts periodically
	if s.burstTimer <= 0 {
		if s.isBursting {
			// End burst
			s.isBursting = false
			s.slices = nil
			s.burstTimer = 20 + rand.IntN(40)
		} else {
			// Start burst
			s.isBursting = true
			s.burstTimer = 3 + rand.IntN(8)
			s.generateSlices()
		}
	} else if s.isBursting && s.frame%2 == 0 {
		// Regenerate slices during burst for flicker
		s.generateSlices()
	}
}

func (s *glitchState) generateSlices() {
	count := 3 + rand.IntN(8)
	s.slices = make([]glitchSlice, count)

	for i := range s.slices {
		s.slices[i] = glitchSlice{
			y:       rand.IntN(400),
			height:  2 + rand.IntN(20),
			offsetX: -30 + rand.IntN(60),
			channel: rand.IntN(3),
			alpha:   uint8(80 + rand.IntN(176)),
		}
	}
}

func (s *glitchState) render(w, h int) []uint8 {
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	if !s.isBursting {
		return pixels
	}

	// Draw glitch slices
	for _, slice := range s.slices {
		s.drawSlice(pixels, slice, w, h)
	}

	// Add random static noise during bursts
	s.addNoise(pixels, w, h)

	return pixels
}

func (s *glitchState) drawSlice(pixels []uint8, slice glitchSlice, w, h int) {
	sy := slice.y * h / 400
	sh := slice.height * h / 400

	for py := sy; py < sy+sh && py < h; py++ {
		if py < 0 {
			continue
		}
		for px := 0; px < w; px++ {
			offset := (py*w + px) * rgbaChannels

			// RGB channel separation effect
			switch slice.channel {
			case 0: // Red shift
				srcX := px - slice.offsetX
				if srcX >= 0 && srcX < w {
					pixels[offset] = slice.alpha
					pixels[offset+3] = slice.alpha / 2
				}
			case 1: // Green shift
				srcX := px + slice.offsetX
				if srcX >= 0 && srcX < w {
					pixels[offset+1] = slice.alpha
					pixels[offset+3] = slice.alpha / 2
				}
			default: // Blue/cyan shift
				pixels[offset+2] = slice.alpha
				pixels[offset+1] = slice.alpha / 3
				pixels[offset+3] = slice.alpha / 2
			}
		}
	}
}

func (s *glitchState) addNoise(pixels []uint8, w, h int) {
	noiseCount := w * h / 40
	for range noiseCount {
		px := rand.IntN(w)
		py := rand.IntN(h)
		offset := (py*w + px) * rgbaChannels
		v := uint8(rand.IntN(256))
		pixels[offset] = v
		pixels[offset+1] = v
		pixels[offset+2] = v
		pixels[offset+3] = uint8(rand.IntN(100))
	}
}

// --- Pride Effect ---

type prideState struct {
	time float64
}

// Pride flag colors (6-stripe rainbow)
var prideColors = [][3]uint8{
	{228, 3, 3},   // Red
	{255, 140, 0}, // Orange
	{255, 237, 0}, // Yellow
	{0, 128, 38},  // Green
	{0, 77, 255},  // Blue
	{117, 7, 135}, // Purple
}

func (p *shenanigans) startPride(pc linkquisition.PickerCanvas) {
	state := &prideState{}

	pc.AddRasterOverlay(0.45, func(w, h int) []uint8 {
		return state.render(w, h)
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
		defer ticker.Stop()

		for range ticker.C {
			if p.stopped.Load() {
				return
			}
			state.time += 0.04
			pc.ScheduleRefresh()
		}
	}()
}

func (s *prideState) render(w, h int) []uint8 {
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	t := s.time
	stripeCount := float64(len(prideColors))

	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			fx := float64(px) / float64(w)
			fy := float64(py) / float64(h)

			// Flag wave: pinned on the left edge, wave amplitude increases to the right
			// This simulates fabric attached to a pole on the left side
			amplitude := fx * fx * 0.12
			wave := sinApprox((fx*2.0-t*1.5)*3.14159) * amplitude
			wave2 := sinApprox((fx*3.0-t*2.0)*3.14159) * amplitude * 0.5

			// Determine which stripe this pixel belongs to (with wave offset)
			stripePos := (fy + wave + wave2) * stripeCount
			stripeIdx := int(stripePos)

			if stripeIdx < 0 {
				stripeIdx = 0
			}
			if stripeIdx >= len(prideColors) {
				stripeIdx = len(prideColors) - 1
			}

			// Smooth blending between stripes
			blend := stripePos - float64(stripeIdx)
			nextIdx := stripeIdx + 1
			if nextIdx >= len(prideColors) {
				nextIdx = len(prideColors) - 1
			}

			c1 := prideColors[stripeIdx]
			c2 := prideColors[nextIdx]

			// Smooth interpolation (smoothstep-like)
			blend = blend * blend * (3 - 2*blend)

			r := uint8(float64(c1[0])*(1-blend) + float64(c2[0])*blend)
			g := uint8(float64(c1[1])*(1-blend) + float64(c2[1])*blend)
			b := uint8(float64(c1[2])*(1-blend) + float64(c2[2])*blend)

			// Subtle shading to simulate fabric folds (stronger toward free edge)
			foldDepth := fx * 0.2
			shade := 1.0 - foldDepth + foldDepth*sinApprox((fx*3-t*1.5)*3.14159)
			r = uint8(float64(r) * shade)
			g = uint8(float64(g) * shade)
			b = uint8(float64(b) * shade)

			offset := (py*w + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = 200
		}
	}

	return pixels
}

// --- Football Effect ---

type footballState struct {
	time   float64
	width  int
	height int
}

func (p *shenanigans) startFootball(pc linkquisition.PickerCanvas) {
	state := &footballState{}

	pc.AddRasterOverlay(0.4, func(w, h int) []uint8 {
		state.width = w
		state.height = h
		return state.render()
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
		defer ticker.Stop()

		for range ticker.C {
			if p.stopped.Load() {
				return
			}
			state.time += 0.03
			pc.ScheduleRefresh()
		}
	}()
}

func (s *footballState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	t := s.time

	for py := 0; py < h; py++ {
		fy := float64(py) / float64(h)
		for px := 0; px < w; px++ {
			fx := float64(px) / float64(w)

			// Green pitch base with subtle stripe pattern (mowed grass look)
			stripeWidth := 0.08
			stripe := int(fx/stripeWidth) % 2
			var gr, gg, gb uint8
			if stripe == 0 {
				gr, gg, gb = 34, 139, 34
			} else {
				gr, gg, gb = 30, 124, 30
			}

			// Animated element: a "spotlight" sweeping across the pitch
			spotX := 0.5 + 0.4*sinApprox(t*1.5)
			spotY := 0.5 + 0.3*sinApprox(t*1.1+1.0)
			spotDist := (fx-spotX)*(fx-spotX) + (fy-spotY)*(fy-spotY)
			spotLight := fastExp(-spotDist*15) * 0.3

			var r, g, b uint8
			if isPitchLine(fx, fy) {
				r, g, b = 255, 255, 255
			} else {
				r = uint8(min(int(float64(gr)*(1+spotLight)), 255))
				g = uint8(min(int(float64(gg)*(1+spotLight)), 255))
				b = uint8(min(int(float64(gb)*(1+spotLight)), 255))
			}

			offset := (py*w + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = 180
		}
	}

	return pixels
}

// isPitchLine determines if a normalized coordinate is on a white pitch marking.
func isPitchLine(fx, fy float64) bool {
	return isPitchCenterMarkings(fx, fy) ||
		isPitchBoundary(fx, fy) ||
		isPitchPenaltyArea(fx, fy)
}

func isPitchCenterMarkings(fx, fy float64) bool {
	// Center line (vertical)
	if absF(fx-0.5) < 0.004 {
		return true
	}

	// Center circle
	cx, cy := 0.5, 0.5
	dist := (fx-cx)*(fx-cx)*1.5 + (fy-cy)*(fy-cy)
	if absF(dist-0.04) < 0.003 {
		return true
	}

	// Center dot
	return dist < 0.002
}

func isPitchBoundary(fx, fy float64) bool {
	if fx < 0.02 || fx > 0.98 || fy < 0.03 || fy > 0.97 {
		if fx > 0.015 && fx < 0.985 && fy > 0.025 && fy < 0.975 {
			return true
		}
	}
	return false
}

func isPitchPenaltyArea(fx, fy float64) bool {
	penaltyW := 0.15
	penaltyH := 0.35
	penaltyTop := 0.5 - penaltyH
	penaltyBot := 0.5 + penaltyH

	// Left penalty area
	if fx < penaltyW && fy > penaltyTop && fy < penaltyBot {
		if absF(fx-penaltyW) < 0.004 || absF(fy-penaltyTop) < 0.005 || absF(fy-penaltyBot) < 0.005 {
			return true
		}
	}

	// Right penalty area
	if fx > (1-penaltyW) && fy > penaltyTop && fy < penaltyBot {
		if absF(fx-(1-penaltyW)) < 0.004 || absF(fy-penaltyTop) < 0.005 || absF(fy-penaltyBot) < 0.005 {
			return true
		}
	}

	return false
}

// --- Fireworks Effect ---

const (
	fireworksMaxRockets   = 5
	fireworksParticles    = 60
	fireworksLaunchChance = 8 // percent chance per frame to launch a new rocket
)

type fireworksParticle struct {
	x, y    float64
	vx, vy  float64
	life    float64
	r, g, b uint8
}

type fireworksRocket struct {
	particles []fireworksParticle
	exploded  bool
	// Pre-explosion rocket position
	x, y    float64
	vy      float64
	targetY float64
	color   [3]uint8
}

type fireworksState struct {
	rockets []fireworksRocket
	width   int
	height  int
}

var fireworksColors = [][3]uint8{
	{255, 200, 50},  // Gold
	{255, 80, 80},   // Red
	{80, 150, 255},  // Blue
	{80, 255, 80},   // Green
	{200, 100, 255}, // Purple
	{255, 150, 200}, // Pink
	{255, 255, 255}, // White
}

func (p *shenanigans) startFireworks(pc linkquisition.PickerCanvas) {
	state := &fireworksState{}

	pc.AddRasterOverlay(0.2, func(w, h int) []uint8 {
		state.width = w
		state.height = h
		return state.render()
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
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

func (s *fireworksState) update() {
	if s.width == 0 || s.height == 0 {
		return
	}

	// Chance to launch a new rocket
	if len(s.rockets) < fireworksMaxRockets && rand.IntN(100) < fireworksLaunchChance {
		color := fireworksColors[rand.IntN(len(fireworksColors))]
		s.rockets = append(s.rockets, fireworksRocket{
			x:       0.2 + rand.Float64()*0.6,
			y:       1.0,
			vy:      -0.025 - rand.Float64()*0.015,
			targetY: 0.15 + rand.Float64()*0.35,
			color:   color,
		})
	}

	// Update rockets
	alive := s.rockets[:0]
	for i := range s.rockets {
		r := &s.rockets[i]

		if !r.exploded {
			r.y += r.vy
			// Explode when reaching target height
			if r.y <= r.targetY {
				r.exploded = true
				r.particles = make([]fireworksParticle, fireworksParticles)
				for j := range r.particles {
					angle := rand.Float64() * 6.283
					speed := 0.005 + rand.Float64()*0.015
					r.particles[j] = fireworksParticle{
						x:    r.x,
						y:    r.y,
						vx:   sinApprox(angle) * speed,
						vy:   sinApprox(angle+1.5708) * speed,
						life: 1.0,
						r:    r.color[0],
						g:    r.color[1],
						b:    r.color[2],
					}
				}
			}
		} else {
			// Update particles
			allDead := true
			for j := range r.particles {
				p := &r.particles[j]
				if p.life <= 0 {
					continue
				}
				p.x += p.vx
				p.y += p.vy
				p.vy += 0.0004 // gravity
				p.vx *= 0.98   // drag
				p.vy *= 0.98
				p.life -= 0.015
				if p.life > 0 {
					allDead = false
				}
			}
			if allDead {
				continue // don't keep this rocket
			}
		}
		alive = append(alive, *r)
	}
	s.rockets = alive
}

func (s *fireworksState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	for _, rocket := range s.rockets {
		if !rocket.exploded {
			// Draw rising rocket as a small bright dot with trail
			px := int(rocket.x * float64(w))
			py := int(rocket.y * float64(h))
			s.drawDot(pixels, px, py, 2, 255, 220, 150, 255)
			// Trail
			s.drawDot(pixels, px, py+3, 1, 255, 150, 50, 150)
			s.drawDot(pixels, px, py+6, 1, 255, 100, 30, 80)
		} else {
			// Draw particles
			for _, p := range rocket.particles {
				if p.life <= 0 {
					continue
				}
				px := int(p.x * float64(w))
				py := int(p.y * float64(h))
				alpha := uint8(p.life * 255)
				size := 1
				if p.life > 0.7 {
					size = 2
				}
				s.drawDot(pixels, px, py, size, p.r, p.g, p.b, alpha)
			}
		}
	}

	return pixels
}

func (s *fireworksState) drawDot(pixels []uint8, cx, cy, radius int, r, g, b, a uint8) {
	w, h := s.width, s.height
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			px := cx + dx
			py := cy + dy
			if px < 0 || px >= w || py < 0 || py >= h {
				continue
			}
			if dx*dx+dy*dy > radius*radius {
				continue
			}
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

// absF returns the absolute value of a float64.
func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// sinNorm returns sinApprox mapped to 0.0-1.0 range.
func sinNorm(x float64) float64 {
	return (sinApprox(x) + 1.0) / 2.0
}

// --- Pong Effect ---

const (
	pongPaddleWidth  = 6
	pongPaddleHeight = 40
	pongBallSize     = 6
	pongPaddleMargin = 12
	pongPaddleSpeed  = 3.5
	pongBallAlpha    = 140
	pongPaddleAlpha  = 120
	pongNetAlpha     = 50
	pongNetDash      = 8
	pongNetGap       = 6
)

type pongState struct {
	width, height int

	// Ball
	ballX, ballY   float64
	ballVX, ballVY float64

	// Paddles (y is the center)
	leftY, rightY float64

	// Score
	leftScore, rightScore int
}

func (p *shenanigans) startPong(pc linkquisition.PickerCanvas) {
	state := &pongState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}
	state.resetBall()

	pc.AddRasterOverlay(0.4, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
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

func (s *pongState) resetBall() {
	s.ballX = float64(s.width) / 2
	s.ballY = float64(s.height) / 2
	s.leftY = float64(s.height) / 2
	s.rightY = float64(s.height) / 2

	// Random initial direction
	s.ballVX = 3.0 + rand.Float64()*2.0
	if rand.IntN(2) == 0 {
		s.ballVX = -s.ballVX
	}
	s.ballVY = (rand.Float64() - 0.5) * 4.0
}

func (s *pongState) update() { //nolint:gocyclo
	h := float64(s.height)
	w := float64(s.width)

	// Move ball
	s.ballX += s.ballVX
	s.ballY += s.ballVY

	// Bounce off top/bottom walls
	if s.ballY <= 0 {
		s.ballY = -s.ballY
		s.ballVY = -s.ballVY
	} else if s.ballY >= h-1 {
		s.ballY = 2*(h-1) - s.ballY
		s.ballVY = -s.ballVY
	}

	// AI for paddles — follow the ball with imperfect tracking
	s.leftY += clampPaddleMove(s.ballY-s.leftY, pongPaddleSpeed*0.85)
	s.rightY += clampPaddleMove(s.ballY-s.rightY, pongPaddleSpeed*0.9)

	// Clamp paddles within bounds
	halfPaddle := float64(pongPaddleHeight) / 2
	s.leftY = clampFloat(s.leftY, halfPaddle, h-halfPaddle)
	s.rightY = clampFloat(s.rightY, halfPaddle, h-halfPaddle)

	// Left paddle collision
	leftPaddleX := float64(pongPaddleMargin + pongPaddleWidth)
	if s.ballX <= leftPaddleX && s.ballVX < 0 {
		if s.ballY >= s.leftY-halfPaddle && s.ballY <= s.leftY+halfPaddle {
			s.ballX = leftPaddleX
			s.ballVX = -s.ballVX * (1.0 + rand.Float64()*0.1)
			// Add spin based on where ball hits the paddle
			offset := (s.ballY - s.leftY) / halfPaddle
			s.ballVY += offset * 1.5
		}
	}

	// Right paddle collision
	rightPaddleX := w - float64(pongPaddleMargin+pongPaddleWidth)
	if s.ballX >= rightPaddleX && s.ballVX > 0 {
		if s.ballY >= s.rightY-halfPaddle && s.ballY <= s.rightY+halfPaddle {
			s.ballX = rightPaddleX
			s.ballVX = -s.ballVX * (1.0 + rand.Float64()*0.1)
			offset := (s.ballY - s.rightY) / halfPaddle
			s.ballVY += offset * 1.5
		}
	}

	// Cap ball speed to prevent it from getting too fast
	maxSpeed := 8.0
	if s.ballVX > maxSpeed {
		s.ballVX = maxSpeed
	} else if s.ballVX < -maxSpeed {
		s.ballVX = -maxSpeed
	}
	if s.ballVY > maxSpeed {
		s.ballVY = maxSpeed
	} else if s.ballVY < -maxSpeed {
		s.ballVY = -maxSpeed
	}

	// Score — ball leaves the field
	if s.ballX < 0 {
		s.rightScore++
		s.resetBall()
	} else if s.ballX > w {
		s.leftScore++
		s.resetBall()
	}
}

func (s *pongState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw center net (dashed line)
	centerX := w / 2
	for y := 0; y < h; y++ {
		segment := y % (pongNetDash + pongNetGap)
		if segment < pongNetDash {
			s.setPixel(pixels, centerX, y, 255, 255, 255, pongNetAlpha)
		}
	}

	// Draw paddles
	halfPaddle := pongPaddleHeight / 2
	// Left paddle
	for dy := -halfPaddle; dy <= halfPaddle; dy++ {
		for dx := 0; dx < pongPaddleWidth; dx++ {
			px := pongPaddleMargin + dx
			py := int(s.leftY) + dy
			s.setPixel(pixels, px, py, 255, 255, 255, pongPaddleAlpha)
		}
	}
	// Right paddle
	for dy := -halfPaddle; dy <= halfPaddle; dy++ {
		for dx := 0; dx < pongPaddleWidth; dx++ {
			px := w - pongPaddleMargin - pongPaddleWidth + dx
			py := int(s.rightY) + dy
			s.setPixel(pixels, px, py, 255, 255, 255, pongPaddleAlpha)
		}
	}

	// Draw ball
	halfBall := pongBallSize / 2
	for dy := -halfBall; dy <= halfBall; dy++ {
		for dx := -halfBall; dx <= halfBall; dx++ {
			px := int(s.ballX) + dx
			py := int(s.ballY) + dy
			s.setPixel(pixels, px, py, 255, 255, 255, pongBallAlpha)
		}
	}

	// Draw score (simple dot-based digits)
	s.drawScore(pixels)

	return pixels
}

func (s *pongState) setPixel(pixels []uint8, x, y int, r, g, b, a uint8) { //nolint:unparam
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

func (s *pongState) drawScore(pixels []uint8) {
	// Simple score display near the top center
	scoreAlpha := uint8(70)
	y := 15
	// Left score — draw dots on the left of center
	cx := s.width/2 - 20
	for i := range s.leftScore {
		s.drawDot(pixels, cx-i*8, y, scoreAlpha)
	}
	// Right score — draw dots on the right of center
	cx = s.width/2 + 20
	for i := range s.rightScore {
		s.drawDot(pixels, cx+i*8, y, scoreAlpha)
	}
}

func (s *pongState) drawDot(pixels []uint8, cx, cy int, alpha uint8) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			s.setPixel(pixels, cx+dx, cy+dy, 255, 255, 255, alpha)
		}
	}
}

func clampPaddleMove(delta, maxMove float64) float64 {
	if delta > maxMove {
		return maxMove
	}
	if delta < -maxMove {
		return -maxMove
	}
	return delta
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// --- Game of Life Effect ---

const (
	lifeCellSize      = 4
	lifeAlpha         = 100
	lifeFadeAlpha     = 40
	lifeFrameInterval = 80 * time.Millisecond
	lifeInitDensity   = 0.35
)

type lifeState struct {
	width, height int
	cols, rows    int
	cells         []bool
	prev          []bool
	generation    int
	staleCount    int
}

func (p *shenanigans) startLife(pc linkquisition.PickerCanvas) {
	state := &lifeState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}
	state.cols = state.width / lifeCellSize
	state.rows = state.height / lifeCellSize
	state.randomize()

	pc.AddRasterOverlay(0.5, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.cols = w / lifeCellSize
			state.rows = h / lifeCellSize
			state.randomize()
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(lifeFrameInterval)
		defer ticker.Stop()

		for range ticker.C {
			if p.stopped.Load() {
				return
			}
			state.step()
			pc.ScheduleRefresh()
		}
	}()
}

func (s *lifeState) randomize() {
	size := s.cols * s.rows
	s.cells = make([]bool, size)
	s.prev = make([]bool, size)
	s.generation = 0
	s.staleCount = 0

	for i := range s.cells {
		s.cells[i] = rand.Float64() < lifeInitDensity
	}
}

func (s *lifeState) step() {
	size := s.cols * s.rows
	if size == 0 {
		return
	}

	next := make([]bool, size)
	for row := range s.rows {
		for col := range s.cols {
			neighbors := s.countNeighbors(row, col)
			idx := row*s.cols + col
			alive := s.cells[idx]

			if alive {
				next[idx] = neighbors == 2 || neighbors == 3
			} else {
				next[idx] = neighbors == 3
			}
		}
	}

	// Detect stale patterns (identical to previous generation = oscillator)
	if s.cellsEqual(next, s.prev) || s.cellsEqual(next, s.cells) {
		s.staleCount++
	} else {
		s.staleCount = 0
	}

	copy(s.prev, s.cells)
	s.cells = next
	s.generation++

	// Re-seed if the pattern has become static or died out
	if s.staleCount > 3 || s.countAlive() < size/20 {
		s.randomize()
	}
}

func (s *lifeState) countNeighbors(row, col int) int {
	count := 0
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			if dr == 0 && dc == 0 {
				continue
			}
			// Wrap around edges (toroidal grid)
			r := (row + dr + s.rows) % s.rows
			c := (col + dc + s.cols) % s.cols
			if s.cells[r*s.cols+c] {
				count++
			}
		}
	}
	return count
}

func (s *lifeState) countAlive() int {
	count := 0
	for _, alive := range s.cells {
		if alive {
			count++
		}
	}
	return count
}

func (s *lifeState) cellsEqual(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *lifeState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	for row := range s.rows {
		for col := range s.cols {
			idx := row*s.cols + col
			alive := s.cells[idx]
			wasAlive := s.prev[idx]

			if alive {
				s.drawCell(pixels, col, row, lifeAlpha)
			} else if wasAlive {
				// Fading ghost of recently dead cells
				s.drawCell(pixels, col, row, lifeFadeAlpha)
			}
		}
	}

	return pixels
}

func (s *lifeState) drawCell(pixels []uint8, col, row int, alpha uint8) {
	startX := col * lifeCellSize
	startY := row * lifeCellSize

	// Draw cell with 1px gap for grid appearance
	for dy := 0; dy < lifeCellSize-1; dy++ {
		for dx := 0; dx < lifeCellSize-1; dx++ {
			px := startX + dx
			py := startY + dy
			if px >= s.width || py >= s.height {
				continue
			}
			offset := (py*s.width + px) * rgbaChannels
			if alpha > pixels[offset+3] {
				pixels[offset] = 180   // R — slight green tint
				pixels[offset+1] = 255 // G
				pixels[offset+2] = 180 // B
				pixels[offset+3] = alpha
			}
		}
	}
}

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
	state := &invadersState{
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

	pc.AddRasterOverlay(0.4, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.reset()
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
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

// --- Snake Effect ---

const (
	snakeInitLength    = 5
	snakeAlpha         = 110
	snakeFoodAlpha     = 130
	snakeFrameInterval = 90 * time.Millisecond
	snakeTargetCols    = 40
)

type snakeDir int

const (
	snakeDirUp snakeDir = iota
	snakeDirDown
	snakeDirRight
	snakeDirLeft
)

type snakePoint struct {
	x, y int
}

type snakeState struct {
	width, height int
	cols, rows    int
	cellSize      int
	body          []snakePoint
	dir           snakeDir
	food          snakePoint
	frameCount    int
}

func (p *shenanigans) startSnake(pc linkquisition.PickerCanvas) {
	state := &snakeState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}
	state.computeGrid()
	state.reset()

	pc.AddRasterOverlay(0.5, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.computeGrid()
			state.reset()
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(snakeFrameInterval)
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

func (s *snakeState) computeGrid() {
	s.cellSize = s.width / snakeTargetCols
	if s.cellSize < 6 {
		s.cellSize = 6
	}
	if s.cellSize > 16 {
		s.cellSize = 16
	}
	s.cols = s.width / s.cellSize
	s.rows = s.height / s.cellSize
}
func (s *snakeState) reset() {
	// Start in the center going right
	cx := s.cols / 2
	cy := s.rows / 2
	s.body = make([]snakePoint, snakeInitLength)
	for i := range s.body {
		s.body[i] = snakePoint{x: cx - i, y: cy}
	}
	s.dir = snakeDirRight
	s.placeFood()
}

func (s *snakeState) placeFood() {
	// Place food at a random empty cell
	for range 100 {
		p := snakePoint{x: rand.IntN(s.cols), y: rand.IntN(s.rows)}
		if !s.isBody(p) {
			s.food = p
			return
		}
	}
	// Fallback: just place it anywhere
	s.food = snakePoint{x: rand.IntN(s.cols), y: rand.IntN(s.rows)}
}

func (s *snakeState) isBody(p snakePoint) bool {
	for _, seg := range s.body {
		if seg == p {
			return true
		}
	}
	return false
}

func (s *snakeState) update() {
	s.frameCount++

	// AI: decide direction every frame
	s.chooseDirection()

	// Move head
	head := s.body[0]
	switch s.dir {
	case snakeDirUp:
		head.y--
	case snakeDirDown:
		head.y++
	case snakeDirRight:
		head.x++
	case snakeDirLeft:
		head.x--
	}

	// Wrap around edges
	head.x = (head.x + s.cols) % s.cols
	head.y = (head.y + s.rows) % s.rows

	// Check self-collision — reset if hit
	if s.isBody(head) {
		s.reset()
		return
	}

	// Grow or move
	s.body = append([]snakePoint{head}, s.body...)
	if head == s.food {
		s.placeFood()
	} else {
		s.body = s.body[:len(s.body)-1]
	}
}

func (s *snakeState) chooseDirection() {
	// Score each valid direction by distance to food after moving
	type option struct {
		dir  snakeDir
		dist int
	}

	var options []option
	for _, d := range []snakeDir{snakeDirUp, snakeDirDown, snakeDirLeft, snakeDirRight} {
		if d == s.oppositeDir() {
			continue
		}
		next := s.nextHead(d)
		if s.isBody(next) {
			continue
		}
		dist := s.wrapDist(next, s.food)
		options = append(options, option{d, dist})
	}

	if len(options) == 0 {
		return // no safe move, will hit self and reset
	}

	// Pick the direction that minimizes distance to food
	best := options[0]
	for _, o := range options[1:] {
		if o.dist < best.dist {
			best = o
		}
	}

	s.dir = best.dir
}

// wrapDist returns the Manhattan distance accounting for toroidal wrapping.
func (s *snakeState) wrapDist(a, b snakePoint) int {
	dx := abs(a.x - b.x)
	dy := abs(a.y - b.y)
	if dx > s.cols/2 {
		dx = s.cols - dx
	}
	if dy > s.rows/2 {
		dy = s.rows - dy
	}
	return dx + dy
}

func (s *snakeState) nextHead(d snakeDir) snakePoint {
	head := s.body[0]
	switch d {
	case snakeDirUp:
		head.y--
	case snakeDirDown:
		head.y++
	case snakeDirRight:
		head.x++
	case snakeDirLeft:
		head.x--
	}
	head.x = (head.x + s.cols) % s.cols
	head.y = (head.y + s.rows) % s.rows
	return head
}

func (s *snakeState) oppositeDir() snakeDir {
	switch s.dir {
	case snakeDirUp:
		return snakeDirDown
	case snakeDirDown:
		return snakeDirUp
	case snakeDirLeft:
		return snakeDirRight
	case snakeDirRight:
		return snakeDirLeft
	}
	return snakeDirLeft
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (s *snakeState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw body with gradient (head brighter, tail dimmer)
	for i, seg := range s.body {
		fade := float64(i) / float64(len(s.body))
		alpha := uint8(float64(snakeAlpha) * (1.0 - fade*0.6))
		s.drawSnakeCell(pixels, seg.x, seg.y, 150, 255, 150, alpha)
	}

	// Draw food (pulsing)
	pulse := uint8(sinNorm(float64(s.frameCount)*0.15) * 60)
	s.drawSnakeCell(pixels, s.food.x, s.food.y, 255, 100, 100, snakeFoodAlpha+pulse)

	return pixels
}

func (s *snakeState) drawSnakeCell(pixels []uint8, cx, cy int, r, g, b, a uint8) {
	startX := cx * s.cellSize
	startY := cy * s.cellSize

	// Draw with 1px padding for grid look
	for dy := 1; dy < s.cellSize-1; dy++ {
		for dx := 1; dx < s.cellSize-1; dx++ {
			px := startX + dx
			py := startY + dy
			if px >= s.width || py >= s.height {
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

// --- Rain Effect ---

const (
	rainDropCount   = 150
	rainAlpha       = 120
	rainSplashAlpha = 100
)

type raindrop struct {
	x      float64
	y      float64
	speed  float64
	length float64
	width  float64
}

type rainState struct {
	width, height int
	drops         []raindrop
	initialized   bool
}

func (p *shenanigans) startRain(pc linkquisition.PickerCanvas) {
	state := &rainState{
		width:  pc.Width(),
		height: pc.Height(),
	}
	if state.width == 0 {
		state.width = 600
	}
	if state.height == 0 {
		state.height = 400
	}

	pc.AddRasterOverlay(0.5, func(w, h int) []uint8 {
		if !state.initialized || w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.init()
			state.initialized = true
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
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

func (s *rainState) init() {
	s.drops = make([]raindrop, rainDropCount)
	for i := range s.drops {
		s.drops[i] = s.newDrop(true)
	}
}

func (s *rainState) newDrop(onScreen bool) raindrop {
	y := -(rand.Float64() * float64(s.height))
	if onScreen {
		y = rand.Float64() * float64(s.height)
	}

	return raindrop{
		x:      rand.Float64() * float64(s.width),
		y:      y,
		speed:  8 + rand.Float64()*12,
		length: 15 + rand.Float64()*30,
		width:  1 + rand.Float64()*1.5,
	}
}

func (s *rainState) update() {
	for i := range s.drops {
		s.drops[i].y += s.drops[i].speed

		// Slight wind drift
		s.drops[i].x += 0.5

		// Respawn above when falling off bottom
		if s.drops[i].y > float64(s.height)+s.drops[i].length {
			s.drops[i] = s.newDrop(false)
		}
	}
}

func (s *rainState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	for i := range s.drops {
		s.drawDrop(pixels, s.drops[i])
	}

	return pixels
}

func (s *rainState) drawDrop(pixels []uint8, d raindrop) { //nolint:gocyclo
	// Draw a vertical streak
	startY := int(d.y - d.length)
	endY := int(d.y)
	x := int(d.x)
	dropWidth := int(d.width)

	for py := startY; py <= endY; py++ {
		if py < 0 || py >= s.height {
			continue
		}

		// Fade: brighter at the bottom (leading edge), dimmer at top
		progress := float64(py-startY) / d.length
		alpha := uint8(float64(rainAlpha) * progress)

		for dx := 0; dx < dropWidth; dx++ {
			px := x + dx
			if px < 0 || px >= s.width {
				continue
			}
			offset := (py*s.width + px) * rgbaChannels
			if alpha > pixels[offset+3] {
				pixels[offset] = 180   // R — cool blue-white
				pixels[offset+1] = 200 // G
				pixels[offset+2] = 255 // B
				pixels[offset+3] = alpha
			}
		}
	}

	// Splash at the bottom when the drop hits — wider and a few pixels tall
	if endY >= s.height-3 && endY < s.height {
		splashWidth := int(d.width) + 4
		for dy := 0; dy < 3; dy++ {
			py := s.height - 1 - dy
			splAlpha := rainSplashAlpha - uint8(dy*30)
			for dx := -splashWidth; dx <= splashWidth; dx++ {
				px := x + dx
				if px >= 0 && px < s.width && py >= 0 {
					offset := (py*s.width + px) * rgbaChannels
					if splAlpha > pixels[offset+3] {
						pixels[offset] = 200
						pixels[offset+1] = 220
						pixels[offset+2] = 255
						pixels[offset+3] = splAlpha
					}
				}
			}
		}
	}
}

// --- Breakout Effect ---

const (
	breakoutRows        = 5
	breakoutCols        = 10
	breakoutBallSize    = 5
	breakoutPaddleH     = 6
	breakoutAlpha       = 120
	breakoutBallAlpha   = 140
	breakoutPaddleAlpha = 110
)

var breakoutColors = [5][3]uint8{
	{220, 50, 50},  // red
	{220, 150, 0},  // orange
	{220, 220, 0},  // yellow
	{50, 200, 50},  // green
	{50, 120, 220}, // blue
}

type breakoutBrick struct {
	alive bool
	color int
}

type breakoutState struct {
	width, height int

	// Grid dimensions (computed from window)
	brickW, brickH int
	marginTop      int

	// Bricks
	bricks []breakoutBrick

	// Ball
	ballX, ballY   float64
	ballVX, ballVY float64

	// Paddle
	paddleX     float64
	paddleW     int
	paddleY     int
	paddlePhase float64
}

func (p *shenanigans) startBreakout(pc linkquisition.PickerCanvas) {
	state := &breakoutState{
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

	pc.AddRasterOverlay(0.4, func(w, h int) []uint8 {
		if w != state.width || h != state.height {
			state.width = w
			state.height = h
			state.reset()
		}
		return p.invertForLight(state.render())
	})

	go func() {
		ticker := time.NewTicker(frameInterval)
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

func (s *breakoutState) reset() {
	// Compute brick sizes to fill ~80% of width
	s.brickW = (s.width * 8) / (breakoutCols * 10)
	if s.brickW < 10 {
		s.brickW = 10
	}
	s.brickH = s.brickW / 3
	if s.brickH < 6 {
		s.brickH = 6
	}
	s.marginTop = s.height / 8
	s.paddleW = s.brickW * 2
	s.paddleY = s.height - s.height/8

	// Create bricks
	s.bricks = make([]breakoutBrick, breakoutRows*breakoutCols)
	for row := range breakoutRows {
		for col := range breakoutCols {
			s.bricks[row*breakoutCols+col] = breakoutBrick{
				alive: true,
				color: row % len(breakoutColors),
			}
		}
	}

	// Reset ball to center, random angle
	s.ballX = float64(s.width) / 2
	s.ballY = float64(s.height) * 0.6
	s.ballVX = 3.0 + rand.Float64()*2.0
	if rand.IntN(2) == 0 {
		s.ballVX = -s.ballVX
	}
	s.ballVY = -(3.0 + rand.Float64()*1.5)

	// Paddle starts centered
	s.paddleX = float64(s.width) / 2
	s.paddlePhase = rand.Float64() * 6.28
}

func (s *breakoutState) update() {
	// Move ball
	s.ballX += s.ballVX
	s.ballY += s.ballVY

	// Bounce off walls
	if s.ballX <= 0 || s.ballX >= float64(s.width)-1 {
		s.ballVX = -s.ballVX
		s.ballX = clampFloat(s.ballX, 1, float64(s.width)-2)
	}
	if s.ballY <= 0 {
		s.ballVY = -s.ballVY
		s.ballY = 1
	}

	// Ball falls below paddle — reset
	if s.ballY > float64(s.height) {
		s.reset()
		return
	}

	// Paddle AI: follow ball with oscillation
	s.paddlePhase += 0.03
	oscillation := sinApprox(s.paddlePhase) * float64(s.width) * 0.05
	target := s.ballX + oscillation
	diff := target - s.paddleX
	s.paddleX += clampPaddleMove(diff, 4.0)
	s.paddleX = clampFloat(s.paddleX, float64(s.paddleW/2), float64(s.width-s.paddleW/2))

	// Ball-paddle collision
	halfPaddle := float64(s.paddleW) / 2
	if s.ballVY > 0 && s.ballY >= float64(s.paddleY)-float64(breakoutPaddleH) &&
		s.ballY <= float64(s.paddleY) &&
		s.ballX >= s.paddleX-halfPaddle && s.ballX <= s.paddleX+halfPaddle {
		s.ballVY = -s.ballVY
		// Spin based on where it hit
		offset := (s.ballX - s.paddleX) / halfPaddle
		s.ballVX += offset * 2.0
		s.ballY = float64(s.paddleY) - float64(breakoutPaddleH) - 1
	}

	// Ball-brick collision
	s.checkBrickCollisions()

	// Speed cap
	maxSpd := 7.0
	s.ballVX = clampFloat(s.ballVX, -maxSpd, maxSpd)
	s.ballVY = clampFloat(s.ballVY, -maxSpd, maxSpd)

	// All bricks destroyed — reset
	allDead := true
	for i := range s.bricks {
		if s.bricks[i].alive {
			allDead = false
			break
		}
	}
	if allDead {
		s.reset()
	}
}

func (s *breakoutState) checkBrickCollisions() {
	startX := (s.width - breakoutCols*s.brickW) / 2

	for row := range breakoutRows {
		for col := range breakoutCols {
			idx := row*breakoutCols + col
			if !s.bricks[idx].alive {
				continue
			}

			bx := float64(startX + col*s.brickW)
			by := float64(s.marginTop + row*(s.brickH+2))

			// Simple AABB check
			if s.ballX >= bx-float64(breakoutBallSize) &&
				s.ballX <= bx+float64(s.brickW)+float64(breakoutBallSize) &&
				s.ballY >= by-float64(breakoutBallSize) &&
				s.ballY <= by+float64(s.brickH)+float64(breakoutBallSize) {
				s.bricks[idx].alive = false
				// Determine bounce direction
				centerX := bx + float64(s.brickW)/2
				centerY := by + float64(s.brickH)/2
				dx := s.ballX - centerX
				dy := s.ballY - centerY
				if abs(int(dx))*s.brickH > abs(int(dy))*s.brickW {
					s.ballVX = -s.ballVX
				} else {
					s.ballVY = -s.ballVY
				}
				return // only break one brick per frame
			}
		}
	}
}

func (s *breakoutState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)
	startX := (s.width - breakoutCols*s.brickW) / 2

	// Draw bricks
	for row := range breakoutRows {
		for col := range breakoutCols {
			idx := row*breakoutCols + col
			if !s.bricks[idx].alive {
				continue
			}
			c := breakoutColors[s.bricks[idx].color]
			bx := startX + col*s.brickW
			by := s.marginTop + row*(s.brickH+2)
			s.drawRect(pixels, bx+1, by+1, s.brickW-2, s.brickH-2, c[0], c[1], c[2], breakoutAlpha)
		}
	}

	// Draw paddle
	px := int(s.paddleX) - s.paddleW/2
	s.drawRect(pixels, px, s.paddleY, s.paddleW, breakoutPaddleH, 200, 200, 200, breakoutPaddleAlpha)

	// Draw ball
	bx := int(s.ballX) - breakoutBallSize/2
	by := int(s.ballY) - breakoutBallSize/2
	s.drawRect(pixels, bx, by, breakoutBallSize, breakoutBallSize, 255, 255, 255, breakoutBallAlpha)

	return pixels
}

func (s *breakoutState) drawRect(pixels []uint8, x, y, rw, rh int, r, g, b, a uint8) {
	for dy := 0; dy < rh; dy++ {
		for dx := 0; dx < rw; dx++ {
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
	state := &dinoState{
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
		ticker := time.NewTicker(frameInterval)
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

// --- Asteroids Effect ---

const (
	astShipSize      = 10
	astAlpha         = 120
	astBulletAlpha   = 140
	astBulletSpeed   = 6.0
	astBulletLife    = 40
	astMaxAsteroids  = 8
	astShootInterval = 15
	astTurnSpeed     = 0.05
	astThrust        = 0.12
	astFriction      = 0.98
)

type astVec struct{ x, y float64 }

type asteroid struct {
	pos    astVec
	vel    astVec
	radius float64
	edges  int // number of vertices (6-10)
}

type astBullet struct {
	pos  astVec
	vel  astVec
	life int
}

type asteroidsState struct {
	width, height int
	scale         float64

	// Ship
	shipPos   astVec
	shipVel   astVec
	shipAngle float64

	// Objects
	asteroids  []asteroid
	bullets    []astBullet
	shootTimer int
	frameCount int
}

func (p *shenanigans) startAsteroids(pc linkquisition.PickerCanvas) {
	state := &asteroidsState{
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
		ticker := time.NewTicker(frameInterval)
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

func (s *asteroidsState) reset() {
	s.scale = float64(s.width) / 600.0
	if s.scale < 0.5 {
		s.scale = 0.5
	}

	s.shipPos = astVec{float64(s.width) / 2, float64(s.height) / 2}
	s.shipVel = astVec{0, 0}
	s.shipAngle = 0
	s.bullets = nil
	s.shootTimer = 0
	s.frameCount = 0
	s.spawnAsteroids()
}

func (s *asteroidsState) spawnAsteroids() {
	s.asteroids = make([]asteroid, astMaxAsteroids)
	for i := range s.asteroids {
		// Spawn at edges
		var pos astVec
		if rand.IntN(2) == 0 {
			pos.x = rand.Float64() * float64(s.width)
			if rand.IntN(2) == 0 {
				pos.y = 0
			} else {
				pos.y = float64(s.height)
			}
		} else {
			pos.y = rand.Float64() * float64(s.height)
			if rand.IntN(2) == 0 {
				pos.x = 0
			} else {
				pos.x = float64(s.width)
			}
		}
		s.asteroids[i] = asteroid{
			pos:    pos,
			vel:    astVec{(rand.Float64() - 0.5) * 2, (rand.Float64() - 0.5) * 2},
			radius: (20 + rand.Float64()*15) * s.scale,
			edges:  6 + rand.IntN(4),
		}
	}
}

func (s *asteroidsState) update() {
	s.frameCount++

	// Ship AI: rotate toward nearest asteroid, thrust toward it, shoot
	nearest := s.nearestAsteroid()
	if nearest != nil {
		// Angle to nearest asteroid
		dx := nearest.pos.x - s.shipPos.x
		dy := nearest.pos.y - s.shipPos.y
		targetAngle := atan2Approx(dy, dx)

		// Rotate toward target
		diff := targetAngle - s.shipAngle
		for diff > 3.14159 {
			diff -= 6.28318
		}
		for diff < -3.14159 {
			diff += 6.28318
		}
		if diff > astTurnSpeed {
			s.shipAngle += astTurnSpeed
		} else if diff < -astTurnSpeed {
			s.shipAngle -= astTurnSpeed
		}

		// Thrust
		s.shipVel.x += cosApprox(s.shipAngle) * astThrust
		s.shipVel.y += sinApprox(s.shipAngle) * astThrust
	}

	// Friction
	s.shipVel.x *= astFriction
	s.shipVel.y *= astFriction

	// Move ship
	s.shipPos.x += s.shipVel.x
	s.shipPos.y += s.shipVel.y

	// Wrap ship
	s.shipPos.x = s.wrapX(s.shipPos.x)
	s.shipPos.y = s.wrapY(s.shipPos.y)

	// Shoot
	s.shootTimer++
	if s.shootTimer >= astShootInterval && nearest != nil {
		s.shootTimer = 0
		s.bullets = append(s.bullets, astBullet{
			pos:  s.shipPos,
			vel:  astVec{cosApprox(s.shipAngle) * astBulletSpeed, sinApprox(s.shipAngle) * astBulletSpeed},
			life: astBulletLife,
		})
	}

	// Move bullets
	alive := s.bullets[:0]
	for i := range s.bullets {
		s.bullets[i].pos.x += s.bullets[i].vel.x
		s.bullets[i].pos.y += s.bullets[i].vel.y
		s.bullets[i].life--
		if s.bullets[i].life > 0 {
			alive = append(alive, s.bullets[i])
		}
	}
	s.bullets = alive

	// Move asteroids
	for i := range s.asteroids {
		s.asteroids[i].pos.x = s.wrapX(s.asteroids[i].pos.x + s.asteroids[i].vel.x)
		s.asteroids[i].pos.y = s.wrapY(s.asteroids[i].pos.y + s.asteroids[i].vel.y)
	}

	// Bullet-asteroid collision
	s.checkBulletHits()

	// Respawn if all destroyed
	if len(s.asteroids) == 0 {
		s.spawnAsteroids()
	}
}

func (s *asteroidsState) checkBulletHits() {
	var newBullets []astBullet
	for _, b := range s.bullets {
		hit := false
		for i := range s.asteroids {
			dx := b.pos.x - s.asteroids[i].pos.x
			dy := b.pos.y - s.asteroids[i].pos.y
			dist := dx*dx + dy*dy
			r := s.asteroids[i].radius
			if dist < r*r {
				// Split asteroid
				if s.asteroids[i].radius > 12*s.scale {
					newR := s.asteroids[i].radius * 0.55
					for range 2 {
						s.asteroids = append(s.asteroids, asteroid{
							pos:    s.asteroids[i].pos,
							vel:    astVec{(rand.Float64() - 0.5) * 3, (rand.Float64() - 0.5) * 3},
							radius: newR,
							edges:  5 + rand.IntN(4),
						})
					}
				}
				// Remove hit asteroid
				s.asteroids = append(s.asteroids[:i], s.asteroids[i+1:]...)
				hit = true
				break
			}
		}
		if !hit {
			newBullets = append(newBullets, b)
		}
	}
	s.bullets = newBullets
}

func (s *asteroidsState) nearestAsteroid() *asteroid {
	if len(s.asteroids) == 0 {
		return nil
	}
	best := 0
	bestDist := 1e18
	for i := range s.asteroids {
		dx := s.asteroids[i].pos.x - s.shipPos.x
		dy := s.asteroids[i].pos.y - s.shipPos.y
		d := dx*dx + dy*dy
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	return &s.asteroids[best]
}

func (s *asteroidsState) wrapX(x float64) float64 {
	w := float64(s.width)
	for x < 0 {
		x += w
	}
	for x >= w {
		x -= w
	}
	return x
}

func (s *asteroidsState) wrapY(y float64) float64 {
	h := float64(s.height)
	for y < 0 {
		y += h
	}
	for y >= h {
		y -= h
	}
	return y
}

func (s *asteroidsState) render() []uint8 {
	w, h := s.width, s.height
	if w == 0 || h == 0 {
		return make([]uint8, rgbaChannels)
	}

	pixels := make([]uint8, w*h*rgbaChannels)

	// Draw asteroids as wireframe polygons
	for i := range s.asteroids {
		s.drawAsteroid(pixels, s.asteroids[i])
	}

	// Draw bullets
	for _, b := range s.bullets {
		bx, by := int(b.pos.x), int(b.pos.y)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				s.setAstPixel(pixels, bx+dx, by+dy, 255, 255, 255, astBulletAlpha)
			}
		}
	}

	// Draw ship (triangle)
	s.drawShip(pixels)

	return pixels
}

func (s *asteroidsState) drawAsteroid(pixels []uint8, a asteroid) {
	cx, cy := a.pos.x, a.pos.y
	n := a.edges

	for i := range n {
		angle1 := float64(i) * 6.28318 / float64(n)
		angle2 := float64(i+1) * 6.28318 / float64(n)
		x1 := cx + cosApprox(angle1)*a.radius
		y1 := cy + sinApprox(angle1)*a.radius
		x2 := cx + cosApprox(angle2)*a.radius
		y2 := cy + sinApprox(angle2)*a.radius
		s.drawLine(pixels, int(x1), int(y1), int(x2), int(y2), 200, 200, 200, astAlpha)
	}
}

func (s *asteroidsState) drawShip(pixels []uint8) {
	// Triangle pointing in shipAngle direction
	size := float64(astShipSize) * s.scale
	cx, cy := s.shipPos.x, s.shipPos.y

	// Nose
	nx := cx + cosApprox(s.shipAngle)*size
	ny := cy + sinApprox(s.shipAngle)*size
	// Left wing
	lx := cx + cosApprox(s.shipAngle+2.5)*size*0.6
	ly := cy + sinApprox(s.shipAngle+2.5)*size*0.6
	// Right wing
	rx := cx + cosApprox(s.shipAngle-2.5)*size*0.6
	ry := cy + sinApprox(s.shipAngle-2.5)*size*0.6

	s.drawLine(pixels, int(nx), int(ny), int(lx), int(ly), 255, 255, 255, astAlpha)
	s.drawLine(pixels, int(nx), int(ny), int(rx), int(ry), 255, 255, 255, astAlpha)
	s.drawLine(pixels, int(lx), int(ly), int(rx), int(ry), 255, 255, 255, astAlpha)
}

// drawLine draws a line using Bresenham's algorithm.
func (s *asteroidsState) drawLine(pixels []uint8, x0, y0, x1, y1 int, r, g, b, a uint8) { //nolint:unparam
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy

	for {
		s.setAstPixel(pixels, x0, y0, r, g, b, a)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (s *asteroidsState) setAstPixel(pixels []uint8, x, y int, r, g, b, a uint8) {
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

// cosApprox uses sinApprox shifted by pi/2.
func cosApprox(x float64) float64 {
	return sinApprox(x + 1.5707963)
}

// atan2Approx is a rough atan2 approximation.
func atan2Approx(y, x float64) float64 {
	// Simple approximation using the identity and sinApprox
	const pi = 3.14159265
	if x == 0 {
		if y > 0 {
			return pi / 2
		}
		return -pi / 2
	}
	a := y / x
	// Clamp for stability
	if a > 10 {
		a = 10
	} else if a < -10 {
		a = -10
	}
	// Polynomial approximation of atan for small values
	result := a / (1 + 0.28*a*a)
	if x < 0 {
		if y >= 0 {
			result += pi
		} else {
			result -= pi
		}
	}
	return result
}

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

// NewForTesting creates a fresh shenanigans instance for use in tests,
// avoiding copying the package-level Plugin variable (which may contain a mutex).
func NewForTesting() *shenanigans {
	return &shenanigans{}
}

var Plugin shenanigans
