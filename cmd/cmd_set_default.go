package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var setDefaultCmd = &cobra.Command{
	Use:   "set-default",
	Short: "Set Linkquisition as the default browser",
	Long: `Set Linkquisition as the default browser for the system.

On Linux, this sets the default-web-browser via xdg-settings.
On macOS, this registers Linkquisition as the handler for http/https
URLs (the OS may show a confirmation dialog).
On Windows, this registers Linkquisition as a URL handler and opens
the Default Apps settings page for the user to confirm.`,
	RunE: runSetDefault,
}

func runSetDefault(_ *cobra.Command, _ []string) error {
	browserService := newBrowserServiceForCLI()

	if browserService.AreWeTheDefaultBrowser() {
		fmt.Println("Linkquisition is already the default browser.")
		return nil
	}

	if err := browserService.MakeUsTheDefaultBrowser(); err != nil {
		return fmt.Errorf("failed to set as default browser: %w", err)
	}

	fmt.Println("Linkquisition is now the default browser.")

	return nil
}
