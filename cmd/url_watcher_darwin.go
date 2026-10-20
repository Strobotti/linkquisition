//go:build darwin

package main

import (
	"context"
	"fmt"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/darwin"
)

// startURLWatcher starts a goroutine that listens for incoming URLs via Apple Events
// while the configurator is open. On macOS, if Linkquisition is already running (showing
// the settings window) and the user clicks a link, macOS delivers the URL to the running
// process via an Apple Event. Without this watcher, those URLs would be silently dropped.
//
// The goroutine exits when the provided context is cancelled (i.e. when the configurator closes).
func (a *Application) startURLWatcher(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case urlToOpen := <-darwin.URLChannel:
				a.handleIncomingURL(ctx, urlToOpen)
			}
		}
	}()
	return done
}

// handleIncomingURL processes a URL received while the app is already running.
// It runs plugin processing and either opens directly or spawns the browser picker.
func (a *Application) handleIncomingURL(ctx context.Context, urlToOpen string) {
	a.Logger.Debug(fmt.Sprintf("Received URL via Apple Event while configurator is open: %#q", urlToOpen))

	pluginCtx, cancel := context.WithTimeout(ctx, pluginProcessTimeout)
	defer cancel()

	processedURL, action, message := a.processPlugins(pluginCtx, urlToOpen)

	switch action {
	case linkquisition.ActionBlock:
		a.Logger.Info("URL blocked by plugin", "url", urlToOpen, "message", message)
		return
	case linkquisition.ActionWarn:
		// In background mode, skip the warning dialog and just open
		a.Logger.Warn("URL triggered warning, opening anyway (configurator active)", "url", processedURL)
		if err := a.openInFirstBrowser(processedURL); err != nil {
			a.Logger.Error("Failed to open incoming URL", "url", processedURL, "error", err)
		}
		return
	case linkquisition.ActionOpenDirect:
		if err := a.openInFirstBrowser(processedURL); err != nil {
			a.Logger.Error("Failed to open incoming URL", "url", processedURL, "error", err)
		}
		return
	case linkquisition.ActionContinue:
		// normal flow
	}

	// Try to match against browser rules first
	if isConfigured, _ := a.SettingsService.IsConfigured(); isConfigured {
		if browser, err := a.SettingsService.GetSettings().GetMatchingBrowser(processedURL); err == nil {
			a.Logger.Debug(fmt.Sprintf("Matched browser rule: %s", browser.Name), "url", processedURL)
			if err := a.BrowserService.OpenUrlWithBrowser(processedURL, browser); err != nil {
				a.Logger.Error("Failed to open incoming URL with matched browser", "url", processedURL, "browser", browser.Name, "error", err)
			}
			return
		}
	}

	// No rule matched — open in first available browser
	// (Opening a picker window while configurator is showing would be complex;
	// for now, use the first browser as a reasonable default)
	if err := a.openInFirstBrowser(processedURL); err != nil {
		a.Logger.Error("Failed to open incoming URL", "url", processedURL, "error", err)
	}
}
