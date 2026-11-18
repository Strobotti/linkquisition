//go:build !windows

package main

import (
    "os"

    "golang.org/x/sys/unix"
)

// trySourceLock prevents concurrent processes from downloading the same source.
func trySourceLock(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func unlockSource(file *os.File) { _ = unix.Flock(int(file.Fd()), unix.LOCK_UN); _ = file.Close() }
