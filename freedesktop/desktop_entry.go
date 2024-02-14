package freedesktop

import (
	"fmt"

	"gopkg.in/ini.v1"
)

// DesktopEntry represents the "Desktop Entry" -section of a .desktop file.
//
// See https://specifications.freedesktop.org/desktop-entry-spec/desktop-entry-spec-latest.html#desktop-entry-file
// @todo add support for translations
type DesktopEntry struct {
	Version        string
	Type           string
	Exec           string
	Terminal       bool
	XMultipleArgs  bool
	Icon           string
	StartupWMClass string
	Categories     string
	MimeType       string
	StartupNotify  bool
	Actions        string
	Name           string
}

type DesktopEntryService struct {
}

func (d *DesktopEntryService) CreateFromPath(path string) (*DesktopEntry, error) {
	inidata, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read the .desktop entry for %s: %v", path, err)
	}

	section := inidata.Section("Desktop Entry")
	if section == nil {
		return nil, fmt.Errorf("failed to read the [Desktop Entry] section for %s", path)
	}

	desktopEntry := &DesktopEntry{
		Version:        section.Key("Version").String(),
		Type:           section.Key("Type").String(),
		Exec:           section.Key("Exec").String(),
		Terminal:       section.Key("Terminal").MustBool(),
		XMultipleArgs:  section.Key("X-MultipleArgs").MustBool(),
		Icon:           section.Key("Icon").String(),
		StartupWMClass: section.Key("StartupWMClass").String(),
		Categories:     section.Key("Categories").String(),
		MimeType:       section.Key("MimeType").String(),
		StartupNotify:  section.Key("StartupNotify").MustBool(),
		Actions:        section.Key("Actions").String(),
		Name:           section.Key("Name").String(),
	}

	return desktopEntry, nil
}
