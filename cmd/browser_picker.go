package main

import (
	"context"
	"image/color"
	"log/slog"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
	"github.com/strobotti/linkquisition/resources"
)

const (
	horizontalButtonIconSize = 96
	horizontalButtonWidth    = 148
	horizontalButtonHeight   = 148
	verticalWindowWidth      = 600
	verticalWindowMinHeight  = 50
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

//nolint:funlen,cyclop
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

	widgets = append(widgets, picker.buildURLDisplay(urlToOpen)...)
	widgets = append(widgets, picker.buildRememberCheck(urlToOpen, remember)...)

	if !settings.Ui.HideKeyboardGuideLabel {
		widgets = append(widgets, picker.buildKeyboardGuide()...)
	}

	w.SetFixedSize(true)
	w.SetIcon(resources.LinkquisitionIcon)
	w.SetContent(container.NewVBox(widgets...))

	if pickerLayout == linkquisition.PickerLayoutHorizontal {
		cols := min(len(buttons), settings.Ui.GetMaxItemsPerRow())
		rows := (len(buttons) + cols - 1) / cols
		width := float32(cols*horizontalButtonWidth) + 32    //nolint:mnd
		height := float32(rows*horizontalButtonHeight) + 140 //nolint:mnd
		w.Resize(fyne.NewSize(width, height))
	} else {
		w.Resize(fyne.NewSize(verticalWindowWidth, verticalWindowMinHeight))
	}
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
	_ = maxPerRow // column count is controlled by window width + cell size
	cellSize := fyne.NewSize(horizontalButtonWidth, horizontalButtonHeight)
	grid := container.New(layout.NewGridWrapLayout(cellSize), buttons...)
	return container.NewPadded(grid)
}

func (picker *BrowserPicker) buildURLDisplay(urlToOpen string) []fyne.CanvasObject {
	text := urlToOpen
	if len(urlToOpen) > 75 { //nolint:mnd
		text = urlToOpen[:75] + "..."
	}

	input := widget.NewEntry()
	input.SetText(text)
	input.Disable()

	return []fyne.CanvasObject{
		container.NewBorder(nil, nil, widget.NewLabel(i18n.T("picker.open_label")), nil, input),
	}
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
		bg := canvas.NewRectangle(color.NRGBA{R: 200, G: 200, B: 200, A: 40}) //nolint:mnd
		bg.SetMinSize(fyne.NewSize(horizontalButtonIconSize, horizontalButtonIconSize))
		placeholder := canvas.NewText(string([]rune(browser.Name)[0]), color.NRGBA{R: 80, G: 80, B: 80, A: 255}) //nolint:mnd,lll
		placeholder.TextSize = 36                                                                                //nolint:mnd
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
	bg.CornerRadius = 8 //nolint:mnd
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
	t.bg.FillColor = color.NRGBA{R: 150, G: 150, B: 150, A: 30} //nolint:mnd
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
