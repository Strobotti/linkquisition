package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const cmdUseList = "list"

var browsersCmd = &cobra.Command{
	Use:   "browsers",
	Short: "List and manage browsers",
	Long:  "List configured browsers or scan the system for available browsers.",
	RunE:  runBrowsersList,
}

var browsersListCmd = &cobra.Command{
	Use:   cmdUseList,
	Short: "List configured browsers",
	RunE:  runBrowsersList,
}

var browsersScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan the system for available browsers",
	Long: `Scan the system for installed browsers and update the configuration.
Equivalent to clicking "Scan browsers" in the configurator GUI.

Manually added browsers (source: "manual") and custom match rules
are preserved during re-scanning.`,
	RunE: runBrowsersScan,
}

func initBrowsersCmd() {
	browsersCmd.AddCommand(browsersListCmd)
	browsersCmd.AddCommand(browsersScanCmd)
}

func runBrowsersList(_ *cobra.Command, _ []string) error {
	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	if len(settings.Browsers) == 0 {
		fmt.Println("No browsers configured. Run \"linkquisition browsers scan\" to detect installed browsers.")
		return nil
	}

	for i, b := range settings.Browsers {
		status := ""
		if b.Hidden {
			status = " (hidden)"
		}

		source := ""
		if b.Source == "manual" {
			source = " [manual]"
		}

		rules := ""
		if len(b.Matches) > 0 {
			rules = fmt.Sprintf(" (%d rules)", len(b.Matches))
		}

		fmt.Printf("  %d. %-25s %s%s%s%s\n", i+1, b.Name, b.Command, status, source, rules)
	}

	return nil
}

func runBrowsersScan(_ *cobra.Command, _ []string) error {
	settingsService := newSettingsServiceForCLI()

	if err := settingsService.ScanBrowsers(); err != nil {
		return fmt.Errorf("browser scan failed: %w", err)
	}

	fmt.Println("Browser scan complete.")

	// Show the result
	settings := settingsService.GetSettings()
	visible := 0
	hidden := 0

	for _, b := range settings.Browsers {
		if b.Hidden {
			hidden++
		} else {
			visible++
		}
	}

	fmt.Printf("Found %d browser(s) (%d visible, %d hidden).\n", len(settings.Browsers), visible, hidden)

	return nil
}
