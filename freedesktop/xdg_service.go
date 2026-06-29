package freedesktop

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/google/shlex" // archived but available
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

	// as most entries will have this pattern: `/usr/bin/chromium --profile-directory=Default %U`
	// we have to remove the CLI args to find equivalent .desktop files
	binaryParts, err := shlex.Split(binary)
	if err != nil || len(binaryParts) == 0 {
		return "", fmt.Errorf("failed to parse binary string: %v", err)
	}
	// The first part is the actual binary path
	binaryPath := binaryParts[0]
	// Check if binaryPath must be cleaned of surrounding quotes (edge case)
	binaryPath = strings.Trim(binaryPath, `"`)

	// grep all the .desktop files in the paths for the binary basename and return the first match:
	pattern := fmt.Sprintf("^Exec=(%s|%s)", binaryPath, filepath.Base(binaryPath))
	grepArgs := []string{"-r", "-l", "-m", "1", "-E", pattern, "--include", "*.desktop"}
	grepArgs = append(grepArgs, paths...)
	cmd := exec.Command("grep", grepArgs...)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		fmt.Println(out.String())
		return "", fmt.Errorf("failed to call grep for determining a .desktop entry for %s: %v", binary, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(out.String()))

	if !scanner.Scan() {
		return "", fmt.Errorf("no .desktop entry found for %s", binary)
	}

	return scanner.Text(), nil
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
