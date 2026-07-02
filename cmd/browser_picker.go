package main

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
)

type BrowserPicker struct {
	fapp            fyne.App
	browserService  linkquisition.BrowserService
	browsers        []linkquisition.Browser
	settingsService linkquisition.SettingsService
	logger          *slog.Logger
}

func NewBrowserPicker(
	fapp fyne.App,
	browserService linkquisition.BrowserService,
	browsers []linkquisition.Browser,
	settingsService linkquisition.SettingsService,
	logger *slog.Logger,
) *BrowserPicker {
	return &BrowserPicker{
		fapp:            fapp,
		browserService:  browserService,
		browsers:        browsers,
		settingsService: settingsService,
		logger:          logger,
	}
}

//nolint:funlen
func (picker *BrowserPicker) Run(_ context.Context, urlToOpen string) error {
	var buttons []fyne.CanvasObject
	remember := binding.NewBool()
	_ = remember.Set(false)
	rememberMatchType := binding.NewString()
	// TODO give user the option to choose between site and domain (and later on regex, too)
	_ = rememberMatchType.Set(linkquisition.BrowserMatchTypeSite)

	for i := range picker.browsers {
		buttons = append(
			buttons,
			picker.makeBrowserButton(picker.browsers[i], urlToOpen, remember, rememberMatchType),
		)
	}

	w := picker.fapp.NewWindow(i18n.T("picker.window_title"))

	w.Canvas().AddShortcut(
		&fyne.ShortcutCopy{}, func(shortcut fyne.Shortcut) {
			picker.logger.Debug("Copying URL to clipboard", "url", urlToOpen)
			w.Clipboard().SetContent(urlToOpen)

			// Sleep for a while to allow the Clipboard.SetContent to finish
			time.Sleep(200 * time.Millisecond) //nolint:mnd
			picker.fapp.Quit()
		},
	)

	w.Canvas().SetOnTypedKey(
		func(keyEvent *fyne.KeyEvent) {
			if keyEvent.Name == fyne.KeyEscape {
				picker.fapp.Quit()
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
	if len(urlToOpen) > 75 { //nolint:mnd
		text = urlToOpen[:75] + "..."
	}

	input := widget.NewEntry()
	input.SetText(text)
	input.Disable()

	widgets = append(
		widgets,
		container.NewBorder(nil, nil, widget.NewLabel(i18n.T("picker.open_label")), nil, input),
	)

	uto := linkquisition.NewURL(urlToOpen)
	site, _ := uto.GetSite()
	check := widget.NewCheckWithData(
		i18n.T("picker.remember_choice", map[string]interface{}{"Site": site}),
		remember,
	)

	widgets = append(
		widgets,
		check,
	)

	if !picker.settingsService.GetSettings().Ui.HideKeyboardGuideLabel {
		copyShortcut := "Ctrl+C"
		if runtime.GOOS == "darwin" {
			copyShortcut = "⌘+C"
		}

		widgets = append(
			widgets,
			layout.NewSpacer(),
			widget.NewLabel(i18n.T("picker.keyboard_guide", map[string]interface{}{
				"CopyShortcut": copyShortcut,
			})),
		)
	}

	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(600, 50)) //nolint:mnd
	w.CenterOnScreen()

	if icon, err := fyne.LoadResourceFromPath("Icon.png"); err == nil {
		w.SetIcon(icon)
	}

	w.SetContent(container.NewVBox(widgets...))

	w.ShowAndRun()

	return nil
}

func (picker *BrowserPicker) makeBrowserButton(
	browser linkquisition.Browser,
	urlToOpen string,
	remember binding.Bool,
	rememberMatchType binding.String,
) fyne.CanvasObject {
	var icon fyne.Resource

	iconBytes, err := picker.browserService.GetIconForBrowser(browser)
	if err != nil {
		picker.logger.Debug("Failed to load browser icon", "browser", browser.Name, "error", err)
	} else {
		icon = fyne.NewStaticResource("icon.png", iconBytes)
	}

	return widget.NewButtonWithIcon(
		browser.Name,
		icon,
		func() {
			rem, _ := remember.Get()
			picker.logger.Debug("Opening URL with browser", "browser", browser.Name, "remember", rem)

			settings := picker.settingsService.GetSettings()
			remType, _ := rememberMatchType.Get()

			if rem {
				uto := linkquisition.NewURL(urlToOpen)
				matchValue, _ := uto.GetDomain()

				if remType == linkquisition.BrowserMatchTypeSite {
					matchValue, _ = uto.GetSite()
				}

				settings.AddRuleToBrowser(&browser, remType, matchValue)
				if writeErr := picker.settingsService.WriteSettings(settings); writeErr != nil {
					picker.logger.Error("Failed to write settings", "error", writeErr)
				}
			}

			go func() {
				_ = picker.browserService.OpenUrlWithBrowser(urlToOpen, &browser)
			}()
			picker.fapp.Quit()
		},
	)
}
