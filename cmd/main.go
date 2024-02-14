package main

//go:generate fyne bundle -o resources/bundled.go --package resources --name LinkquisitionIcon Icon.png

import (
	"context"
	"log"
	"os"
)

var version string // Will be set by the build script Taskfile.build.yml

func main() {
	ctx, stop := context.WithCancel(context.Background())

	app := NewApplication()
	exitCode := 0

	if err := app.Run(ctx); err != nil {
		log.Println(
			"main: app.Run returned an error",
			"error", err.Error(),
		)

		exitCode = 1
	}

	stop()
	<-ctx.Done()

	os.Exit(exitCode)
}
