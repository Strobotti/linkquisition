//go:build !windows

package main

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/strobotti/linkquisition"
)

func TestProcessPlugins_NoPlugins(t *testing.T) {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	url, action, message := app.processPlugins(context.Background(), "https://example.com")

	if url != "https://example.com" {
		t.Errorf("expected original URL, got %q", url)
	}
	if action != linkquisition.ActionContinue {
		t.Errorf("expected ActionContinue, got %v", action)
	}
	if message != "" {
		t.Errorf("expected empty message, got %q", message)
	}
}

func TestProcessPlugins_SinglePlugin_Continue(t *testing.T) {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		plugins: []linkquisition.Plugin{
			&testPlugin{
				name: "sanitize",
				result: linkquisition.PluginResult{
					URL:    "https://example.com/clean",
					Action: linkquisition.ActionContinue,
				},
			},
		},
	}

	url, action, message := app.processPlugins(context.Background(), "https://example.com/dirty")

	if url != "https://example.com/clean" {
		t.Errorf("expected modified URL, got %q", url)
	}
	if action != linkquisition.ActionContinue {
		t.Errorf("expected ActionContinue, got %v", action)
	}
	if message != "" {
		t.Errorf("expected empty message, got %q", message)
	}
}

func TestProcessPlugins_SinglePlugin_Block(t *testing.T) {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		plugins: []linkquisition.Plugin{
			&testPlugin{
				name: "defang",
				result: linkquisition.PluginResult{
					URL:     "https://malware.com",
					Action:  linkquisition.ActionBlock,
					Message: "Known malware",
				},
			},
		},
	}

	url, action, message := app.processPlugins(context.Background(), "https://malware.com")

	if url != "https://malware.com" {
		t.Errorf("expected plugin URL, got %q", url)
	}
	if action != linkquisition.ActionBlock {
		t.Errorf("expected ActionBlock, got %v", action)
	}
	if message != "Known malware" {
		t.Errorf("expected message 'Known malware', got %q", message)
	}
}

func TestProcessPlugins_SinglePlugin_Warn(t *testing.T) {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		plugins: []linkquisition.Plugin{
			&testPlugin{
				name: "shenanigans",
				result: linkquisition.PluginResult{
					URL:     "https://phishy.com",
					Action:  linkquisition.ActionWarn,
					Message: "Looks phishy",
				},
			},
		},
	}

	url, action, message := app.processPlugins(context.Background(), "https://phishy.com")

	if url != "https://phishy.com" {
		t.Errorf("expected URL, got %q", url)
	}
	if action != linkquisition.ActionWarn {
		t.Errorf("expected ActionWarn, got %v", action)
	}
	if message != "Looks phishy" {
		t.Errorf("expected message, got %q", message)
	}
}

func TestProcessPlugins_SinglePlugin_OpenDirect(t *testing.T) {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		plugins: []linkquisition.Plugin{
			&testPlugin{
				name: "terminus",
				result: linkquisition.PluginResult{
					URL:    "https://internal.com",
					Action: linkquisition.ActionOpenDirect,
				},
			},
		},
	}

	url, action, message := app.processPlugins(context.Background(), "https://internal.com")

	if url != "https://internal.com" {
		t.Errorf("expected URL, got %q", url)
	}
	if action != linkquisition.ActionOpenDirect {
		t.Errorf("expected ActionOpenDirect, got %v", action)
	}
	if message != "" {
		t.Errorf("expected empty message, got %q", message)
	}
}

func TestProcessPlugins_ChainStopsOnBlock(t *testing.T) {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		plugins: []linkquisition.Plugin{
			&testPlugin{
				name: "blocker",
				result: linkquisition.PluginResult{
					URL:     "https://evil.com",
					Action:  linkquisition.ActionBlock,
					Message: "blocked",
				},
			},
			&testPlugin{
				name: "should-not-run",
				result: linkquisition.PluginResult{
					URL:    "https://evil.com/modified-by-second",
					Action: linkquisition.ActionContinue,
				},
			},
		},
	}

	url, action, _ := app.processPlugins(context.Background(), "https://evil.com")

	if action != linkquisition.ActionBlock {
		t.Errorf("expected ActionBlock, got %v", action)
	}
	if url == "https://evil.com/modified-by-second" {
		t.Error("second plugin should not have run")
	}
}

func TestProcessPlugins_MultiplePlugins_Chained(t *testing.T) {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		plugins: []linkquisition.Plugin{
			&testPlugin{
				name: "step1",
				result: linkquisition.PluginResult{
					URL:    "https://example.com/step1",
					Action: linkquisition.ActionContinue,
				},
			},
			&testPlugin{
				name: "step2",
				result: linkquisition.PluginResult{
					URL:    "https://example.com/step2",
					Action: linkquisition.ActionContinue,
				},
			},
		},
	}

	url, action, _ := app.processPlugins(context.Background(), "https://example.com/original")

	if url != "https://example.com/step2" {
		t.Errorf("expected final URL from step2, got %q", url)
	}
	if action != linkquisition.ActionContinue {
		t.Errorf("expected ActionContinue, got %v", action)
	}
}

func TestCollectUIHooks_NoPlugins(t *testing.T) {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	hooks := app.collectUIHooks()
	if len(hooks) != 0 {
		t.Errorf("expected no hooks, got %d", len(hooks))
	}
}

func TestCollectUIHooks_NoHookPlugins(t *testing.T) {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		plugins: []linkquisition.Plugin{
			&testPlugin{name: "basic"},
		},
	}

	hooks := app.collectUIHooks()
	if len(hooks) != 0 {
		t.Errorf("expected no hooks from basic plugin, got %d", len(hooks))
	}
}

func TestShutdownPlugins_Empty(t *testing.T) {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	// Should not panic with no plugins
	app.shutdownPlugins()
	t.Log("shutdownPlugins with no plugins did not panic")
}

func TestShutdownPlugins_WithPlugins(t *testing.T) {
	shutdownCalled := false

	plug := &shutdownTrackingPlugin{
		testPlugin: testPlugin{name: "tracker"},
		onShutdown: func() { shutdownCalled = true },
	}

	app := &Application{
		Logger:  slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		plugins: []linkquisition.Plugin{plug},
	}

	app.shutdownPlugins()

	if !shutdownCalled {
		t.Error("expected Shutdown to be called on the plugin")
	}
}

// shutdownTrackingPlugin tracks whether Shutdown was called.
type shutdownTrackingPlugin struct {
	testPlugin
	onShutdown func()
}

func (p *shutdownTrackingPlugin) Shutdown(_ context.Context) {
	if p.onShutdown != nil {
		p.onShutdown()
	}
}
