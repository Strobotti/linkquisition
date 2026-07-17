package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	appAuthor    = "Juha Jantunen (@Strobotti)"
	appGithubURL = "https://github.com/Strobotti/linkquisition"
)

// pluginOpts holds runtime plugin setting overrides from --plugin-opt flags.
// Format: "pluginname.key=value" → map[pluginname]map[key]value
var pluginOpts []string

// noPlugins disables all plugin loading for debugging.
var noPlugins bool

// logLevelOverride temporarily overrides the log level from config (not persisted).
var logLevelOverride string

var rootCmd = &cobra.Command{
	Use:   "linkquisition [url]",
	Short: "A fast, configurable browser-picker",
	Long: fmt.Sprintf(`Linkquisition - Nobody expects the Linkquisition!

A fast, configurable browser-picker for Linux, macOS, and Windows.
Automatically chooses a browser based on domain, site, or regex rules.

Author:  %s
GitHub:  %s`, appAuthor, appGithubURL),
	Args:              cobra.MaximumNArgs(1),
	SilenceUsage:      true,
	SilenceErrors:     true,
	CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
	RunE:              runRoot,
}

func runRoot(cmd *cobra.Command, args []string) error {
	var urlToOpen string

	if len(args) == 1 {
		urlToOpen = args[0]
	} else {
		// No CLI args — check if a URL arrived via platform event (macOS Apple Events)
		urlToOpen = getURLFromPlatformEvent()
	}

	if urlToOpen != "" {
		urlToOpen = normalizeInputURL(urlToOpen)
	}

	// GUI path — needs full application with fyne
	app := NewApplication()
	ctx, stop := context.WithCancel(cmd.Context())

	if err := app.RunGUI(ctx, urlToOpen); err != nil {
		stop()
		<-ctx.Done()

		return err
	}

	stop()
	<-ctx.Done()

	return nil
}

// normalizeInputURL converts raw file paths to file:// URLs and validates the input.
// It accepts:
// - http:// and https:// URLs (passed through)
// - file:// URLs (passed through)
// - Absolute file paths (converted to file:// URLs)
// - Relative file paths (resolved to absolute, then converted to file:// URLs)
func normalizeInputURL(input string) string {
	// Already a valid URL with a scheme — pass through
	if parsed, err := url.ParseRequestURI(input); err == nil && parsed.Scheme != "" {
		return input
	}

	// Looks like a file path — convert to file:// URL
	if strings.HasPrefix(input, "/") || strings.HasPrefix(input, "~") || isRelativePath(input) {
		absPath, err := filepath.Abs(input)
		if err != nil {
			return input
		}
		return "file://" + absPath
	}

	return input
}

// isRelativePath returns true if the input looks like a relative file path
// (contains path separators or common file extensions, and doesn't look like a URL).
func isRelativePath(input string) bool {
	// If it contains a scheme-like prefix, it's not a path
	if strings.Contains(input, "://") {
		return false
	}

	// If the file exists on disk, it's definitely a path
	if _, err := os.Stat(input); err == nil {
		return true
	}

	return false
}

func initRootCmd() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("Version: {{.Version}}\n")

	rootCmd.Flags().StringVar(&logLevelOverride, "log-level", "",
		`override log level for this run without changing config (debug, info, warn, error)`)

	initConfigCmd()
	initBrowsersCmd()
	initRuleCmd()

	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(browsersCmd)
	rootCmd.AddCommand(setDefaultCmd)
	rootCmd.AddCommand(ruleCmd)
	rootCmd.AddCommand(testURLCmd)

	// Register plugin-related commands and flags (no-op on Windows)
	initPluginSupport()
}

func execute() int {
	initRootCmd()

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}
