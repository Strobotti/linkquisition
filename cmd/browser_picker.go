package main

import (
	"context"
	"image/color"
	"log/slog"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/browserprofiles"
	"github.com/strobotti/linkquisition/internal/favicon"
	"github.com/strobotti/linkquisition/internal/i18n"
	"github.com/strobotti/linkquisition/internal/qrcode"
	"github.com/strobotti/linkquisition/internal/safety"
	"github.com/strobotti/linkquisition/internal/ui"
	"github.com/strobotti/linkquisition/resources"
)

const (
	horizontalButtonIconSize = 96
	horizontalButtonWidth    = 148
	horizontalButtonHeight   = 148
	verticalWindowWidth      = 600
	verticalWindowMinHeight  = 50
	faviconDisplaySize       = 16
	rememberIndentMultiplier = 4
)

type BrowserPicker struct {
	fapp             fyne.App
	browserService   linkquisition.BrowserService
	browsers         []linkquisition.Browser
	settingsService  linkquisition.SettingsService
	logger           *slog.Logger
	uiHooks          []linkquisition.PluginUIHook
	modalOpen        bool
	shiftLeftHeld    bool
	shiftRightHeld   bool
	privateToggle    *widget.Check
	privateToggleSet bool
	privateSupported bool
	pickerCanvas     fyne.Canvas
	pickerWindow     fyne.Window
}

func NewBrowserPicker(
	fapp fyne.App,
	browserService linkquisition.BrowserService,
	browsers []linkquisition.Browser,
	settingsService linkquisition.SettingsService,
	logger *slog.Logger,
	uiHooks []linkquisition.PluginUIHook,
) *BrowserPicker {
	return &BrowserPicker{
		fapp:            fapp,
		browserService:  browserService,
		browsers:        browsers,
		settingsService: settingsService,
		logger:          logger,
		uiHooks:         uiHooks,
	}
}

// urlOpener returns a URLOpener function that opens URLs using the browser service,
// avoiding circular loops when Linkquisition is the default browser.
func (picker *BrowserPicker) urlOpener() ui.URLOpener {
	return func(rawURL string) error {
		picker.logger.Debug("Opening external URL from picker", "url", rawURL)

		return openExternalURLWithService(rawURL, picker.browserService)
	}
}

//nolint:cyclop
func (picker *BrowserPicker) Run(_ context.Context, urlToOpen string) error {
	var buttons []fyne.CanvasObject
	remember := binding.NewBool()
	_ = remember.Set(false)
	rememberMatchType := binding.NewString()
	_ = rememberMatchType.Set(linkquisition.BrowserMatchTypeSite)

	settings := picker.settingsService.GetSettings()
	pickerLayout := settings.Ui.GetPickerLayout()

	// Prune expired security cache entries in the background
	if settings.Security.Cache.Enabled && settings.Security.IsConfigured() {
		go safety.NewCache(
			picker.settingsService.GetConfigFolderPath(),
			settings.Security.GetProvider(),
			settings.Security.Cache.GetTTL(),
		).PruneExpired()
	}

	for i := range picker.browsers {
		if pickerLayout == linkquisition.PickerLayoutHorizontal {
			buttons = append(
				buttons,
				picker.makeHorizontalBrowserButton(
					&picker.browsers[i], urlToOpen, remember, rememberMatchType,
				),
			)
		} else {
			buttons = append(
				buttons,
				picker.makeBrowserButton(&picker.browsers[i], urlToOpen, remember, rememberMatchType),
			)
		}
	}

	w := picker.fapp.NewWindow(i18n.T("picker.window_title"))
	picker.pickerWindow = w

	w.Canvas().AddShortcut(
		&fyne.ShortcutCopy{}, func(shortcut fyne.Shortcut) {
			picker.logger.Debug("Copying URL to clipboard", "url", urlToOpen)
			w.Clipboard().SetContent(urlToOpen)

			// Sleep for a while to allow the Clipboard.SetContent to finish
			time.Sleep(200 * time.Millisecond) //nolint:mnd
			picker.fapp.Quit()
		},
	)

	picker.pickerCanvas = w.Canvas()
	picker.setupKeyboardShortcuts(w, buttons)
	picker.setupPrivateModeModifier(w)

	var widgets []fyne.CanvasObject

	if pickerLayout == linkquisition.PickerLayoutHorizontal {
		widgets = append(widgets, picker.buildHorizontalGrid(buttons, settings.Ui.GetMaxItemsPerRow()))
	} else {
		widgets = append(widgets, buttons...)
	}

	widgets = append(widgets, picker.buildURLDisplay(urlToOpen, w)...)
	widgets = append(widgets, picker.buildPrivateModeControls(w)...)
	widgets = append(widgets, picker.buildRememberCheck(urlToOpen, remember, rememberMatchType)...)

	if !settings.Ui.HideKeyboardGuideLabel {
		widgets = append(widgets, picker.buildKeyboardGuide()...)
	}

	w.SetIcon(resources.LinkquisitionIcon)

	if pickerLayout == linkquisition.PickerLayoutHorizontal {
		cols := min(len(buttons), settings.Ui.GetMaxItemsPerRow())
		if cols == 0 {
			cols = 1
		}
		rows := (len(buttons) + cols - 1) / cols
		width := float32(cols+1) * horizontalButtonWidth
		height := float32(rows*horizontalButtonHeight) + 180
		w.Resize(fyne.NewSize(width, height))
	} else {
		w.Resize(fyne.NewSize(verticalWindowWidth, verticalWindowMinHeight))
	}

	w.SetFixedSize(true)

	mainContent := container.NewVBox(widgets...)
	lightMode := picker.settingsService.GetSettings().Ui.GetTheme() == linkquisition.ThemeLight
	w.SetContent(buildPickerContent(mainContent, w, picker.uiHooks, lightMode))

	w.CenterOnScreen()

	setPickerWindowAlwaysOnTop()

	w.ShowAndRun()

	return nil
}

func supportedPrivateArgument(browsers []linkquisition.Browser) (string, bool) {
	for _, browser := range browsers {
		if argument, ok := browserprofiles.PrivateArgument(browser.Name); ok {
			return argument, true
		}
	}
	return "", false
}

func (picker *BrowserPicker) handleShiftKey(event *fyne.KeyEvent, pressed bool) {
	switch {
	case event.Name == desktop.KeyShiftLeft:
		picker.shiftLeftHeld = pressed
	case event.Name == desktop.KeyShiftRight:
		picker.shiftRightHeld = pressed
	default:
		return
	}
	picker.refreshPrivateModeToggle()
}

// buildPrivateModeControls creates the "open in private window" toggle, plus a note when no
// available browser supports private launching.
func (picker *BrowserPicker) buildPrivateModeControls(w fyne.Window) []fyne.CanvasObject {
	_, picker.privateSupported = supportedPrivateArgument(picker.browsers)
	picker.privateToggle = widget.NewCheck(i18n.T("picker.private_mode"), func(enabled bool) {
		if !picker.shiftLeftHeld && !picker.shiftRightHeld {
			picker.privateToggleSet = enabled
		}
		picker.refreshPrivateModeToggle()
		w.Canvas().Unfocus()
	})

	var controls []fyne.CanvasObject
	if !picker.privateSupported {
		picker.privateToggle.Disable()
		guide := widget.NewLabel(i18n.T("picker.private_unsupported"))
		guide.Wrapping = fyne.TextWrapWord
		guide.TextStyle = fyne.TextStyle{Italic: true}
		controls = append(controls, guide)
	}
	return append(controls, picker.privateToggle)
}

func (picker *BrowserPicker) setupKeyboardShortcuts(w fyne.Window, buttons []fyne.CanvasObject) {
	w.Canvas().SetOnTypedKey(
		func(keyEvent *fyne.KeyEvent) {
			if picker.modalOpen {
				return
			}
			if keyEvent.Name == fyne.KeyEscape {
				picker.fapp.Quit()
			}
			if len(buttons) > 0 {
				if keyEvent.Name == fyne.KeyReturn {
					picker.tapButton(buttons[0])
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
						picker.tapButton(buttons[i])
						return
					}
				}
			}
		},
	)
}

// setupPrivateModeModifier tracks held Shift keys so the private toggle reflects them while held.
func (picker *BrowserPicker) setupPrivateModeModifier(w fyne.Window) {
	desktopCanvas, ok := w.Canvas().(desktop.Canvas)
	if !ok {
		return
	}

	desktopCanvas.SetOnKeyDown(func(event *fyne.KeyEvent) {
		picker.handleShiftKey(event, true)
	})
	desktopCanvas.SetOnKeyUp(func(event *fyne.KeyEvent) {
		picker.handleShiftKey(event, false)
	})
}

func (picker *BrowserPicker) refreshPrivateModeToggle() {
	if picker.privateToggle == nil {
		return
	}
	picker.privateToggle.Checked = picker.privateModeEnabled()
	picker.privateToggle.Refresh()
}

func (picker *BrowserPicker) privateModeEnabled() bool {
	return picker.privateSupported && (picker.privateToggleSet || picker.shiftLeftHeld || picker.shiftRightHeld)
}

func (picker *BrowserPicker) tapButton(obj fyne.CanvasObject) {
	if btn, ok := obj.(*widget.Button); ok {
		btn.OnTapped()
		return
	}
	// For horizontal layout, the tappable wrapper holds the callback
	if tappable, ok := obj.(*ui.TappableContainer); ok {
		tappable.OnTapped()
	}
}

func (picker *BrowserPicker) buildHorizontalGrid(buttons []fyne.CanvasObject, maxPerRow int) fyne.CanvasObject {
	cols := min(len(buttons), maxPerRow)
	return container.NewGridWithColumns(cols, buttons...)
}

func (picker *BrowserPicker) buildURLDisplay(urlToOpen string, w fyne.Window) []fyne.CanvasObject {
	text := urlToOpen
	if len(urlToOpen) > 75 { //nolint:mnd
		text = urlToOpen[:75] + "..."
	}

	urlLabel := widget.NewLabel(text)
	urlLabel.TextStyle = fyne.TextStyle{Monospace: true}

	menuButton := picker.buildMenuButton(urlToOpen, w)

	settings := picker.settingsService.GetSettings()

	var urlRowItems []fyne.CanvasObject
	urlRowItems = append(urlRowItems, menuButton)

	// Safety indicator (only if configured) — between menu and favicon
	if settings.Security.IsConfigured() {
		indicator := picker.buildSafetyIndicator(urlToOpen, w, settings)
		urlRowItems = append(urlRowItems, indicator)
	}

	if settings.Ui.ShowFavicon {
		faviconImg := canvas.NewImageFromResource(theme.ComputerIcon())
		faviconImg.FillMode = canvas.ImageFillContain
		faviconImg.SetMinSize(fyne.NewSize(faviconDisplaySize, faviconDisplaySize))
		urlRowItems = append(urlRowItems, faviconImg)

		// Fetch favicon lazily in a goroutine
		go picker.fetchAndUpdateFavicon(urlToOpen, faviconImg, settings)
	}

	urlRowItems = append(urlRowItems, urlLabel)

	urlRow := container.NewHBox(urlRowItems...)

	return []fyne.CanvasObject{urlRow}
}

func (picker *BrowserPicker) buildMenuButton(urlToOpen string, w fyne.Window) fyne.CanvasObject {
	menuButton := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil)
	menuButton.Importance = widget.LowImportance
	menuButton.OnTapped = func() {
		menu := fyne.NewMenu("",
			fyne.NewMenuItem(i18n.T("picker.menu_copy"), func() {
				w.Clipboard().SetContent(urlToOpen)
			}),
			fyne.NewMenuItem(i18n.T("picker.menu_qr"), func() {
				picker.showQRCodePopup(urlToOpen, w)
			}),
			fyne.NewMenuItem(i18n.T("picker.menu_whois"), func() {
				picker.showWhoisWindow(urlToOpen)
			}),
		)

		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(menuButton)
		pos.Y += menuButton.Size().Height
		widget.ShowPopUpMenuAtPosition(menu, w.Canvas(), pos)
	}

	return menuButton
}

func (picker *BrowserPicker) showQRCodePopup(urlToOpen string, w fyne.Window) {
	const qrSize = 256

	png, err := qrcode.Generate(urlToOpen, qrSize)
	if err != nil {
		picker.logger.Error("Failed to generate QR code", "url", urlToOpen, "error", err)
		return
	}

	qrResource := fyne.NewStaticResource("qrcode.png", png)
	qrImage := canvas.NewImageFromResource(qrResource)
	qrImage.FillMode = canvas.ImageFillContain
	qrImage.SetMinSize(fyne.NewSize(qrSize, qrSize))

	var popup *widget.PopUp

	closeButton := widget.NewButtonWithIcon(
		i18n.T("picker.qr_close"),
		theme.CancelIcon(),
		func() {
			popup.Hide()
		},
	)

	content := container.NewVBox(
		container.NewCenter(qrImage),
		container.NewCenter(closeButton),
	)

	popup = widget.NewModalPopUp(content, w.Canvas())
	popup.Show()
}

// fetchAndUpdateFavicon fetches the favicon in the background and updates the image widget.
// This runs in a goroutine to avoid blocking the UI startup.
func (picker *BrowserPicker) fetchAndUpdateFavicon(
	urlToOpen string, img *canvas.Image, settings *linkquisition.Settings,
) {
	strategy := settings.Ui.GetFaviconStrategy()
	cacheDir := filepath.Join(picker.settingsService.GetConfigFolderPath(), "favicons")

	fetcher := favicon.NewFetcher(strategy, cacheDir)

	ctx, cancel := context.WithTimeout(context.Background(), faviconFetchTimeout)
	defer cancel()

	data, err := fetcher.Fetch(ctx, urlToOpen)
	if err != nil {
		picker.logger.Debug("Failed to fetch favicon", "url", urlToOpen, "error", err)
		return
	}

	res := fyne.NewStaticResource("favicon", data)
	fyne.Do(func() {
		img.Resource = res
		img.Refresh()
	})
}

func (picker *BrowserPicker) buildRememberCheck(
	urlToOpen string,
	remember binding.Bool,
	rememberMatchType binding.String,
) []fyne.CanvasObject {
	uto := linkquisition.NewURL(urlToOpen)
	site, siteErr := uto.GetSite()
	domain, domainErr := uto.GetDomain()
	if siteErr != nil || domainErr != nil || site == "" || domain == "" {
		picker.logger.Warn(
			"Unable to determine URL match value for remember choice",
			"url", urlToOpen,
			"site_error", siteErr,
			"domain_error", domainErr,
		)
		check := widget.NewCheckWithData(i18n.T("picker.remember_choice"), remember)
		check.Disable()
		return []fyne.CanvasObject{check}
	}

	// If site == domain (no subdomain), just show the simple checkbox
	if site == domain {
		check := widget.NewCheckWithData(
			i18n.T("picker.remember_choice"),
			remember,
		)
		return []fyne.CanvasObject{check}
	}

	// Build radio options with actual values shown
	siteLabel := i18n.T("picker.remember_site", map[string]interface{}{"Site": site})
	domainLabel := i18n.T("picker.remember_domain", map[string]interface{}{"Domain": domain})

	radio := widget.NewRadioGroup(
		[]string{siteLabel, domainLabel},
		func(selected string) {
			if selected == domainLabel {
				_ = rememberMatchType.Set(linkquisition.BrowserMatchTypeDomain)
			} else {
				_ = rememberMatchType.Set(linkquisition.BrowserMatchTypeSite)
			}
		},
	)
	radio.SetSelected(siteLabel)

	// Indent the radio group to visually nest it under the checkbox
	indent := canvas.NewRectangle(color.Transparent)
	indent.SetMinSize(fyne.NewSize(theme.Padding()*rememberIndentMultiplier, 0))
	indentedRadio := container.NewHBox(indent, radio)
	indentedRadio.Hide()

	check := widget.NewCheck(i18n.T("picker.remember_choice"), func(checked bool) {
		_ = remember.Set(checked)
		if checked {
			indentedRadio.Show()
		} else {
			indentedRadio.Hide()
		}
	})

	return []fyne.CanvasObject{check, indentedRadio}
}

func (picker *BrowserPicker) buildKeyboardGuide() []fyne.CanvasObject {
	copyShortcut := "Ctrl+C"
	if runtime.GOOS == osDarwin {
		copyShortcut = "⌘+C"
	}

	return []fyne.CanvasObject{
		layout.NewSpacer(),
		widget.NewLabel(i18n.T("picker.keyboard_guide", map[string]interface{}{
			"CopyShortcut": copyShortcut,
		})),
	}
}
