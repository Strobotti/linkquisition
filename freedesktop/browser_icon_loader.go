package freedesktop

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
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

	// TODO how to abstract this away? Also should probably not use find but go-code to find the icon
	// TODO we should probably check the Icon field for a full path to an icon file
	findArgs := []string{
		"/usr/share/icons",
		"-type",
		"f,l",
		"-name",
		desktopEntry.Icon + ".*",
	}

	cmd := exec.Command("find", findArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out

	if errRun := cmd.Run(); errRun != nil {
		return nil, fmt.Errorf("failed to fetch icons with name `%s`: %v", desktopEntry.Icon, errRun)
	}

	scanner := bufio.NewScanner(strings.NewReader(out.String()))

	for scanner.Scan() {
		// TODO we're using the first icon we find, but we should probably pick one with optimal resolution
		//      matching the possible theme and color scheme. Now we probably have a following response:
		//      	/usr/share/icons/hicolor/128x128/apps/firefox.png
		// 			/usr/share/icons/hicolor/48x48/apps/firefox.png
		// 			/usr/share/icons/HighContrast/24x24/apps/firefox.png
		// 			/usr/share/icons/HighContrast/22x22/apps/firefox.png
		//      ...and by sheer "luck" we get the 128x128 icon here which is probably the best one, but still
		//      not guaranteed to be the best one - especially if the user wants to use a high-contrast theme.
		if icon, errLoad := fyne.LoadResourceFromPath(scanner.Text()); errLoad == nil {
			return icon.Content(), nil
		}
	}

	// As a fallback we'll use the application icon - not very elegant approach, is it?
	icon, err := fyne.LoadResourceFromPath("Icon.png")
	if err != nil {
		return nil, err
	}

	return icon.Content(), nil
}
