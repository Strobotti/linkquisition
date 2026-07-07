//go:build linux

package freedesktop

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type XdgService struct {
}

func (x *XdgService) GetDesktopEntryPathForFilename(name string) (string, error) {
	paths := x.GetApplicationPaths()

	if len(paths) == 0 {
		return "", fmt.Errorf("no valid desktop entry paths found in $XDG_DATA_DIRS")
	}

	for _, path := range paths {
		desktopEntryPath := filepath.Join(path, name)
		if _, err := os.Stat(desktopEntryPath); err == nil {
			return desktopEntryPath, nil
		}
	}

	return "", fmt.Errorf("no .desktop entry found for %s", name)
}

func (x *XdgService) GetDesktopEntryPathForBinary(binary string) (string, error) {
	paths := x.GetApplicationPaths()

	if len(paths) == 0 {
		return "", fmt.Errorf("no valid desktop entry paths found in $XDG_DATA_DIRS")
	}

	baseName := filepath.Base(binary)

	for _, dir := range paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".desktop") {
				continue
			}

			entryPath := filepath.Join(dir, entry.Name())
			if desktopEntryExecMatches(entryPath, binary, baseName) {
				return entryPath, nil
			}
		}
	}

	return "", fmt.Errorf("no .desktop entry found for %s", binary)
}

func (x *XdgService) GetApplicationPaths() []string {
	datadirs, isset := os.LookupEnv("XDG_DATA_DIRS")
	if !isset {
		datadirs = "/usr/local/share/:/usr/share/"
	}

	var paths []string

	for _, datadir := range strings.Split(datadirs, ":") {
		desktopEntryPath := filepath.Join(datadir, "applications")

		if _, err := os.Stat(desktopEntryPath); err == nil {
			paths = append(paths, desktopEntryPath)
		}
	}
	return paths
}

func (x *XdgService) SettingsCheck(property, subProperty string) (string, error) {
	cmd := exec.Command("xdg-settings", "check", property, subProperty)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to call xdg-settings check: %v", err)
	}

	// TODO not exatcly sure if this is The Way™
	return strings.Trim(out.String(), "\n"), nil
}

func (x *XdgService) SettingsGet(property string) (string, error) {
	cmd := exec.Command("xdg-settings", "get", property)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to call xdg-settings get: %v", err)
	}
	return out.String(), nil
}

func (x *XdgService) SettingsSet(property, value string) error {
	cmd := exec.Command("xdg-settings", "set", property, value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to call xdg-settings set: %v", err)
	}
	return nil
}

// desktopEntryExecMatches checks if a .desktop file's Exec= line starts with
// the given binary path or its basename.
func desktopEntryExecMatches(path, binary, baseName string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	prefix := "Exec="
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			execValue := strings.TrimPrefix(line, prefix)
			return strings.HasPrefix(execValue, binary) || strings.HasPrefix(execValue, baseName)
		}
	}
	return false
}
