package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/strobotti/linkquisition"
)

const (
	statusEnabled  = "enabled"
	statusDisabled = "disabled"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long:  "List, enable, disable, or add plugins.",
	RunE:  runPluginList,
}

var pluginListCmd = &cobra.Command{
	Use:   cmdUseList,
	Short: "List configured and available plugins",
	RunE:  runPluginList,
}

var pluginEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a plugin",
	Long: `Enable a plugin by name (e.g. "sanitize", "defang").
The name is matched against the plugin path without the .so extension.`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginEnable,
}

var pluginDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a plugin",
	Long: `Disable a plugin by name (e.g. "sanitize", "defang").
The name is matched against the plugin path without the .so extension.`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginDisable,
}

var pluginAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add an available plugin to the configuration",
	Long: `Add a plugin that exists in the plugin folder but is not yet configured.
Use "plugin list" to see available plugins. The plugin is added with default settings.`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginAdd,
}

func initPluginCmd() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
	pluginCmd.AddCommand(pluginAddCmd)
}

func runPluginList(_ *cobra.Command, _ []string) error {
	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	if len(settings.Plugins) > 0 {
		fmt.Println("Configured plugins:")
		for _, p := range settings.Plugins {
			status := statusEnabled
			if p.IsDisabled {
				status = statusDisabled
			}

			name := pluginDisplayName(p.Path)
			fmt.Printf("  %-20s %s\n", name, status)
		}
	} else {
		fmt.Println("No plugins configured.")
	}

	// Show available but unconfigured plugins from the plugin folder
	available := discoverAvailablePlugins(settingsService, settings)
	if len(available) > 0 {
		fmt.Println("\nAvailable (not configured):")
		for _, name := range available {
			fmt.Printf("  %-20s (use \"plugin add %s\" to configure)\n", name, name)
		}
	}

	return nil
}

func runPluginEnable(_ *cobra.Command, args []string) error {
	return setPluginDisabledState(args[0], false)
}

func runPluginDisable(_ *cobra.Command, args []string) error {
	return setPluginDisabledState(args[0], true)
}

func setPluginDisabledState(name string, disabled bool) error {
	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	idx := findPluginIndex(settings, name)
	if idx < 0 {
		available := make([]string, 0, len(settings.Plugins))
		for _, p := range settings.Plugins {
			available = append(available, pluginDisplayName(p.Path))
		}

		return fmt.Errorf(
			"plugin %q not found\nConfigured plugins: %s",
			name,
			strings.Join(available, ", "),
		)
	}

	settings.Plugins[idx].IsDisabled = disabled

	if err := settingsService.WriteSettings(settings); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	action := statusEnabled
	if disabled {
		action = statusDisabled
	}

	fmt.Printf("%s: %s\n", pluginDisplayName(settings.Plugins[idx].Path), action)

	return nil
}

// findPluginIndex returns the index of the plugin matching the given name, or -1.
// Matches against the base name of the path with or without .so extension.
func findPluginIndex(settings *linkquisition.Settings, name string) int {
	name = strings.TrimSuffix(strings.ToLower(name), ".so")

	for i, p := range settings.Plugins {
		pluginName := strings.TrimSuffix(strings.ToLower(filepath.Base(p.Path)), ".so")
		if pluginName == name {
			return i
		}
	}

	return -1
}

// pluginDisplayName extracts a human-readable name from the plugin path.
func pluginDisplayName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".so")
}

// discoverAvailablePlugins scans the plugin folder for .so files that are not
// already in the settings, returning their base names without the .so extension.
func discoverAvailablePlugins(
	settingsService linkquisition.SettingsService,
	settings *linkquisition.Settings,
) []string {
	pluginDir := settingsService.GetPluginFolderPath()

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil
	}

	// Build a set of already-configured plugin names
	configured := make(map[string]bool, len(settings.Plugins))
	for _, p := range settings.Plugins {
		configured[strings.ToLower(pluginDisplayName(p.Path))] = true
	}

	var available []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".so") {
			continue
		}

		baseName := strings.TrimSuffix(name, ".so")
		if !configured[strings.ToLower(baseName)] {
			available = append(available, baseName)
		}
	}

	return available
}

func runPluginAdd(_ *cobra.Command, args []string) error {
	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	name := args[0]
	normalizedName := strings.TrimSuffix(strings.ToLower(name), ".so")

	// Check if already configured
	if idx := findPluginIndex(settings, name); idx >= 0 {
		return fmt.Errorf("plugin %q is already configured (use \"plugin enable/disable\" to change its state)", name)
	}

	// Verify the plugin file exists in the plugin folder
	pluginDir := settingsService.GetPluginFolderPath()
	pluginFile := normalizedName + ".so"
	pluginPath := filepath.Join(pluginDir, pluginFile)

	if _, err := os.Stat(pluginPath); err != nil {
		available := discoverAvailablePlugins(settingsService, settings)
		if len(available) > 0 {
			return fmt.Errorf(
				"plugin file %q not found in %s\nAvailable plugins: %s",
				pluginFile,
				pluginDir,
				strings.Join(available, ", "),
			)
		}

		return fmt.Errorf("plugin file %q not found in %s", pluginFile, pluginDir)
	}

	// Probe plugin metadata to get default settings
	var pluginSettings map[string]interface{}
	discardLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	meta, err := probePluginMetadata(pluginPath, discardLogger)
	if err == nil && len(meta.Settings) > 0 {
		pluginSettings = buildDefaultSettingsFromMetadata(&meta)
	}

	settings.Plugins = append(settings.Plugins, linkquisition.PluginSettings{
		Path:       pluginFile,
		IsDisabled: false,
		Settings:   pluginSettings,
	})

	if err := settingsService.WriteSettings(settings); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	fmt.Printf("%s: added and enabled\n", normalizedName)

	return nil
}
