//go:build windows

package main

import "context"

// startURLWatcher is a no-op on Windows. URLs arrive via command-line arguments
// (Windows invokes the registered protocol handler with the URL as an argument),
// so there's no need to watch for incoming events while the configurator is open.
func (a *Application) startURLWatcher(_ context.Context) {}
