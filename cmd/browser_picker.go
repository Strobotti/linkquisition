package main

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/strobotti/linkquisition"
)

type BrowserPicker struct {
	fapp           fyne.App
	browserService linkquisition.BrowserService
	browsers       []linkquisition.Browser
}

func NewBrowserPicker(
	fapp fyne.App,
	browserService linkquisition.BrowserService,
	browsers []linkquisition.Browser,
) *BrowserPicker {
	return &BrowserPicker{
		fapp:           fapp,
		browserService: browserService,
		browsers:       browsers,
	}
}

//nolint:funlen
func (bp *BrowserPicker) Run(_ context.Context, urlToOpen string) error {
	var buttons []fyne.CanvasObject

	for i := range bp.browsers {
		browser := bp.browsers[i]

		buttons = append(
			buttons,
			widget.NewButton(
				browser.Name,
				func() {
					go func() {
						_ = bp.browserService.OpenUrlWithBrowser(urlToOpen, &browser)
					}()
					bp.fapp.Quit()
				},
			),
		)
	}

	w := bp.fapp.NewWindow("Linkquisition")

	w.Canvas().AddShortcut(
		&fyne.ShortcutCopy{}, func(shortcut fyne.Shortcut) {
			fmt.Println("Copying URL to clipboard: " + urlToOpen)
			w.Clipboard().SetContent(urlToOpen)

			// Sleep for a while to allow the Clipboard.SetContent to finish
			time.Sleep(200 * time.Millisecond) //nolint:gomnd
			bp.fapp.Quit()
		},
	)

	w.Canvas().SetOnTypedKey(
		func(keyEvent *fyne.KeyEvent) {
			if keyEvent.Name == fyne.KeyEscape {
				bp.fapp.Quit()
			}
			if len(buttons) > 0 {
				if keyEvent.Name == fyne.KeyReturn {
					buttons[0].(*widget.Button).OnTapped()
					return
				}

				// TODO there must be a better way of doing this
				numkeyNames := []fyne.KeyName{
					fyne.Key1,
					fyne.Key2,
					fyne.Key3,
					fyne.Key4,
					fyne.Key5,
					fyne.Key6,
					fyne.Key7,
					fyne.Key8,
					fyne.Key9,
				}

				for i := range buttons {
					if keyEvent.Name == numkeyNames[i] {
						buttons[i].(*widget.Button).OnTapped()
						return
					}
				}
			}
		},
	)

	var widgets []fyne.CanvasObject

	widgets = append(widgets, buttons...)

	// if the text is too long, it will be truncated
	text := urlToOpen
	if len(urlToOpen) > 75 { //nolint:gomnd
		text = urlToOpen[:75] + "..."
	}

	input := widget.NewEntry()
	input.SetText(text)
	input.Disable()

	widgets = append(
		widgets,
		container.NewBorder(nil, nil, widget.NewLabel("Open:"), nil, input),
		layout.NewSpacer(),
		widget.NewLabel("Press 'ENTER' to pick first, 'ESC' to quit, 'ctrl+c' to copy URL to clipboard"),
	)

	var windowHeight float32 = 50
	for _, wdgt := range widgets {
		windowHeight += wdgt.MinSize().Height
	}

	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(600, windowHeight)) //nolint:gomnd
	w.CenterOnScreen()

	if icon, err := fyne.LoadResourceFromPath("Icon.png"); err == nil {
		w.SetIcon(icon)
	}

	w.SetContent(container.NewVBox(widgets...))

	w.ShowAndRun()

	return nil
}
