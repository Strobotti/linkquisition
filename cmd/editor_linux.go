//go:build linux

package main

import (
	"context"
	"os/exec"
)

func openFileInEditor(path string) error {
	return exec.CommandContext(context.Background(), "xdg-open", path).Start()
}
