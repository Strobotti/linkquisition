//nolint:mnd,gosec // Test file for visual effects plugin — magic numbers expected.
package main

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strobotti/linkquisition"
)

func newTestServiceProvider() linkquisition.PluginServiceProvider {
	return linkquisition.NewPluginServiceProvider(
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
		&linkquisition.Settings{},
		"",
	)
}

// mockPickerCanvas implements PickerCanvas for testing.
type mockPickerCanvas struct {
	drawFn       func(w, h int) []uint8
	refreshes    int
	overlayAdded bool
}

func (m *mockPickerCanvas) AddRasterOverlay(_ float64, draw func(w, h int) []uint8) {
	m.drawFn = draw
	m.overlayAdded = true
}

func (m *mockPickerCanvas) ScheduleRefresh() {
	m.refreshes++
}

func (m *mockPickerCanvas) Width() int  { return 600 }
func (m *mockPickerCanvas) Height() int { return 400 }

func TestShenanigans_Metadata(t *testing.T) {
	p := NewForTesting()
	meta := p.Metadata()

	assert.Equal(t, "Shenanigans", meta.Name)
	assert.NotEmpty(t, meta.Description)
	assert.Len(t, meta.Settings, 1)
	assert.Equal(t, "effect", meta.Settings[0].Key)
	assert.Equal(t, linkquisition.SettingTypeChoice, meta.Settings[0].Type)
	assert.Equal(t, []string{
		"random",
		"aurora", "fire", "fireworks", "football",
		"glitch", "matrix", "plasma", "pride",
		"snow", "starfield",
	}, meta.Settings[0].Options)
}

func TestShenanigans_Setup_DefaultEffect(t *testing.T) {
	p := NewForTesting()
	err := p.Setup(newTestServiceProvider(), map[string]interface{}{})

	assert.NoError(t, err)
	assert.Equal(t, effectRandom, p.effect)
}

func TestShenanigans_Setup_SpecificEffect(t *testing.T) {
	p := NewForTesting()
	err := p.Setup(newTestServiceProvider(), map[string]interface{}{
		"effect": "fire",
	})

	assert.NoError(t, err)
	assert.Equal(t, effectFire, p.effect)
}

func TestShenanigans_ProcessURL_PassThrough(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{})

	testURL := "https://example.com/page?query=test"
	result := p.ProcessURL(context.Background(), testURL)

	assert.Equal(t, testURL, result.URL)
	assert.Equal(t, linkquisition.ActionContinue, result.Action)
	assert.True(t, result.ContinueChain)
	assert.Empty(t, result.Message)
}

func TestShenanigans_ImplementsPluginUIHook(t *testing.T) {
	p := NewForTesting()

	// Verify the plugin satisfies the PluginUIHook interface at compile time
	var hook linkquisition.PluginUIHook = p
	assert.NotNil(t, hook)
}

func TestShenanigans_OnPickerShown_Matrix(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{"effect": "matrix"})

	mc := &mockPickerCanvas{}
	p.OnPickerShown(mc)

	assert.True(t, mc.overlayAdded)
	assert.NotNil(t, mc.drawFn)

	// Call the draw function to ensure it doesn't panic
	pixels := mc.drawFn(200, 100)
	assert.Len(t, pixels, 200*100*4)
}

func TestShenanigans_OnPickerShown_Fire(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{"effect": "fire"})

	mc := &mockPickerCanvas{}
	p.OnPickerShown(mc)

	assert.True(t, mc.overlayAdded)
	assert.NotNil(t, mc.drawFn)

	// Call the draw function to ensure it doesn't panic
	pixels := mc.drawFn(200, 100)
	assert.Len(t, pixels, 200*100*4)
}

func TestShenanigans_OnPickerShown_Snow(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{"effect": "snow"})

	mc := &mockPickerCanvas{}
	p.OnPickerShown(mc)

	assert.True(t, mc.overlayAdded)
	assert.NotNil(t, mc.drawFn)

	// Call the draw function to ensure it doesn't panic
	pixels := mc.drawFn(600, 400)
	assert.Len(t, pixels, 600*400*4)
}

func TestSnowState_Update(t *testing.T) {
	state := &snowState{width: 600, height: 400}
	state.init()

	assert.Len(t, state.flakes, snowFlakeCount)

	// Should not panic
	state.update()

	// All flakes should still be within bounds (with wrapping)
	for _, f := range state.flakes {
		assert.GreaterOrEqual(t, f.x, 0.0)
		assert.Less(t, f.x, float64(state.width))
	}
}

func TestShenanigans_OnPickerShown_Plasma(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{"effect": "plasma"})

	mc := &mockPickerCanvas{}
	p.OnPickerShown(mc)

	assert.True(t, mc.overlayAdded)
	assert.NotNil(t, mc.drawFn)

	// Call the draw function to ensure it doesn't panic
	pixels := mc.drawFn(200, 100)
	assert.Len(t, pixels, 200*100*4)
}

func TestPlasmaState_Render(t *testing.T) {
	state := &plasmaState{time: 1.5}
	pixels := state.render(100, 50)

	assert.Len(t, pixels, 100*50*4)

	// Should have some non-zero color values
	hasColor := false
	for i := 0; i < len(pixels); i += 4 {
		if pixels[i] > 0 || pixels[i+1] > 0 || pixels[i+2] > 0 {
			hasColor = true
			break
		}
	}
	assert.True(t, hasColor)
}

func TestShenanigans_OnPickerShown_Starfield(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{"effect": "starfield"})

	mc := &mockPickerCanvas{}
	p.OnPickerShown(mc)

	assert.True(t, mc.overlayAdded)
	assert.NotNil(t, mc.drawFn)

	pixels := mc.drawFn(400, 300)
	assert.Len(t, pixels, 400*300*4)
}

func TestStarfieldState_Update(t *testing.T) {
	state := &starfieldState{width: 400, height: 300}
	state.init()

	assert.Len(t, state.stars, starCount)

	// All z values should be positive after init
	for _, s := range state.stars {
		assert.Greater(t, s.z, 0.0)
	}

	// After update, stars should have moved closer
	initialZ := state.stars[0].z
	state.update()
	assert.Less(t, state.stars[0].z, initialZ)
}

func TestShenanigans_OnPickerShown_Aurora(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{"effect": "aurora"})

	mc := &mockPickerCanvas{}
	p.OnPickerShown(mc)

	assert.True(t, mc.overlayAdded)
	assert.NotNil(t, mc.drawFn)

	pixels := mc.drawFn(400, 300)
	assert.Len(t, pixels, 400*300*4)
}

func TestShenanigans_OnPickerShown_Glitch(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{"effect": "glitch"})

	mc := &mockPickerCanvas{}
	p.OnPickerShown(mc)

	assert.True(t, mc.overlayAdded)
	assert.NotNil(t, mc.drawFn)

	// Glitch starts in non-burst state, should return empty pixels
	pixels := mc.drawFn(400, 300)
	assert.Len(t, pixels, 400*300*4)
}

func TestGlitchState_BurstCycle(t *testing.T) {
	state := &glitchState{}

	// Run enough updates to trigger a burst
	for range 100 {
		state.update()
	}

	// At some point during 100 frames, a burst should have occurred
	// (timer starts at 0, so first update triggers a burst)
	// Just verify it doesn't panic
	pixels := state.render(200, 100)
	assert.Len(t, pixels, 200*100*4)
}

func TestShenanigans_OnPickerShown_Pride(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{"effect": "pride"})

	mc := &mockPickerCanvas{}
	p.OnPickerShown(mc)

	assert.True(t, mc.overlayAdded)
	assert.NotNil(t, mc.drawFn)

	pixels := mc.drawFn(400, 300)
	assert.Len(t, pixels, 400*300*4)

	// Should have colorful pixels (not all black)
	hasColor := false
	for i := 0; i < len(pixels); i += 4 {
		if pixels[i] > 0 || pixels[i+1] > 0 || pixels[i+2] > 0 {
			hasColor = true
			break
		}
	}
	assert.True(t, hasColor)
}

func TestShenanigans_OnPickerShown_Football(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{"effect": "football"})

	mc := &mockPickerCanvas{}
	p.OnPickerShown(mc)

	assert.True(t, mc.overlayAdded)
	assert.NotNil(t, mc.drawFn)

	pixels := mc.drawFn(400, 300)
	assert.Len(t, pixels, 400*300*4)

	// Should have green pixels (pitch)
	hasGreen := false
	for i := 0; i < len(pixels); i += 4 {
		if pixels[i+1] > pixels[i] && pixels[i+1] > pixels[i+2] {
			hasGreen = true
			break
		}
	}
	assert.True(t, hasGreen)
}

func TestShenanigans_OnPickerShown_Fireworks(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{"effect": "fireworks"})

	mc := &mockPickerCanvas{}
	p.OnPickerShown(mc)

	assert.True(t, mc.overlayAdded)
	assert.NotNil(t, mc.drawFn)

	pixels := mc.drawFn(400, 300)
	assert.Len(t, pixels, 400*300*4)
}

func TestShenanigans_Shutdown(t *testing.T) {
	p := NewForTesting()
	_ = p.Setup(newTestServiceProvider(), map[string]interface{}{})

	p.Shutdown(context.Background())
	assert.True(t, p.stopped.Load())
}

func TestFireState_Update(t *testing.T) {
	state := &fireState{width: fireWidth, height: fireHeight}
	state.init()

	// Should not panic
	state.update()

	// Bottom row should have hot values (180+)
	for x := range state.width {
		assert.Greater(t, state.grid[state.height-1][x], uint8(179))
	}
}

func TestFireState_Render(t *testing.T) {
	state := &fireState{width: fireWidth, height: fireHeight}
	state.init()
	state.update()

	pixels := state.render(200, 100)
	assert.Len(t, pixels, 200*100*4)
}

func TestMatrixState_InitAndUpdate(t *testing.T) {
	state := &matrixState{width: 200, height: 400}
	state.initColumns()

	assert.NotEmpty(t, state.columns)

	// Should not panic
	state.update()
}

func TestMatrixState_Render(t *testing.T) {
	state := &matrixState{width: 200, height: 400}
	state.initColumns()
	state.update()

	pixels := state.render()
	assert.Len(t, pixels, 200*400*4)
}

func TestFireColor(t *testing.T) {
	// Low values should be transparent
	r, g, b, a := fireColor(0)
	assert.Equal(t, uint8(0), a)
	assert.Equal(t, uint8(0), r)
	assert.Equal(t, uint8(0), g)
	assert.Equal(t, uint8(0), b)

	// High values should have full red
	r, _, _, _ = fireColor(200)
	assert.Equal(t, uint8(255), r)
}
