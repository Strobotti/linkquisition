//go:build darwin

package main

import (
	"context"
	"os/exec"
)

func openFileInEditor(path string) error {
	return exec.CommandContext(context.Background(), "open", "-t", path).Start()
}
