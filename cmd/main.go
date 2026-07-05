package main

//go:generate fyne bundle -o resources/bundled.go --package resources --name LinkquisitionIcon Icon.png

import (
	"log"
	"os"
)

// version is set at build time via -ldflags '-X main.version=...' (see Taskfile.build.yml).
// The default "dev" ensures --version always works during development.
var version = "dev"

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
