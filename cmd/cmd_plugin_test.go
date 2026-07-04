package main

import (
	"os"
	"path/filepath"
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

	err = svc.WriteSettings(settings)
	if err != nil {
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

func TestDiscoverAvailablePlugins_FindsUnconfigured(t *testing.T) {
	svc := newTestSettingsService(t)
	pluginDir := svc.GetPluginFolderPath()

	// Create some .so files in the plugin folder
	for _, name := range []string{"sanitize.so", "defang.so", "unwrap.so"} {
		if err := os.WriteFile(filepath.Join(pluginDir, name), []byte("fake"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	// Only sanitize is configured
	settings := &linkquisition.Settings{
		Plugins: []linkquisition.PluginSettings{
			{Path: "sanitize.so"},
		},
	}

	available := discoverAvailablePlugins(svc, settings)

	if len(available) != 2 {
		t.Fatalf("expected 2 available plugins, got %d: %v", len(available), available)
	}

	// Should contain defang and unwrap but not sanitize
	found := map[string]bool{}
	for _, name := range available {
		found[name] = true
	}

	if found["sanitize"] {
		t.Error("sanitize should not be in available list (already configured)")
	}

	if !found["defang"] {
		t.Error("defang should be in available list")
	}

	if !found["unwrap"] {
		t.Error("unwrap should be in available list")
	}
}

func TestDiscoverAvailablePlugins_IgnoresNonSoFiles(t *testing.T) {
	svc := newTestSettingsService(t)
	pluginDir := svc.GetPluginFolderPath()

	// Create mixed files
	for _, name := range []string{"sanitize.so", "readme.txt", ".hidden"} {
		if err := os.WriteFile(filepath.Join(pluginDir, name), []byte("fake"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	settings := &linkquisition.Settings{}
	available := discoverAvailablePlugins(svc, settings)

	if len(available) != 1 {
		t.Fatalf("expected 1 available plugin, got %d: %v", len(available), available)
	}

	if available[0] != "sanitize" {
		t.Errorf("expected 'sanitize', got %q", available[0])
	}
}

func TestDiscoverAvailablePlugins_EmptyDir(t *testing.T) {
	svc := newTestSettingsService(t)
	settings := &linkquisition.Settings{}

	available := discoverAvailablePlugins(svc, settings)

	if len(available) != 0 {
		t.Errorf("expected empty list, got %v", available)
	}
}

func TestDiscoverAvailablePlugins_NonexistentDir(t *testing.T) {
	// Use a path provider pointing to a dir that doesn't exist
	svc := &linkquisition.FileSettingsService{
		PathProvider: &testPathProvider{
			configFolder: "/nonexistent/path",
			logFolder:    "/nonexistent/path",
			pluginFolder: "/nonexistent/path/plugins",
		},
	}
	settings := &linkquisition.Settings{}

	available := discoverAvailablePlugins(svc, settings)

	if available != nil {
		t.Errorf("expected nil for nonexistent dir, got %v", available)
	}
}
