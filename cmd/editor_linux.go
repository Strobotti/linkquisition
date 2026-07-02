//go:build linux

package main

import "os/exec"

func openFileInEditor(path string) error {
	return exec.Command("xdg-open", path).Start()
}
