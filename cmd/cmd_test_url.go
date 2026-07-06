package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/strobotti/linkquisition"
)

const testURLTimeout = 30 * time.Second

var testURLCmd = &cobra.Command{
	Use:   "test-url <url>",
	Short: "Simulate opening a URL without actually opening it",
	Long: `Trace how a URL would be processed through the plugin chain and browser matching.

Shows each plugin's effect on the URL (modified, blocked, warned) and which browser
would be selected by the matching rules. Nothing is actually opened.`,
	Args: cobra.ExactArgs(1),
	RunE: runTestURL,
}

func runTestURL(_ *cobra.Command, args []string) error {
	urlToOpen := args[0]

	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	// Set up a logger that writes to stdout for tracing
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pluginServiceProvider := linkquisition.NewPluginServiceProvider(
		logger, settings, settingsService.GetConfigFolderPath(),
	)

	plugins := setupPlugins(settingsService, pluginServiceProvider, logger)

	fmt.Printf("Input URL: %s\n", urlToOpen)
	fmt.Printf("Plugins:   %d loaded\n\n", len(plugins))

	ctx, cancel := context.WithTimeout(context.Background(), testURLTimeout)
	defer cancel()

	currentURL := urlToOpen
	var finalAction linkquisition.PluginAction
	var finalMessage string

	for i, plug := range plugins {
		meta := plug.Metadata()
		result := plug.ProcessURL(ctx, currentURL)

		switch result.Action {
		case linkquisition.ActionContinue:
			if result.URL != currentURL {
				fmt.Printf("  [%d] %s: %s → %s\n", i+1, meta.Name, currentURL, result.URL)
				currentURL = result.URL
			} else {
				fmt.Printf("  [%d] %s: (unchanged)\n", i+1, meta.Name)
			}
		case linkquisition.ActionBlock:
			fmt.Printf("  [%d] %s: ✗ BLOCKED\n", i+1, meta.Name)
			fmt.Printf("      Message: %s\n", singleLine(result.Message))
			finalAction = linkquisition.ActionBlock
			finalMessage = result.Message
		case linkquisition.ActionWarn:
			fmt.Printf("  [%d] %s: ⚠ WARN\n", i+1, meta.Name)
			fmt.Printf("      Message: %s\n", singleLine(result.Message))
			finalAction = linkquisition.ActionWarn
			finalMessage = result.Message
			currentURL = result.URL
		case linkquisition.ActionOpenDirect:
			fmt.Printf("  [%d] %s: → OPEN DIRECT\n", i+1, meta.Name)
			currentURL = result.URL
			finalAction = linkquisition.ActionOpenDirect
		}

		// Stop the chain if the plugin says so
		if result.Action != linkquisition.ActionContinue && !result.ContinueChain {
			fmt.Printf("      (chain stopped)\n")
			break
		}
	}

	fmt.Println()

	// Show final outcome
	switch finalAction {
	case linkquisition.ActionBlock:
		fmt.Printf("Result: BLOCKED\n")
		fmt.Printf("  %s\n", singleLine(finalMessage))
		return nil
	case linkquisition.ActionWarn:
		fmt.Printf("Result: WARN (user would be prompted)\n")
		fmt.Printf("  %s\n", singleLine(finalMessage))
		fmt.Printf("  URL if user proceeds: %s\n", currentURL)
	case linkquisition.ActionOpenDirect:
		fmt.Printf("Result: OPEN DIRECT (bypass browser matching)\n")
		fmt.Printf("  URL: %s\n", currentURL)
		return nil
	default:
		// ActionContinue — proceed to browser matching
		if currentURL != urlToOpen {
			fmt.Printf("Final URL: %s\n", currentURL)
		}
	}

	// Browser matching
	fmt.Println()
	fmt.Println("Browser matching:")

	if browser, err := settings.GetMatchingBrowser(currentURL); err == nil {
		fmt.Printf("  ✓ Matched: %s\n", browser.Name)
		fmt.Printf("    Command: %s\n", browser.Command)
	} else {
		fmt.Println("  ✗ No rule matched — browser picker would be shown")

		selectable := settings.GetSelectableBrowsers()
		if len(selectable) > 0 {
			fmt.Println("  Available browsers:")
			for i, b := range selectable {
				fmt.Printf("    [%d] %s\n", i+1, b.Name)
			}
		}
	}

	return nil
}

// singleLine replaces newlines with spaces for compact CLI output
func singleLine(s string) string {
	result := make([]byte, 0, len(s))
	for i := range len(s) {
		if s[i] == '\n' {
			if len(result) > 0 && result[len(result)-1] != ' ' {
				result = append(result, ' ')
			}
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}
