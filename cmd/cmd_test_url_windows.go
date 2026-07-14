//go:build windows

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/strobotti/linkquisition"
)

var testURLCmd = &cobra.Command{
	Use:   "test-url <url>",
	Short: "Simulate opening a URL without actually opening it",
	Long: `Trace how a URL would be processed through browser matching rules.

Shows which browser would be selected by the matching rules. Nothing is actually opened.
Note: Plugins are not supported on Windows.`,
	Args: cobra.ExactArgs(1),
	RunE: runTestURL,
}

func runTestURL(_ *cobra.Command, args []string) error {
	urlToOpen := args[0]

	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	fmt.Printf("Input URL: %s\n", urlToOpen)
	fmt.Println("Plugins:   not supported on Windows")
	fmt.Println()

	printBrowserMatch(settings, urlToOpen)

	return nil
}

func printBrowserMatch(settings *linkquisition.Settings, url string) {
	fmt.Println("Browser matching:")

	if browser, err := settings.GetMatchingBrowser(url); err == nil {
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
