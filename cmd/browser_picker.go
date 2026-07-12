package main

import (
	"context"
	"image/color"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/strobotti/linkquisition/internal/safety"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/favicon"
	"github.com/strobotti/linkquisition/internal/i18n"
	"github.com/strobotti/linkquisition/internal/qrcode"
	internalwhois "github.com/strobotti/linkquisition/internal/whois"
	"github.com/strobotti/linkquisition/resources"
)

const (
	horizontalButtonIconSize = 96
	horizontalButtonWidth    = 148
	horizontalButtonHeight   = 148
	verticalWindowWidth      = 600
	verticalWindowMinHeight  = 50
	faviconDisplaySize       = 16
	safetyIndicatorSize      = 12
	safetyCheckTimeout       = 10 * time.Second
)

type BrowserPicker struct {
	fapp            fyne.App
	browserService  linkquisition.BrowserService
	browsers        []linkquisition.Browser
	settingsService linkquisition.SettingsService
	logger          *slog.Logger
	uiHooks         []linkquisition.PluginUIHook
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

//nolint:cyclop
func (picker *BrowserPicker) Run(_ context.Context, urlToOpen string) error {
	var buttons []fyne.CanvasObject
	remember := binding.NewBool()
	_ = remember.Set(false)
	rememberMatchType := binding.NewString()
	// TODO give user the option to choose between site and domain (and later on regex, too)
	_ = rememberMatchType.Set(linkquisition.BrowserMatchTypeSite)

	settings := picker.settingsService.GetSettings()
	pickerLayout := settings.Ui.GetPickerLayout()

	for i := range picker.browsers {
		if pickerLayout == linkquisition.PickerLayoutHorizontal {
			buttons = append(
				buttons,
				picker.makeHorizontalBrowserButton(
					picker.browsers[i], urlToOpen, remember, rememberMatchType,
				),
			)
		} else {
			buttons = append(
				buttons,
				picker.makeBrowserButton(picker.browsers[i], urlToOpen, remember, rememberMatchType),
			)
		}
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

	picker.setupKeyboardShortcuts(w, buttons)

	var widgets []fyne.CanvasObject

	if pickerLayout == linkquisition.PickerLayoutHorizontal {
		widgets = append(widgets, picker.buildHorizontalGrid(buttons, settings.Ui.GetMaxItemsPerRow()))
	} else {
		widgets = append(widgets, buttons...)
	}

	widgets = append(widgets, picker.buildURLDisplay(urlToOpen, w)...)
	widgets = append(widgets, picker.buildRememberCheck(urlToOpen, remember)...)

	if !settings.Ui.HideKeyboardGuideLabel {
		widgets = append(widgets, picker.buildKeyboardGuide()...)
	}

	w.SetIcon(resources.LinkquisitionIcon)

	if pickerLayout == linkquisition.PickerLayoutHorizontal {
		cols := min(len(buttons), settings.Ui.GetMaxItemsPerRow())
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

	w.ShowAndRun()

	return nil
}

func (picker *BrowserPicker) setupKeyboardShortcuts(w fyne.Window, buttons []fyne.CanvasObject) {
	w.Canvas().SetOnTypedKey(
		func(keyEvent *fyne.KeyEvent) {
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

// tapButton triggers the OnTapped handler for a button canvas object.
// Supports both widget.Button (vertical layout) and the tappable container (horizontal layout).
func (picker *BrowserPicker) tapButton(obj fyne.CanvasObject) {
	if btn, ok := obj.(*widget.Button); ok {
		btn.OnTapped()
		return
	}
	// For horizontal layout, the tappable wrapper holds the callback
	if tappable, ok := obj.(*tappableContainer); ok {
		tappable.onTapped()
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

	if settings.Ui.ShowFavicon {
		faviconImg := canvas.NewImageFromResource(theme.ComputerIcon())
		faviconImg.FillMode = canvas.ImageFillContain
		faviconImg.SetMinSize(fyne.NewSize(faviconDisplaySize, faviconDisplaySize))
		urlRowItems = append(urlRowItems, faviconImg)

		// Fetch favicon lazily in a goroutine
		go picker.fetchAndUpdateFavicon(urlToOpen, faviconImg, settings)
	}

	urlRowItems = append(urlRowItems, urlLabel)

	// Safety indicator (only if configured)
	if settings.Security.IsConfigured() {
		indicator := picker.buildSafetyIndicator(urlToOpen, w, settings)
		urlRowItems = append(urlRowItems, layout.NewSpacer(), indicator)
	}

	urlRow := container.NewHBox(urlRowItems...)

	return []fyne.CanvasObject{urlRow}
}

func (picker *BrowserPicker) buildSafetyIndicator(
	urlToOpen string, w fyne.Window, settings *linkquisition.Settings,
) fyne.CanvasObject {
	// Gray circle initially
	grayColor := color.NRGBA{R: 150, G: 150, B: 150, A: 255}
	indicator := canvas.NewCircle(grayColor)
	indicator.StrokeWidth = 0
	indicatorContainer := container.New(
		layout.NewCenterLayout(),
		indicator,
	)
	indicatorContainer.Resize(fyne.NewSize(safetyIndicatorSize, safetyIndicatorSize))

	// Store the result for the tappable report
	var checkResult *safety.CheckResult

	// Run check in background
	go func() {
		checker, err := safety.NewChecker(settings.Security.GetProvider(), settings.Security.APIKey)
		if err != nil {
			picker.logger.Error("Failed to create safety checker", "error", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), safetyCheckTimeout)
		defer cancel()

		result, err := checker.Check(ctx, urlToOpen)
		if err != nil {
			picker.logger.Error("Safety check failed", "url", urlToOpen, "error", err)
			fyne.Do(func() {
				// Yellow for "could not determine"
				indicator.FillColor = color.NRGBA{R: 220, G: 180, B: 50, A: 255}
				indicator.Refresh()
			})
			return
		}

		checkResult = result

		fyne.Do(func() {
			switch result.Level {
			case safety.ThreatLevelSafe:
				indicator.FillColor = color.NRGBA{R: 50, G: 180, B: 50, A: 255}
			case safety.ThreatLevelSuspicious:
				indicator.FillColor = color.NRGBA{R: 220, G: 180, B: 50, A: 255}
			case safety.ThreatLevelDangerous:
				indicator.FillColor = color.NRGBA{R: 220, G: 50, B: 50, A: 255}
			}
			indicator.Refresh()
		})
	}()

	// Wrap in a tappable container for the report popup
	tappable := newTappableContainer(
		indicatorContainer,
		func() {
			if checkResult != nil {
				picker.showSafetyReport(checkResult, w)
			}
		},
	)

	return tappable
}

func (picker *BrowserPicker) showSafetyReport(result *safety.CheckResult, w fyne.Window) {
	var levelText string
	var levelColor color.NRGBA

	switch result.Level {
	case safety.ThreatLevelSafe:
		levelText = i18n.T("picker.safety_level_safe")
		levelColor = color.NRGBA{R: 50, G: 180, B: 50, A: 255}
	case safety.ThreatLevelSuspicious:
		levelText = i18n.T("picker.safety_level_suspicious")
		levelColor = color.NRGBA{R: 220, G: 180, B: 50, A: 255}
	case safety.ThreatLevelDangerous:
		levelText = i18n.T("picker.safety_level_dangerous")
		levelColor = color.NRGBA{R: 220, G: 50, B: 50, A: 255}
	}

	levelLabel := canvas.NewText(levelText, levelColor)
	levelLabel.TextStyle = fyne.TextStyle{Bold: true}
	levelLabel.TextSize = theme.TextSize()

	grid := container.New(layout.NewFormLayout(),
		picker.whoisLabel(i18n.T("picker.safety_provider")), picker.whoisValue(result.Provider),
		picker.whoisLabel(i18n.T("picker.safety_result")), levelLabel,
	)

	if len(result.Details) > 0 {
		detailsText := strings.Join(result.Details, "\n")
		grid.Add(picker.whoisLabel(i18n.T("picker.safety_details")))
		grid.Add(picker.whoisValue(detailsText))
	}

	closeButton := widget.NewButtonWithIcon(
		i18n.T("picker.safety_close"),
		theme.CancelIcon(),
		nil,
	)

	content := container.NewVBox(
		grid,
		container.NewCenter(closeButton),
	)

	popup := widget.NewModalPopUp(content, w.Canvas())
	closeButton.OnTapped = func() { popup.Hide() }
	popup.Show()
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

const (
	whoisWindowWidth          = 450
	whoisTimeout              = 10 * time.Second
	whoisLoadingHeight        = 100
	whoisErrorHeight          = 150
	whoisContentHeightPadding = 20
)

func (picker *BrowserPicker) showWhoisWindow(urlToOpen string) {
	whoisWindow := picker.fapp.NewWindow(i18n.T("picker.whois_title"))

	whoisWindow.Canvas().SetOnTypedKey(func(keyEvent *fyne.KeyEvent) {
		if keyEvent.Name == fyne.KeyEscape {
			whoisWindow.Close()
		}
	})

	// Show loading state
	loading := widget.NewProgressBarInfinite()
	loadingLabel := widget.NewLabel(i18n.T("picker.whois_loading"))
	loadingLabel.Alignment = fyne.TextAlignCenter
	whoisWindow.SetContent(container.NewVBox(
		loadingLabel,
		loading,
	))
	whoisWindow.Resize(fyne.NewSize(whoisWindowWidth, whoisLoadingHeight))
	whoisWindow.SetFixedSize(true)
	whoisWindow.CenterOnScreen()

	// Perform lookup in background
	ctx, cancel := context.WithTimeout(context.Background(), whoisTimeout)

	whoisWindow.SetOnClosed(func() {
		cancel()
	})

	go func() {
		info, err := internalwhois.Lookup(ctx, urlToOpen)

		fyne.Do(func() {
			if err != nil {
				picker.logger.Error("WHOIS lookup failed", "url", urlToOpen, "error", err)
				whoisWindow.SetContent(picker.buildWhoisError(err, whoisWindow))
				whoisWindow.SetFixedSize(false)
				whoisWindow.Resize(fyne.NewSize(whoisWindowWidth, whoisErrorHeight))
				whoisWindow.SetFixedSize(true)
				return
			}
			content := picker.buildWhoisContent(info, whoisWindow)
			whoisWindow.SetContent(content)
			whoisWindow.SetFixedSize(false)
			whoisWindow.Resize(fyne.NewSize(whoisWindowWidth, content.MinSize().Height+whoisContentHeightPadding))
			whoisWindow.SetFixedSize(true)
		})
	}()

	whoisWindow.Show()
}

func (picker *BrowserPicker) buildWhoisContent(
	info *internalwhois.DomainInfo, w fyne.Window,
) fyne.CanvasObject {
	dnssecText := "✗"
	dnssecColor := color.NRGBA{R: 220, G: 50, B: 50, A: 255}
	if info.DNSSec {
		dnssecText = "✓"
		dnssecColor = color.NRGBA{R: 50, G: 180, B: 50, A: 255}
	}

	dnssecLabel := canvas.NewText(dnssecText, dnssecColor)
	dnssecLabel.TextStyle = fyne.TextStyle{Bold: true}
	dnssecLabel.TextSize = theme.TextSize()

	grid := container.New(layout.NewFormLayout(),
		picker.whoisLabel(i18n.T("picker.whois_domain")), picker.whoisValue(info.Domain),
		picker.whoisLabel(i18n.T("picker.whois_registrar")), picker.whoisValue(info.Registrar),
		picker.whoisLabel(i18n.T("picker.whois_created")), picker.whoisValue(info.CreatedDate),
		picker.whoisLabel(i18n.T("picker.whois_expires")), picker.whoisValue(info.ExpiryDate),
		picker.whoisLabel(i18n.T("picker.whois_updated")), picker.whoisValue(info.UpdatedDate),
		picker.whoisLabel(i18n.T("picker.whois_age")), picker.whoisValue(info.DomainAge),
		picker.whoisLabel(i18n.T("picker.whois_dnssec")), dnssecLabel,
	)

	if len(info.NameServers) > 0 {
		nsText := strings.Join(info.NameServers, ", ")
		grid.Add(picker.whoisLabel(i18n.T("picker.whois_nameservers")))
		grid.Add(picker.whoisValue(nsText))
	}

	if len(info.Status) > 0 {
		statusText := strings.Join(info.Status, ", ")
		grid.Add(picker.whoisLabel(i18n.T("picker.whois_status")))
		grid.Add(picker.whoisValue(statusText))
	}

	closeButton := widget.NewButtonWithIcon(
		i18n.T("picker.whois_close"),
		theme.CancelIcon(),
		func() { w.Close() },
	)

	return container.NewVBox(
		grid,
		container.NewCenter(closeButton),
	)
}

func (picker *BrowserPicker) buildWhoisError(err error, w fyne.Window) fyne.CanvasObject {
	errLabel := widget.NewLabel(i18n.T("picker.whois_error", map[string]interface{}{
		"Error": err.Error(),
	}))
	errLabel.Wrapping = fyne.TextWrapWord

	closeButton := widget.NewButtonWithIcon(
		i18n.T("picker.whois_close"),
		theme.CancelIcon(),
		func() { w.Close() },
	)

	return container.NewVBox(
		container.NewCenter(errLabel),
		container.NewCenter(closeButton),
	)
}

func (picker *BrowserPicker) whoisValue(value string) *widget.Label {
	if value == "" {
		value = "—"
	}

	v := widget.NewLabel(value)
	v.Wrapping = fyne.TextWrapWord

	return v
}

func (picker *BrowserPicker) whoisLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.TextStyle = fyne.TextStyle{Bold: true}

	return l
}

// fetchAndUpdateFavicon fetches the favicon in the background and updates the image widget.
// This runs in a goroutine to avoid blocking the UI startup.
func (picker *BrowserPicker) fetchAndUpdateFavicon(
	urlToOpen string, img *canvas.Image, settings *linkquisition.Settings,
) {
	strategy := settings.Ui.GetFaviconStrategy()
	cacheDir := filepath.Join(picker.settingsService.GetConfigFolderPath(), "favicons")

	fetcher := favicon.NewFetcher(strategy, cacheDir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint:mnd
	defer cancel()

	data, err := fetcher.Fetch(ctx, urlToOpen)
	if err != nil {
		picker.logger.Debug("Failed to fetch favicon", "url", urlToOpen, "error", err)
		return
	}

	res := fyne.NewStaticResource("favicon.png", data)
	img.Resource = res
	img.Refresh()
}

func (picker *BrowserPicker) buildRememberCheck(urlToOpen string, remember binding.Bool) []fyne.CanvasObject {
	uto := linkquisition.NewURL(urlToOpen)
	site, _ := uto.GetSite()
	check := widget.NewCheckWithData(
		i18n.T("picker.remember_choice", map[string]interface{}{"Site": site}),
		remember,
	)

	return []fyne.CanvasObject{check}
}

func (picker *BrowserPicker) buildKeyboardGuide() []fyne.CanvasObject {
	copyShortcut := "Ctrl+C"
	if runtime.GOOS == "darwin" {
		copyShortcut = "⌘+C"
	}

	return []fyne.CanvasObject{
		layout.NewSpacer(),
		widget.NewLabel(i18n.T("picker.keyboard_guide", map[string]interface{}{
			"CopyShortcut": copyShortcut,
		})),
	}
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
		picker.browserOpenCallback(browser, urlToOpen, remember, rememberMatchType),
	)
}

func (picker *BrowserPicker) makeHorizontalBrowserButton(
	browser linkquisition.Browser,
	urlToOpen string,
	remember binding.Bool,
	rememberMatchType binding.String,
) fyne.CanvasObject {
	callback := picker.browserOpenCallback(browser, urlToOpen, remember, rememberMatchType)

	var iconWidget fyne.CanvasObject
	iconBytes, err := picker.browserService.GetIconForBrowser(browser)
	if err != nil {
		picker.logger.Debug("Failed to load browser icon", "browser", browser.Name, "error", err)
		// Show a placeholder with the first letter when no icon is available
		bg := canvas.NewRectangle(color.NRGBA{R: 200, G: 200, B: 200, A: 40})
		bg.SetMinSize(fyne.NewSize(horizontalButtonIconSize, horizontalButtonIconSize))
		firstRune, _ := utf8.DecodeRuneInString(browser.Name)
		placeholder := canvas.NewText(string(firstRune), color.NRGBA{R: 80, G: 80, B: 80, A: 255})
		placeholder.TextSize = 36
		placeholder.TextStyle = fyne.TextStyle{Bold: true}
		placeholder.Alignment = fyne.TextAlignCenter
		iconWidget = container.NewStack(bg, container.NewCenter(placeholder))
	} else {
		res := fyne.NewStaticResource("icon.png", iconBytes)
		img := canvas.NewImageFromResource(res)
		img.FillMode = canvas.ImageFillContain
		img.SetMinSize(fyne.NewSize(horizontalButtonIconSize, horizontalButtonIconSize))
		iconWidget = img
	}

	nameLabel := widget.NewLabel(browser.Name)
	nameLabel.Alignment = fyne.TextAlignCenter
	nameLabel.Truncation = fyne.TextTruncateEllipsis

	content := container.NewVBox(
		container.NewCenter(iconWidget),
		nameLabel,
	)

	return newTappableContainer(content, callback)
}

func (picker *BrowserPicker) browserOpenCallback(
	browser linkquisition.Browser,
	urlToOpen string,
	remember binding.Bool,
	rememberMatchType binding.String,
) func() {
	return func() {
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
	}
}

// tappableContainer wraps a canvas object to make it respond to tap and hover events.
type tappableContainer struct {
	widget.BaseWidget
	content  fyne.CanvasObject
	bg       *canvas.Rectangle
	onTapped func()
}

// Compile-time interface checks.
var (
	_ fyne.Tappable     = (*tappableContainer)(nil)
	_ desktop.Hoverable = (*tappableContainer)(nil)
)

func newTappableContainer(content fyne.CanvasObject, onTapped func()) *tappableContainer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = 8
	t := &tappableContainer{
		content:  content,
		bg:       bg,
		onTapped: onTapped,
	}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableContainer) Tapped(_ *fyne.PointEvent) {
	if t.onTapped != nil {
		t.onTapped()
	}
}

func (t *tappableContainer) MouseIn(_ *desktop.MouseEvent) {
	t.bg.FillColor = color.NRGBA{R: 150, G: 150, B: 150, A: 30}
	t.bg.Refresh()
}

func (t *tappableContainer) MouseMoved(_ *desktop.MouseEvent) {}

func (t *tappableContainer) MouseOut() {
	t.bg.FillColor = color.Transparent
	t.bg.Refresh()
}

func (t *tappableContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(t.bg, t.content))
}
