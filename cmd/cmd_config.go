package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/strobotti/linkquisition"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or modify configuration",
	Long:  "View the full configuration, or get/set individual values.",
	RunE:  runConfigShow,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get a configuration value by key.

Available keys:
  locale       UI locale (e.g. "fi", "en", "es", "sv")
  logLevel     Log level (debug, info, warn, error)`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value by key.

Available keys:
  locale       UI locale (e.g. "fi", "en", "es", "sv")
  logLevel     Log level (debug, info, warn, error)`,
	Args: cobra.ExactArgs(2), //nolint:mnd
	RunE: runConfigSet,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the configuration file path",
	RunE:  runConfigPath,
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPathCmd)
}

func newSettingsServiceForCLI() linkquisition.SettingsService {
	_, settingsService := newPlatformServices()
	return settingsService
}

func runConfigShow(_ *cobra.Command, _ []string) error {
	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	fmt.Println(string(data))

	return nil
}

func runConfigGet(_ *cobra.Command, args []string) error {
	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	key := args[0]

	value, err := getSettingsValue(settings, key)
	if err != nil {
		return err
	}

	fmt.Println(value)

	return nil
}

func runConfigSet(_ *cobra.Command, args []string) error {
	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	key := args[0]
	value := args[1]

	if err := setSettingsValue(settings, key, value); err != nil {
		return err
	}

	if err := settingsService.WriteSettings(settings); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	fmt.Printf("%s = %s\n", key, value)

	return nil
}

func runConfigPath(_ *cobra.Command, _ []string) error {
	settingsService := newSettingsServiceForCLI()
	fmt.Println(settingsService.GetConfigFilePath())

	return nil
}

func getSettingsValue(settings *linkquisition.Settings, key string) (string, error) {
	switch strings.ToLower(key) {
	case "locale":
		return settings.Locale, nil
	case "loglevel":
		return settings.LogLevel, nil
	default:
		return "", fmt.Errorf("unknown configuration key: %s\nAvailable keys: locale, logLevel", key)
	}
}

func setSettingsValue(settings *linkquisition.Settings, key, value string) error {
	switch strings.ToLower(key) {
	case "locale":
		settings.Locale = value
	case "loglevel":
		validLevels := []string{
			linkquisition.LogLevelDebug,
			linkquisition.LogLevelInfo,
			linkquisition.LogLevelWarn,
			linkquisition.LogLevelError,
		}

		valueLower := strings.ToLower(value)

		valid := false
		for _, level := range validLevels {
			if valueLower == level {
				valid = true
				break
			}
		}

		if !valid {
			return fmt.Errorf("invalid log level %q, must be one of: %s", value, strings.Join(validLevels, ", "))
		}

		settings.LogLevel = valueLower
	default:
		return fmt.Errorf("unknown configuration key: %s\nAvailable keys: locale, logLevel", key)
	}

	return nil
}
