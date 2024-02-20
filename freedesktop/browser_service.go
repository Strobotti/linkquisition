package freedesktop

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/strobotti/linkquisition"
	"gopkg.in/alessio/shellescape.v1"
	"log"
	"os/exec"
	"strings"
)

var _ linkquisition.BrowserService = (*BrowserService)(nil)

type BrowserService struct {
	App                 linkquisition.Application
	XdgService          *XdgService
	DesktopEntryService *DesktopEntryService
}

func (b *BrowserService) GetAvailableBrowsers() ([]linkquisition.Browser, error) {
	paths := b.XdgService.GetApplicationPaths()

	if len(paths) == 0 {
		return nil, fmt.Errorf("no valid desktop entry paths found in $XDG_DATA_DIRS")
	}

	// grep all the .desktop files in the paths for the category "WebBrowser":
	grepArgs := []string{
		"-r",
		"-l",
		"-E",
		"^Categories=.*WebBrowser",
	}

	grepArgs = append(grepArgs, paths...)

	cmd := exec.Command("grep", grepArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch available browsers: %v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(out.String()))

	var browsers []linkquisition.Browser

	for scanner.Scan() {
		// skip Linkquisition as an available browser
		// TODO this should happen on a higher level
		if strings.Contains(scanner.Text(), "linkquisition.desktop") {
			continue
		}

		fmt.Printf("Found browser: %s\n", scanner.Text())
		desktopEntry, err := b.DesktopEntryService.CreateFromPath(scanner.Text())
		if err != nil {
			log.Fatal(err)
		}

		browser := linkquisition.Browser{
			Name:    desktopEntry.Name,
			Command: desktopEntry.Exec,
		}
		browsers = append(browsers, browser)
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

	b.App.GetLogger().Info(fmt.Sprintf("Opening URL `%s` with browser `%s` using command `%s`", u, browser.Name, command))

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
		log.Printf("failed to check if we are the default browser: %v", err)
		return false
	}

	fmt.Println("'" + value + "'")

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

func (b *BrowserService) NewBrowser(command string) linkquisition.Browser {
	browser := linkquisition.Browser{
		Command: command,
	}

	dePath, err := b.XdgService.GetDesktopEntryPathForBinary(command)
	if err != nil {
		log.Fatal(err)
	}

	desktopEntry, err := b.DesktopEntryService.CreateFromPath(dePath)
	if err != nil {
		log.Fatal(err)
	}

	browser.Name = desktopEntry.Name

	return browser
}
