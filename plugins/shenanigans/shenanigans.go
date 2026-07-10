//nolint:gosec // Visual effects plugin: weak random is by design.
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
	effectTetris      = "tetris"
	effectFrogger     = "frogger"
	effectMinesweeper = "minesweeper"
	effectFlappy      = "flappy"
	effectLava        = "lava"
	effectSineScroll  = "sinescroll"
	effectFireflies   = "fireflies"
	effectBoids       = "boids"
	effectRaycast     = "raycast"
	effectPipes       = "pipes"
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
					effectAsteroids, effectAurora, effectBoids, effectBreakout,
					effectDino, effectFire, effectFireflies, effectFireworks,
					effectFlappy, effectFootball, effectFrogger, effectGlitch,
					effectInvaders, effectLava, effectLife, effectMatrix,
					effectMinesweeper, effectPacman, effectPipes, effectPlasma, effectPong,
					effectPride, effectRain, effectRaycast, effectSnake,
					effectSineScroll, effectSnow, effectStarfield, effectTetris,
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

func (p *shenanigans) OnPickerShown(canvas linkquisition.PickerCanvas) {
	p.lightMode = canvas.IsLightTheme()
	effect := p.effect

	allEffects := []string{
		effectMatrix, effectFire, effectSnow, effectPlasma,
		effectStarfield, effectAurora, effectGlitch, effectPride,
		effectFootball, effectFireworks, effectPong, effectLife,
		effectInvaders, effectSnake, effectRain, effectBreakout, effectDino,
		effectAsteroids, effectPacman, effectTetris, effectFrogger,
		effectMinesweeper, effectFlappy, effectLava, effectSineScroll,
		effectFireflies, effectBoids, effectRaycast, effectPipes,
	}

	if effect == effectRandom || !isKnownEffect(effect, allEffects) {
		effect = allEffects[rand.IntN(len(allEffects))]
	}

	p.serviceProvider.GetLogger().Debug("Shenanigans activating", "effect", effect)

	p.dispatchEffect(effect, canvas)
}

func (p *shenanigans) dispatchEffect(effect string, canvas linkquisition.PickerCanvas) {
	dispatch := map[string]func(linkquisition.PickerCanvas){
		effectMatrix:      p.startMatrixRain,
		effectFire:        p.startFire,
		effectSnow:        p.startSnow,
		effectPlasma:      p.startPlasma,
		effectStarfield:   p.startStarfield,
		effectAurora:      p.startAurora,
		effectGlitch:      p.startGlitch,
		effectPride:       p.startPride,
		effectFootball:    p.startFootball,
		effectFireworks:   p.startFireworks,
		effectPong:        p.startPong,
		effectLife:        p.startLife,
		effectInvaders:    p.startInvaders,
		effectSnake:       p.startSnake,
		effectRain:        p.startRain,
		effectBreakout:    p.startBreakout,
		effectDino:        p.startDino,
		effectAsteroids:   p.startAsteroids,
		effectPacman:      p.startPacman,
		effectTetris:      p.startTetris,
		effectFrogger:     p.startFrogger,
		effectMinesweeper: p.startMinesweeper,
		effectFlappy:      p.startFlappy,
		effectLava:        p.startLava,
		effectSineScroll:  p.startSineScroll,
		effectFireflies:   p.startFireflies,
		effectBoids:       p.startBoids,
		effectRaycast:     p.startRaycast,
		effectPipes:       p.startPipes,
	}

	if fn, ok := dispatch[effect]; ok {
		fn(canvas)
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

// NewForTesting creates a fresh shenanigans instance for use in tests,
// avoiding copying the package-level Plugin variable (which may contain a mutex).
func NewForTesting() *shenanigans {
	return &shenanigans{}
}

var Plugin shenanigans
