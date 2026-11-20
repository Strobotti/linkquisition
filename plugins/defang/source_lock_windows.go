//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func trySourceLock(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, sourceLockFilePerms)
	if err != nil {
		return nil, err
	}
	var overlapped windows.Overlapped
	const flags = windows.LOCKFILE_EXCLUSIVE_LOCK | windows.LOCKFILE_FAIL_IMMEDIATELY
	if err = windows.LockFileEx(windows.Handle(file.Fd()), flags, 0, 1, 0, &overlapped); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func unlockSource(file *os.File) {
	var overlapped windows.Overlapped
	_ = windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &overlapped)
	_ = file.Close()
}
