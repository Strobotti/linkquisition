//go:build !windows

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

type testURLResult struct {
	url     string
	action  linkquisition.PluginAction
	message string
}

func runTestURL(_ *cobra.Command, args []string) error {
	urlToOpen := args[0]

	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pluginServiceProvider := linkquisition.NewPluginServiceProvider(
		logger, settings, settingsService.GetConfigFolderPath(),
	)

	plugins := setupPlugins(settingsService, pluginServiceProvider, logger, parsePluginOpts(pluginOpts))

	fmt.Printf("Input URL: %s\n", urlToOpen)
	fmt.Printf("Plugins:   %d loaded\n\n", len(plugins))

	ctx, cancel := context.WithTimeout(context.Background(), testURLTimeout)
	defer cancel()

	result := tracePlugins(ctx, plugins, urlToOpen)

	fmt.Println()
	printOutcome(result, urlToOpen)
	printBrowserMatch(settings, result)

	return nil
}

func tracePlugins(
	ctx context.Context, plugins []linkquisition.Plugin, inputURL string,
) testURLResult {
	currentURL := inputURL
	var finalAction linkquisition.PluginAction
	var finalMessage string

	for i, plug := range plugins {
		meta := plug.Metadata()
		r := plug.ProcessURL(ctx, currentURL)

		printPluginStep(i+1, meta.Name, r, currentURL)

		switch r.Action {
		case linkquisition.ActionContinue:
			currentURL = r.URL
		case linkquisition.ActionBlock:
			finalAction = linkquisition.ActionBlock
			finalMessage = r.Message
		case linkquisition.ActionWarn:
			finalAction = linkquisition.ActionWarn
			finalMessage = r.Message
			currentURL = r.URL
		case linkquisition.ActionOpenDirect:
			finalAction = linkquisition.ActionOpenDirect
			currentURL = r.URL
		}

		if r.Action != linkquisition.ActionContinue && !r.ContinueChain {
			fmt.Printf("      (chain stopped)\n")
			break
		}
	}

	return testURLResult{url: currentURL, action: finalAction, message: finalMessage}
}

func printPluginStep(index int, name string, r linkquisition.PluginResult, prevURL string) {
	switch r.Action {
	case linkquisition.ActionContinue:
		if r.URL != prevURL {
			fmt.Printf("  [%d] %s: %s → %s\n", index, name, prevURL, r.URL)
		} else {
			fmt.Printf("  [%d] %s: (unchanged)\n", index, name)
		}
	case linkquisition.ActionBlock:
		fmt.Printf("  [%d] %s: ✗ BLOCKED\n", index, name)
		fmt.Printf("      Message: %s\n", singleLine(r.Message))
	case linkquisition.ActionWarn:
		fmt.Printf("  [%d] %s: ⚠ WARN\n", index, name)
		fmt.Printf("      Message: %s\n", singleLine(r.Message))
	case linkquisition.ActionOpenDirect:
		fmt.Printf("  [%d] %s: → OPEN DIRECT\n", index, name)
	}
}

func printOutcome(result testURLResult, inputURL string) {
	switch result.action {
	case linkquisition.ActionBlock:
		fmt.Printf("Result: BLOCKED\n")
		fmt.Printf("  %s\n", singleLine(result.message))
	case linkquisition.ActionWarn:
		fmt.Printf("Result: WARN (user would be prompted)\n")
		fmt.Printf("  %s\n", singleLine(result.message))
		fmt.Printf("  URL if user proceeds: %s\n", result.url)
	case linkquisition.ActionOpenDirect:
		fmt.Printf("Result: OPEN DIRECT (bypass browser matching)\n")
		fmt.Printf("  URL: %s\n", result.url)
	case linkquisition.ActionContinue:
		if result.url != inputURL {
			fmt.Printf("Final URL: %s\n", result.url)
		}
	}
}

func printBrowserMatch(settings *linkquisition.Settings, result testURLResult) {
	if result.action == linkquisition.ActionBlock || result.action == linkquisition.ActionOpenDirect {
		return
	}

	fmt.Println()
	fmt.Println("Browser matching:")

	if browser, err := settings.GetMatchingBrowser(result.url); err == nil {
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
