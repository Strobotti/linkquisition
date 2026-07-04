package main

import (
	"testing"

	"github.com/strobotti/linkquisition"
)

func TestFindPluginIndex_ExactName(t *testing.T) {
	settings := &linkquisition.Settings{
		Plugins: []linkquisition.PluginSettings{
			{Path: "sanitize.so"},
			{Path: "defang.so"},
		},
	}

	idx := findPluginIndex(settings, "sanitize")
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}

	idx = findPluginIndex(settings, "defang")
	if idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}
}

func TestFindPluginIndex_WithExtension(t *testing.T) {
	settings := &linkquisition.Settings{
		Plugins: []linkquisition.PluginSettings{
			{Path: "sanitize.so"},
		},
	}

	idx := findPluginIndex(settings, "sanitize.so")
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestFindPluginIndex_CaseInsensitive(t *testing.T) {
	settings := &linkquisition.Settings{
		Plugins: []linkquisition.PluginSettings{
			{Path: "Sanitize.so"},
		},
	}

	idx := findPluginIndex(settings, "sanitize")
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestFindPluginIndex_WithDirectory(t *testing.T) {
	settings := &linkquisition.Settings{
		Plugins: []linkquisition.PluginSettings{
			{Path: "/usr/lib/linkquisition/plugins/sanitize.so"},
		},
	}

	idx := findPluginIndex(settings, "sanitize")
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestFindPluginIndex_NotFound(t *testing.T) {
	settings := &linkquisition.Settings{
		Plugins: []linkquisition.PluginSettings{
			{Path: "sanitize.so"},
		},
	}

	idx := findPluginIndex(settings, "nonexistent")
	if idx != -1 {
		t.Errorf("expected -1 for not found, got %d", idx)
	}
}

func TestPluginDisplayName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"sanitize.so", "sanitize"},
		{"defang.so", "defang"},
		{"/usr/lib/linkquisition/plugins/unwrap.so", "unwrap"},
		{"terminus", "terminus"},
	}

	for _, tt := range tests {
		got := pluginDisplayName(tt.path)
		if got != tt.expected {
			t.Errorf("pluginDisplayName(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestSetPluginDisabledState_Integration(t *testing.T) {
	svc := newTestSettingsService(t)

	settings := &linkquisition.Settings{
		Plugins: []linkquisition.PluginSettings{
			{Path: "sanitize.so", IsDisabled: false},
			{Path: "defang.so", IsDisabled: false},
		},
	}

	if err := svc.WriteSettings(settings); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	// Disable sanitize by re-reading and modifying
	settings, err := svc.ReadSettings()
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	idx := findPluginIndex(settings, "sanitize")
	if idx < 0 {
		t.Fatal("expected to find sanitize plugin")
	}

	settings.Plugins[idx].IsDisabled = true

	if err := svc.WriteSettings(settings); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	// Read back and verify
	settings, err = svc.ReadSettings()
	if err != nil {
		t.Fatalf("failed to re-read settings: %v", err)
	}

	if !settings.Plugins[0].IsDisabled {
		t.Error("expected sanitize plugin to be disabled")
	}

	if settings.Plugins[1].IsDisabled {
		t.Error("expected defang plugin to still be enabled")
	}
}
