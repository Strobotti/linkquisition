//go:build linux

package main

import "context"

// startURLWatcher is a no-op on Linux. URLs arrive via command-line arguments,
// so there's no need to watch for incoming events while the configurator is open.
func (a *Application) startURLWatcher(_ context.Context) {}
