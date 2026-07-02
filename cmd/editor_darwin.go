//go:build darwin

package main

import "os/exec"

func openFileInEditor(path string) error {
	return exec.Command("open", "-t", path).Start()
}
