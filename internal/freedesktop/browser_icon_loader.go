//go:build linux

package freedesktop

import (
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/resources"
)

// BrowserIconLoader loads browser icons and resolves icon names to file paths.
type BrowserIconLoader interface {
	LoadIcon(browser linkquisition.Browser) ([]byte, error)
	// ResolveIconName resolves an icon name (from a .desktop file's Icon= field) to an
	// absolute file path. If the name is already an absolute path and the file exists,
	// it is returned as-is. Otherwise, it searches standard icon directories.
	// Returns an empty string if the icon cannot be found.
	ResolveIconName(iconName string) string
}

// DefaultBrowserIconLoader is the standard implementation of BrowserIconLoader.
type DefaultBrowserIconLoader struct {
	XdgService          *XdgService
	DesktopEntryService *DesktopEntryService
}

func (l *DefaultBrowserIconLoader) LoadIcon(browser linkquisition.Browser) ([]byte, error) {
	// If the browser has a cached icon path, use it directly
	if browser.IconPath != "" {
		if icon, errLoad := fyne.LoadResourceFromPath(browser.IconPath); errLoad == nil {
			return icon.Content(), nil
		}
	}

	// Fall back to resolving from .desktop entry
	iconPath := l.resolveIconPathForCommand(browser.Command)
	if iconPath != "" {
		if icon, errLoad := fyne.LoadResourceFromPath(iconPath); errLoad == nil {
			return icon.Content(), nil
		}
	}

	// As a fallback we'll use the bundled application icon
	return resources.LinkquisitionIcon.Content(), nil
}

// resolveIconPathForCommand finds the icon path for a browser command by looking up
// the .desktop entry and resolving its Icon field to an absolute path.
func (l *DefaultBrowserIconLoader) resolveIconPathForCommand(command string) string {
	dePath, err := l.XdgService.GetDesktopEntryPathForBinary(command)
	if err != nil {
		return ""
	}

	desktopEntry, err := l.DesktopEntryService.CreateFromPath(dePath)
	if err != nil {
		return ""
	}

	return l.ResolveIconName(desktopEntry.Icon)
}

// ResolveIconName resolves an icon name (from a .desktop file's Icon= field) to an
// absolute file path. If the name is already an absolute path and the file exists,
// it is returned as-is. Otherwise, it searches standard icon directories.
// Returns an empty string if the icon cannot be found.
func (l *DefaultBrowserIconLoader) ResolveIconName(iconName string) string {
	if iconName == "" {
		return ""
	}

	// If the icon field is already a full path, use it directly if it exists
	if filepath.IsAbs(iconName) {
		if _, err := os.Stat(iconName); err == nil {
			return iconName
		}
		return ""
	}

	// Search for the icon by name in standard icon directories
	return findIconByName(iconName)
}

// findIconByName searches for an icon file matching the given name (without extension)
// in standard icon directories, including user-local paths.
func findIconByName(iconName string) string {
	iconDirs := getIconSearchDirs()

	for _, dir := range iconDirs {
		if path := searchDirForIcon(dir, iconName); path != "" {
			return path
		}
	}

	return ""
}

// getIconSearchDirs returns the list of directories to search for icons,
// including user-local directories.
func getIconSearchDirs() []string {
	var dirs []string

	// User-local icons ($XDG_DATA_HOME/icons/)
	dataHome, isset := os.LookupEnv("XDG_DATA_HOME")
	if !isset {
		if home, err := os.UserHomeDir(); err == nil {
			dataHome = filepath.Join(home, ".local", "share")
		}
	}
	if dataHome != "" {
		dirs = append(dirs, filepath.Join(dataHome, "icons"))
	}

	// System icon directories
	dirs = append(dirs, "/usr/share/icons", "/usr/share/pixmaps")

	return dirs
}

// searchDirForIcon walks a directory tree looking for an icon file matching the given name.
func searchDirForIcon(dir, iconName string) string {
	var foundPath string

	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
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

	return foundPath
}
