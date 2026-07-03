package main

//go:generate fyne bundle -o resources/bundled.go --package resources --name LinkquisitionIcon Icon.png

import (
	"context"
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

	ctx, stop := context.WithCancel(context.Background())

	app := NewApplication()

	if err := app.Run(ctx); err != nil {
		log.Println(
			"main: app.Run returned an error",
			"error", err.Error(),
		)

		stop()
		<-ctx.Done()

		return 1
	}

	stop()
	<-ctx.Done()

	return 0
}
