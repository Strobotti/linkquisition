//go:build windows

package main

import (
	"os"
	"syscall"
)

const attachParentProcess = ^uintptr(0) // ATTACH_PARENT_PROCESS = (DWORD)-1

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole = kernel32.NewProc("AttachConsole")
)

// attachParentConsoleIfCLI attaches to the parent process's console so that
// stdout/stderr are visible when the GUI-subsystem binary is launched from a
// terminal (PowerShell, cmd). This enables --help, --version, and subcommand
// output to appear in the calling console.
//
// Must be called before any output is written (i.e. at the top of main).
// No-ops silently if there is no parent console (e.g. launched from Explorer).
func attachParentConsoleIfCLI() {
	// Only bother if there are arguments that look like CLI usage.
	// When launched with no args or a URL arg, skip — we don't need a console.
	if len(os.Args) <= 1 {
		return
	}

	// If the first (and only) arg looks like a URL, skip.
	if len(os.Args) == 2 && !isFlag(os.Args[1]) {
		return
	}

	ret, _, _ := procAttachConsole.Call(attachParentProcess)
	if ret == 0 {
		// Failed to attach (no parent console) — nothing to do.
		return
	}

	// Reopen stdout and stderr to the attached console.
	conout, err := syscall.Open("CONOUT$", syscall.O_RDWR, 0)
	if err != nil {
		return
	}

	os.Stdout = os.NewFile(uintptr(conout), "CONOUT$")
	os.Stderr = os.NewFile(uintptr(conout), "CONOUT$")
}

// isFlag returns true if the argument starts with "-" or "/".
func isFlag(arg string) bool {
	if len(arg) == 0 {
		return false
	}
	return arg[0] == '-' || arg[0] == '/'
}
