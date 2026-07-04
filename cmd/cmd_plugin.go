package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/strobotti/linkquisition"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long:  "List, enable, or disable plugins.",
	RunE:  runPluginList,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured plugins",
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

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
}

func runPluginList(_ *cobra.Command, _ []string) error {
	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	if len(settings.Plugins) == 0 {
		fmt.Println("No plugins configured.")
		return nil
	}

	for _, p := range settings.Plugins {
		status := "enabled"
		if p.IsDisabled {
			status = "disabled"
		}

		name := pluginDisplayName(p.Path)
		fmt.Printf("  %-20s %s\n", name, status)
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

	action := "enabled"
	if disabled {
		action = "disabled"
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
