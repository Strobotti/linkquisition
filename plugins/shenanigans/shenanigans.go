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
	effectRandom      = "random"
	frameInterval     = 30 * time.Millisecond
	fireFrameInterval = 25 * time.Millisecond
	fireWidth         = 200
	fireHeight        = 100
	matrixColumns     = 40
	rgbaChannels      = 4
)

// Compile-time interface checks.
var _ linkquisition.Plugin = (*shenanigans)(nil)
var _ linkquisition.PluginUIHook = (*shenanigans)(nil)

type shenanigans struct {
	serviceProvider linkquisition.PluginServiceProvider
	effect          string
	stopped         atomic.Bool
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
				Key:         "effect",
				Label:       "Effect",
				Description: "Which visual effect to show on the picker window",
				Type:        linkquisition.SettingTypeChoice,
				Default:     effectRandom,
				Options: []string{
					effectMatrix, effectFire, effectSnow, effectPlasma,
					effectStarfield, effectAurora, effectGlitch, effectPride,
					effectFootball, effectFireworks, effectRandom,
				},
			},
		},
	}
}

func (p *shenanigans) Setup(
	serviceProvider linkquisition.PluginServiceProvider, config map[string]interface{},
) error {
	p.serviceProvider = serviceProvider

	if effectVal, ok := config["effect"]; ok {
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

func (p *shenanigans) OnPickerShown(canvas linkquisition.PickerCanvas) {
	effect := p.effect
	if effect == effectRandom {
		effects := []string{
			effectMatrix, effectFire, effectSnow, effectPlasma,
			effectStarfield, effectAurora, effectGlitch, effectPride,
			effectFootball, effectFireworks,
		}
		effect = effects[rand.IntN(len(effects))]
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
	}
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
			chars: generateMatrixChars(20 + rand.IntN(15)), //nolint:mnd
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
			s.columns[i].chars = generateMatrixChars(20 + rand.IntN(15)) //nolint:mnd
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
			if cy < 0 || cy >= h {
				continue
			}

			// Fade out older characters
			brightness := uint8(255 - min(j*12, 230)) //nolint:mnd

			// Draw a simple block for each character
			for dx := 2; dx < min(colWidth-2, 10); dx++ {
				for dy := 2; dy < charHeight-2; dy++ {
					px := x + dx
					py := cy + dy
					if px < w && py < h {
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
		s.grid[s.height-1][x] = uint8(180 + rand.IntN(76))  //nolint:mnd
		s.grid[s.height-2][x] = uint8(150 + rand.IntN(106)) //nolint:mnd
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

			avg := sum / 9 //nolint:mnd

			// Cooling increases toward the top for natural fadeout
			coolBase := 2 + (s.height-y)/15 //nolint:mnd
			cooling := rand.IntN(coolBase + 1)
			val := avg - cooling

			if val < 0 {
				val = 0
			}
			s.grid[y][x] = uint8(val) //nolint:gosec
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

	return uint8(min(max(int(val), 0), 255)) //nolint:gosec
}

// fireColor maps a heat value (0-255) to a realistic fire palette.
// Gradient: transparent → dark red/brown → red → orange → gold → pale yellow
func fireColor(val uint8) (r, g, b, a uint8) {
	if val < 24 { //nolint:mnd
		return 0, 0, 0, 0
	}

	// Normalize to 0.0-1.0 range (24-255 → 0.0-1.0)
	t := float64(val-24) / 231.0 //nolint:mnd

	// Piecewise palette for realistic fire
	switch {
	case t < 0.2: //nolint:mnd
		// Black → dark maroon/brown
		p := t / 0.2       //nolint:mnd
		r = uint8(p * 80)  //nolint:mnd,gosec
		g = uint8(p * 10)  //nolint:mnd,gosec
		a = uint8(p * 180) //nolint:mnd,gosec
		return r, g, 0, a
	case t < 0.45: //nolint:mnd
		// Dark maroon → bright red
		p := (t - 0.2) / 0.25 //nolint:mnd
		r = uint8(80 + p*175) //nolint:mnd,gosec
		g = uint8(10 + p*20)  //nolint:mnd,gosec
		a = uint8(180 + p*75) //nolint:mnd,gosec
		return r, g, 0, a
	case t < 0.7: //nolint:mnd
		// Red → orange
		p := (t - 0.45) / 0.25 //nolint:mnd
		r = 255                //nolint:mnd
		g = uint8(30 + p*170)  //nolint:mnd,gosec
		a = 255                //nolint:mnd
		return r, g, 0, a
	case t < 0.9: //nolint:mnd
		// Orange → golden yellow
		p := (t - 0.7) / 0.2  //nolint:mnd
		r = 255               //nolint:mnd
		g = uint8(200 + p*55) //nolint:mnd,gosec
		b = uint8(p * 30)     //nolint:mnd,gosec
		a = 255               //nolint:mnd
		return r, g, b, a
	default:
		// Golden yellow → pale yellow/white tips
		p := (t - 0.9) / 0.1  //nolint:mnd
		r = 255               //nolint:mnd
		g = 255               //nolint:mnd
		b = uint8(30 + p*120) //nolint:mnd,gosec
		a = uint8(255 - p*80) //nolint:mnd,gosec
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

func (s *snowState) init() {
	s.flakes = make([]snowflake, snowFlakeCount)
	for i := range s.flakes {
		// Start most flakes above the window so they drift in gradually.
		// A few start on-screen so it's not completely empty at first.
		onScreen := i < snowFlakeCount/5 //nolint:mnd
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
		drift:  (rand.Float64() - 0.5) * 0.3, //nolint:mnd
		wobble: 0.3 + rand.Float64()*0.7,     //nolint:mnd
		phase:  rand.Float64() * 6.28,        //nolint:mnd
	}
}

func (s *snowState) update() {
	for i := range s.flakes {
		f := &s.flakes[i]
		f.y += f.speed
		f.phase += 0.05 //nolint:mnd

		// Gentle sine-wave wobble for horizontal drift
		f.x += f.drift + f.wobble*sinApprox(f.phase)*0.3 //nolint:mnd

		// Respawn at top if fallen below window
		if f.y > float64(s.height)+10 { //nolint:mnd
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
			alpha := uint8((1.0 - dist*dist) * 220) //nolint:mnd,gosec

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
	num := 16 * x * (pi - x)    //nolint:mnd
	den := 5*pi*pi - 4*x*(pi-x) //nolint:mnd
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
			state.time += 0.06 //nolint:mnd
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
			v1 := sinApprox((fx*4 + t) * 3.14159)                //nolint:mnd
			v2 := sinApprox((fy*4 + t*0.7) * 3.14159)            //nolint:mnd
			v3 := sinApprox(((fx+fy)*3 + t*1.3) * 3.14159)       //nolint:mnd
			v4 := sinApprox(((fx-fy)*2 + t*0.5) * 3.14159)       //nolint:mnd
			v5 := sinApprox(((fx*fx+fy*fy)*4 - t*0.9) * 3.14159) //nolint:mnd

			// Combine waves (result in -1 to 1 range, normalize to 0-1)
			val := (v1 + v2 + v3 + v4 + v5) / 5.0 //nolint:mnd
			val = (val + 1.0) / 2.0

			// Map to color using three phase-shifted sine waves for RGB
			r := uint8(sinNorm(val*3.14159*2+t*0.3) * 255)      //nolint:mnd,gosec
			g := uint8(sinNorm(val*3.14159*2+t*0.3+2.09) * 255) //nolint:mnd,gosec
			b := uint8(sinNorm(val*3.14159*2+t*0.3+4.19) * 255) //nolint:mnd,gosec

			offset := (py*w + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = 200 //nolint:mnd
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
		z = 0.1 + rand.Float64()*0.9 //nolint:mnd
	}

	return star{
		x: (rand.Float64() - 0.5) * 2.0,
		y: (rand.Float64() - 0.5) * 2.0,
		z: z,
	}
}

func (s *starfieldState) update() {
	for i := range s.stars {
		s.stars[i].z -= 0.015 //nolint:mnd

		// Respawn stars that have passed the viewer
		if s.stars[i].z <= 0.001 { //nolint:mnd
			s.stars[i] = s.newStar(false)
			s.stars[i].z = 0.9 + rand.Float64()*0.1 //nolint:mnd
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
		brightness := uint8(min(int((1.0-st.z)*255), 255)) //nolint:mnd,gosec
		size := int(1 + (1.0-st.z)*3)                      //nolint:mnd

		// Draw star with glow
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
				alpha := uint8(float64(brightness) * falloff) //nolint:gosec

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

	return pixels
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
			state.time += 0.03 //nolint:mnd
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
				wave := sinApprox((fx*(3+fl) + t*(0.4+fl*0.15) + fl*1.7) * 3.14159)     //nolint:mnd
				wave2 := sinApprox((fx*(2+fl*0.7) - t*(0.3+fl*0.1) + fl*2.3) * 3.14159) //nolint:mnd

				// Curtain shape: thin band that undulates
				curtainCenter := 0.2 + 0.15*fl + 0.1*(wave*0.5+0.5) //nolint:mnd
				curtainWidth := 0.08 + 0.04*wave2                   //nolint:mnd

				// Gaussian-like falloff from the curtain center
				dist := (fy - curtainCenter) / curtainWidth
				layerIntensity := fastExp(-dist * dist * 0.5) //nolint:mnd

				intensity += layerIntensity * (0.6 + 0.4/(fl+1)) //nolint:mnd
			}

			if intensity < 0.01 { //nolint:mnd
				continue
			}
			if intensity > 1.0 {
				intensity = 1.0
			}

			// Aurora color: shift from green to purple/blue based on position and time
			colorPhase := fx*0.5 + fy*0.3 + t*0.1 //nolint:mnd
			r, g, b := auroraColor(colorPhase, intensity)

			alpha := uint8(intensity * 180) //nolint:mnd,gosec

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
	p := sinApprox(phase * 3.14159 * 2) //nolint:mnd
	p = (p + 1.0) / 2.0                 // normalize to 0-1

	// Blend between green-dominant and purple-dominant
	var rf, gf, bf float64
	switch {
	case p < 0.33: //nolint:mnd
		// Green to teal
		t := p / 0.33 //nolint:mnd
		rf = 0.1 * t
		gf = 0.8 + 0.2*t //nolint:mnd
		bf = 0.2 + 0.5*t //nolint:mnd
	case p < 0.66: //nolint:mnd
		// Teal to purple
		t := (p - 0.33) / 0.33 //nolint:mnd
		rf = 0.1 + 0.5*t       //nolint:mnd
		gf = 1.0 - 0.6*t       //nolint:mnd
		bf = 0.7 + 0.3*t       //nolint:mnd
	default:
		// Purple back to green
		t := (p - 0.66) / 0.34 //nolint:mnd
		rf = 0.6 - 0.5*t       //nolint:mnd
		gf = 0.4 + 0.4*t       //nolint:mnd
		bf = 1.0 - 0.8*t       //nolint:mnd
	}

	r = uint8(rf * intensity * 255) //nolint:mnd,gosec
	g = uint8(gf * intensity * 255) //nolint:mnd,gosec
	b = uint8(bf * intensity * 255) //nolint:mnd,gosec
	return r, g, b
}

// fastExp approximates e^x for negative x values (used for Gaussian falloff).
func fastExp(x float64) float64 {
	if x < -6 { //nolint:mnd
		return 0
	}
	// Padé approximation: (1 + x/n)^n for small |x|
	// Using n=8 for reasonable accuracy
	t := 1.0 + x/8.0 //nolint:mnd
	t *= t           // ^2
	t *= t           // ^4
	t *= t           // ^8
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
			s.burstTimer = 20 + rand.IntN(40) //nolint:mnd
		} else {
			// Start burst
			s.isBursting = true
			s.burstTimer = 3 + rand.IntN(8) //nolint:mnd
			s.generateSlices()
		}
	} else if s.isBursting && s.frame%2 == 0 {
		// Regenerate slices during burst for flicker
		s.generateSlices()
	}
}

func (s *glitchState) generateSlices() {
	count := 3 + rand.IntN(8) //nolint:mnd
	s.slices = make([]glitchSlice, count)

	for i := range s.slices {
		s.slices[i] = glitchSlice{
			y:       rand.IntN(400),             //nolint:mnd
			height:  2 + rand.IntN(20),          //nolint:mnd
			offsetX: -30 + rand.IntN(60),        //nolint:mnd
			channel: rand.IntN(3),               //nolint:mnd
			alpha:   uint8(80 + rand.IntN(176)), //nolint:mnd,gosec
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
		sy := slice.y * h / 400      //nolint:mnd
		sh := slice.height * h / 400 //nolint:mnd

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
				case 2: // Blue/cyan shift
					pixels[offset+2] = slice.alpha
					pixels[offset+1] = slice.alpha / 3
					pixels[offset+3] = slice.alpha / 2
				}
			}
		}
	}

	// Add random static noise during bursts
	if s.isBursting {
		noiseCount := w * h / 40 //nolint:mnd
		for range noiseCount {
			px := rand.IntN(w)
			py := rand.IntN(h)
			offset := (py*w + px) * rgbaChannels
			v := uint8(rand.IntN(256)) //nolint:mnd,gosec
			pixels[offset] = v
			pixels[offset+1] = v
			pixels[offset+2] = v
			pixels[offset+3] = uint8(rand.IntN(100)) //nolint:mnd,gosec
		}
	}

	return pixels
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
			state.time += 0.04 //nolint:mnd
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
			amplitude := fx * fx * 0.05 //nolint:mnd
			wave := sinApprox((fx*2.0-t*1.5)*3.14159) * amplitude
			wave2 := sinApprox((fx*3.0-t*2.0)*3.14159) * amplitude * 0.4 //nolint:mnd

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
			blend = blend * blend * (3 - 2*blend) //nolint:mnd

			r := uint8(float64(c1[0])*(1-blend) + float64(c2[0])*blend) //nolint:gosec
			g := uint8(float64(c1[1])*(1-blend) + float64(c2[1])*blend) //nolint:gosec
			b := uint8(float64(c1[2])*(1-blend) + float64(c2[2])*blend) //nolint:gosec

			// Subtle shading to simulate fabric folds (stronger toward free edge)
			foldDepth := fx * 0.15                                               //nolint:mnd
			shade := 1.0 - foldDepth + foldDepth*sinApprox((fx*3-t*1.5)*3.14159) //nolint:mnd
			r = uint8(float64(r) * shade)                                        //nolint:gosec
			g = uint8(float64(g) * shade)                                        //nolint:gosec
			b = uint8(float64(b) * shade)                                        //nolint:gosec

			offset := (py*w + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = 200 //nolint:mnd
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
			state.time += 0.03 //nolint:mnd
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
			stripeWidth := 0.08 //nolint:mnd
			stripe := int(fx/stripeWidth) % 2
			var gr, gg, gb uint8
			if stripe == 0 {
				gr, gg, gb = 34, 139, 34 //nolint:mnd
			} else {
				gr, gg, gb = 30, 124, 30 //nolint:mnd
			}

			// Draw white pitch lines
			isLine := false

			// Center line (vertical)
			if absF(fx-0.5) < 0.004 { //nolint:mnd
				isLine = true
			}

			// Center circle
			cx, cy := 0.5, 0.5
			dist := (fx-cx)*(fx-cx)*1.5 + (fy-cy)*(fy-cy) //nolint:mnd
			if absF(dist-0.04) < 0.003 {                  //nolint:mnd
				isLine = true
			}

			// Center dot
			if dist < 0.002 { //nolint:mnd
				isLine = true
			}

			// Outer boundary
			if fx < 0.02 || fx > 0.98 || fy < 0.03 || fy > 0.97 { //nolint:mnd
				if fx > 0.015 && fx < 0.985 && fy > 0.025 && fy < 0.975 { //nolint:mnd
					isLine = true
				}
			}

			// Penalty areas (left and right)
			penaltyW := 0.15 //nolint:mnd
			penaltyH := 0.35 //nolint:mnd
			penaltyTop := 0.5 - penaltyH
			penaltyBot := 0.5 + penaltyH

			// Left penalty area
			if fx < penaltyW && fy > penaltyTop && fy < penaltyBot {
				if absF(fx-penaltyW) < 0.004 || absF(fy-penaltyTop) < 0.005 || absF(fy-penaltyBot) < 0.005 { //nolint:mnd
					isLine = true
				}
			}

			// Right penalty area
			if fx > (1-penaltyW) && fy > penaltyTop && fy < penaltyBot {
				if absF(fx-(1-penaltyW)) < 0.004 || absF(fy-penaltyTop) < 0.005 || absF(fy-penaltyBot) < 0.005 { //nolint:mnd
					isLine = true
				}
			}

			// Animated element: a "spotlight" sweeping across the pitch
			spotX := 0.5 + 0.4*sinApprox(t*1.5)     //nolint:mnd
			spotY := 0.5 + 0.3*sinApprox(t*1.1+1.0) //nolint:mnd
			spotDist := (fx-spotX)*(fx-spotX) + (fy-spotY)*(fy-spotY)
			spotLight := fastExp(-spotDist*15) * 0.3 //nolint:mnd

			var r, g, b uint8
			if isLine {
				r, g, b = 255, 255, 255
			} else {
				r = uint8(min(int(float64(gr)*(1+spotLight)), 255)) //nolint:gosec
				g = uint8(min(int(float64(gg)*(1+spotLight)), 255)) //nolint:gosec
				b = uint8(min(int(float64(gb)*(1+spotLight)), 255)) //nolint:gosec
			}

			offset := (py*w + px) * rgbaChannels
			pixels[offset] = r
			pixels[offset+1] = g
			pixels[offset+2] = b
			pixels[offset+3] = 180 //nolint:mnd
		}
	}

	return pixels
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
	if len(s.rockets) < fireworksMaxRockets && rand.IntN(100) < fireworksLaunchChance { //nolint:mnd
		color := fireworksColors[rand.IntN(len(fireworksColors))]
		s.rockets = append(s.rockets, fireworksRocket{
			x:       0.2 + rand.Float64()*0.6, //nolint:mnd
			y:       1.0,
			vy:      -0.025 - rand.Float64()*0.015, //nolint:mnd
			targetY: 0.15 + rand.Float64()*0.35,    //nolint:mnd
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
					angle := rand.Float64() * 6.283       //nolint:mnd
					speed := 0.005 + rand.Float64()*0.015 //nolint:mnd
					r.particles[j] = fireworksParticle{
						x:    r.x,
						y:    r.y,
						vx:   sinApprox(angle) * speed,
						vy:   sinApprox(angle+1.5708) * speed, //nolint:mnd
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
				p.vy += 0.0004  // gravity //nolint:mnd
				p.vx *= 0.98    // drag    //nolint:mnd
				p.vy *= 0.98    //nolint:mnd
				p.life -= 0.015 //nolint:mnd
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
			s.drawDot(pixels, px, py, 2, 255, 220, 150, 255) //nolint:mnd
			// Trail
			s.drawDot(pixels, px, py+3, 1, 255, 150, 50, 150) //nolint:mnd
			s.drawDot(pixels, px, py+6, 1, 255, 100, 30, 80)  //nolint:mnd
		} else {
			// Draw particles
			for _, p := range rocket.particles {
				if p.life <= 0 {
					continue
				}
				px := int(p.x * float64(w))
				py := int(p.y * float64(h))
				alpha := uint8(p.life * 255) //nolint:mnd,gosec
				size := 1
				if p.life > 0.7 { //nolint:mnd
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

// NewForTesting creates a fresh shenanigans instance for use in tests,
// avoiding copying the package-level Plugin variable (which may contain a mutex).
func NewForTesting() *shenanigans {
	return &shenanigans{}
}

var Plugin shenanigans
