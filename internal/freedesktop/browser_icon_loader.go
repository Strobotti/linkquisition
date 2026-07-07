//go:build linux

package freedesktop

import (
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"

	"github.com/strobotti/linkquisition"
)

type BrowserIconLoader interface {
	LoadIcon(browser linkquisition.Browser) ([]byte, error)
}

type DefaultBrowserIconLoader struct {
	XdgService          *XdgService
	DesktopEntryService *DesktopEntryService
}

func (l *DefaultBrowserIconLoader) LoadIcon(browser linkquisition.Browser) ([]byte, error) {
	dePath, err := l.XdgService.GetDesktopEntryPathForBinary(browser.Command)
	if err != nil {
		return nil, err
	}

	desktopEntry, err := l.DesktopEntryService.CreateFromPath(dePath)
	if err != nil {
		return nil, err
	}

	// If the icon field is already a full path, use it directly
	if strings.HasPrefix(desktopEntry.Icon, "/") {
		if icon, errLoad := fyne.LoadResourceFromPath(desktopEntry.Icon); errLoad == nil {
			return icon.Content(), nil
		}
	}

	// Search for the icon in /usr/share/icons
	iconName := desktopEntry.Icon
	var foundPath string

	_ = filepath.WalkDir("/usr/share/icons", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}

		baseName := d.Name()
		ext := filepath.Ext(baseName)
		nameWithoutExt := strings.TrimSuffix(baseName, ext)

		if nameWithoutExt == iconName {
			foundPath = path
			return filepath.SkipAll
		}
		return nil
	})

	if foundPath != "" {
		if icon, errLoad := fyne.LoadResourceFromPath(foundPath); errLoad == nil {
			return icon.Content(), nil
		}
	}

	// As a fallback we'll use the application icon
	icon, err := fyne.LoadResourceFromPath("Icon.png")
	if err != nil {
		return nil, err
	}

	return icon.Content(), nil
}
