//go:build linux

package freedesktop

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/alessio/shellescape.v1"

	"github.com/strobotti/linkquisition"
)

var _ linkquisition.BrowserService = (*BrowserService)(nil)

type BrowserService struct {
	XdgService          *XdgService
	DesktopEntryService *DesktopEntryService
	BrowserIconLoader   BrowserIconLoader
}

func (b *BrowserService) GetAvailableBrowsers() ([]linkquisition.Browser, error) {
	paths := b.XdgService.GetApplicationPaths()

	if len(paths) == 0 {
		return nil, fmt.Errorf("no valid desktop entry paths found in $XDG_DATA_DIRS")
	}

	var browsers []linkquisition.Browser

	for _, dir := range paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".desktop") {
				continue
			}

			path := filepath.Join(dir, entry.Name())

			if !desktopEntryHasCategory(path, "WebBrowser") {
				continue
			}

			// skip Linkquisition as an available browser
			if strings.Contains(entry.Name(), "linkquisition") {
				continue
			}

			desktopEntry, err := b.DesktopEntryService.CreateFromPath(path)
			if err != nil {
				return nil, fmt.Errorf("failed to parse desktop entry %q: %v", path, err)
			}

			browser := linkquisition.Browser{
				Name:     desktopEntry.Name,
				Command:  desktopEntry.Exec,
				IconPath: b.BrowserIconLoader.ResolveIconName(desktopEntry.Icon),
			}
			browsers = append(browsers, browser)
		}
	}

	return browsers, nil
}

func (b *BrowserService) GetDefaultBrowser() (linkquisition.Browser, error) {
	deName, err := b.XdgService.SettingsGet("default-web-browser")
	if err != nil {
		return linkquisition.Browser{}, err
	}

	dePath, err := b.XdgService.GetDesktopEntryPathForFilename(deName)
	if err != nil {
		return linkquisition.Browser{}, err
	}

	desktopEntry, err := b.DesktopEntryService.CreateFromPath(dePath)
	if err != nil {
		return linkquisition.Browser{}, err
	}

	browser := linkquisition.Browser{
		Name:    desktopEntry.Name,
		Command: desktopEntry.Exec,
	}

	return browser, nil
}

func (b *BrowserService) OpenUrlWithDefaultBrowser(url string) error {
	cmd := exec.Command("xdg-open", url)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open URL `%s` with default browser: %v", url, err)
	}

	return nil
}

func (b *BrowserService) OpenUrlWithBrowser(u string, browser *linkquisition.Browser) error {
	u = shellescape.Quote(u)

	command := browser.Command
	command = strings.ReplaceAll(command, "%u", u)
	command = strings.ReplaceAll(command, "%U", u)

	// now just execute the damn command
	cmd := exec.Command("sh", "-c", command)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open URL `%s` with browser `%s`: %v", u, browser.Name, err)
	}

	return nil
}

func (b *BrowserService) AreWeTheDefaultBrowser() bool {
	value, err := b.XdgService.SettingsCheck("default-web-browser", "linkquisition.desktop")
	if err != nil {
		return false
	}

	return value == "yes"
}

func (b *BrowserService) MakeUsTheDefaultBrowser() error {
	err := b.XdgService.SettingsSet("default-web-browser", "linkquisition.desktop")
	if err != nil {
		return fmt.Errorf("failed to set Linkquisition as the default browser: %v", err)
	}

	return nil
}

func (b *BrowserService) SetDefaultBrowser(browser linkquisition.Browser) error {
	return b.XdgService.SettingsSet("default-web-browser", browser.Name)
}

func (b *BrowserService) NewBrowser(command string) (linkquisition.Browser, error) {
	browser := linkquisition.Browser{
		Command: command,
	}

	dePath, err := b.XdgService.GetDesktopEntryPathForBinary(command)
	if err != nil {
		return browser, fmt.Errorf("failed to find desktop entry for binary %q: %v", command, err)
	}

	desktopEntry, err := b.DesktopEntryService.CreateFromPath(dePath)
	if err != nil {
		return browser, fmt.Errorf("failed to parse desktop entry for binary %q: %v", command, err)
	}

	browser.Name = desktopEntry.Name

	return browser, nil
}

// GetIconForBrowser returns the icon for the given browser
func (b *BrowserService) GetIconForBrowser(browser linkquisition.Browser) ([]byte, error) {
	return b.BrowserIconLoader.LoadIcon(browser)
}

// desktopEntryHasCategory checks if a .desktop file contains the given category
// by scanning the Categories= line without fully parsing the file.
func desktopEntryHasCategory(path, category string) bool { //nolint:unparam // keeping param for testability
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	prefix := "Categories="
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			return strings.Contains(line, category)
		}
	}
	return false
}
