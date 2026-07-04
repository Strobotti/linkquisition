package main

//go:generate fyne bundle -o resources/bundled.go --package resources --name LinkquisitionIcon Icon.png

import (
	"log"
	"os"
)

var version string // Will be set by the build script Taskfile.build.yml

const exitCodePanic = 2

func main() {
	os.Exit(run())
}

func run() (exitCode int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("FATAL: unrecovered panic: %v\n", r)
			exitCode = exitCodePanic
		}
	}()

	return execute()
}
