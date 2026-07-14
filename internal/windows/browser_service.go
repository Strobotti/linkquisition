//go:build windows

package windows

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/windows/registry"

	"github.com/strobotti/linkquisition"
)

var _ linkquisition.BrowserService = (*BrowserService)(nil)

// BrowserService implements browser discovery and launching on Windows.
// It reads the registry for registered browser applications and uses
// cmd /c start to launch URLs.
type BrowserService struct{}

// registryBrowser holds information about a browser discovered from the registry.
type registryBrowser struct {
	Name    string
	Command string
}

func (b *BrowserService) GetAvailableBrowsers() ([]linkquisition.Browser, error) {
	var browsers []linkquisition.Browser
	seen := make(map[string]bool)

	// Check both HKLM and HKCU for StartMenuInternet entries
	for _, rootKey := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		entries, err := readStartMenuInternetEntries(rootKey)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			lowerCmd := strings.ToLower(entry.Command)
			if seen[lowerCmd] {
				continue
			}
			// Skip ourselves
			if strings.Contains(lowerCmd, "linkquisition") {
				continue
			}
			seen[lowerCmd] = true
			browsers = append(browsers, linkquisition.Browser{
				Name:    entry.Name,
				Command: entry.Command,
			})
		}
	}

	return browsers, nil
}

// readStartMenuInternetEntries reads browser entries from
// SOFTWARE\Clients\StartMenuInternet under the given root key.
func readStartMenuInternetEntries(rootKey registry.Key) ([]registryBrowser, error) {
	keyPath := `SOFTWARE\Clients\StartMenuInternet`
	k, err := registry.OpenKey(rootKey, keyPath, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer k.Close()

	subKeys, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}

	var entries []registryBrowser

	for _, subKeyName := range subKeys {
		entry, err := readBrowserEntry(rootKey, keyPath+`\`+subKeyName)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// readBrowserEntry reads a single browser's name and command from the registry.
func readBrowserEntry(rootKey registry.Key, keyPath string) (registryBrowser, error) {
	// Get the display name from the key's default value
	k, err := registry.OpenKey(rootKey, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return registryBrowser{}, err
	}

	name, _, _ := k.GetStringValue("")
	k.Close()

	if name == "" {
		// Fall back to the key name itself
		parts := strings.Split(keyPath, `\`)
		name = parts[len(parts)-1]
	}

	// Get the command from shell\open\command
	cmdKeyPath := keyPath + `\shell\open\command`
	cmdKey, err := registry.OpenKey(rootKey, cmdKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return registryBrowser{}, err
	}
	defer cmdKey.Close()

	command, _, err := cmdKey.GetStringValue("")
	if err != nil {
		return registryBrowser{}, err
	}

	// Clean up the command — remove quotes and trailing args
	command = strings.Trim(command, `"`)
	// Some entries have "%1" or similar at the end
	if idx := strings.Index(command, `" `); idx > 0 {
		command = command[:idx]
	}

	return registryBrowser{
		Name:    name,
		Command: command,
	}, nil
}

func (b *BrowserService) GetDefaultBrowser() (linkquisition.Browser, error) {
	// Read the default browser from UserChoice registry key
	keyPath := `SOFTWARE\Microsoft\Windows\Shell\Associations\UrlAssociations\https\UserChoice`
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return linkquisition.Browser{Name: "Default Browser"}, nil
	}
	defer k.Close()

	progID, _, err := k.GetStringValue("ProgId")
	if err != nil {
		return linkquisition.Browser{Name: "Default Browser"}, nil
	}

	name := progIDToName(progID)
	return linkquisition.Browser{Name: name, Command: progID}, nil
}

func (b *BrowserService) OpenUrlWithDefaultBrowser(url string) error {
	// Use rundll32 to open with system default — avoids shell injection
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open URL `%s` with default browser: %v", url, err)
	}
	return nil
}

func (b *BrowserService) OpenUrlWithBrowser(url string, browser *linkquisition.Browser) error {
	// The browser command is the path to the executable
	command := browser.Command

	// Handle the "%U" or "%u" placeholder pattern used in config
	if strings.Contains(command, "%U") || strings.Contains(command, "%u") {
		command = strings.ReplaceAll(command, "%U", url)
		command = strings.ReplaceAll(command, "%u", url)
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return fmt.Errorf("empty browser command for %s", browser.Name)
		}
		cmd := exec.Command(parts[0], parts[1:]...)
		return cmd.Start()
	}

	cmd := exec.Command(command, url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open URL `%s` with browser `%s`: %v", url, browser.Name, err)
	}
	return nil
}

func (b *BrowserService) AreWeTheDefaultBrowser() bool {
	defaultBrowser, err := b.GetDefaultBrowser()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(defaultBrowser.Command), "linkquisition")
}

func (b *BrowserService) MakeUsTheDefaultBrowser() error {
	// On Windows 10+, apps cannot silently set themselves as default.
	// We register our capabilities and then open the Default Apps settings page.
	if err := registerURLCapabilities(); err != nil {
		return fmt.Errorf("failed to register URL handler capabilities: %w", err)
	}

	// Open the Windows Default Apps settings page
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", "ms-settings:defaultapps")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open Default Apps settings: %w", err)
	}
	return nil
}

// registerURLCapabilities writes the necessary registry entries to register
// Linkquisition as a URL handler capable of handling http and https.
func registerURLCapabilities() error {
	exePath, err := selfExePath()
	if err != nil {
		return err
	}

	// Register under HKCU\Software\Clients\StartMenuInternet\Linkquisition
	clientBase := `SOFTWARE\Clients\StartMenuInternet\Linkquisition`
	if err := writeRegistryDefault(clientBase, "Linkquisition"); err != nil {
		return err
	}

	// Capabilities
	capPath := clientBase + `\Capabilities`
	if err := writeRegistryString(capPath, "ApplicationName", "Linkquisition"); err != nil {
		return err
	}
	if err := writeRegistryString(capPath, "ApplicationDescription", "A fast, configurable browser-picker"); err != nil {
		return err
	}

	// URL associations
	urlAssocPath := capPath + `\URLAssociations`
	if err := writeRegistryString(urlAssocPath, "http", "LinkquisitionURL"); err != nil {
		return err
	}
	if err := writeRegistryString(urlAssocPath, "https", "LinkquisitionURL"); err != nil {
		return err
	}

	// Register in RegisteredApplications
	regAppsPath := `SOFTWARE\RegisteredApplications`
	if err := writeRegistryString(regAppsPath, "Linkquisition", capPath); err != nil {
		return err
	}

	// Register the URL class
	classPath := `SOFTWARE\Classes\LinkquisitionURL`
	if err := writeRegistryDefault(classPath, "Linkquisition URL"); err != nil {
		return err
	}
	if err := writeRegistryString(classPath, "URL Protocol", ""); err != nil {
		return err
	}

	cmdPath := classPath + `\shell\open\command`
	commandStr := `"` + exePath + `" "%1"`
	if err := writeRegistryDefault(cmdPath, commandStr); err != nil {
		return err
	}

	return nil
}

func (b *BrowserService) GetIconForBrowser(_ linkquisition.Browser) ([]byte, error) {
	// TODO: Extract icons from .exe resources using shell32 APIs or a Go library.
	// For now, return an error so the UI falls back to a default icon.
	return nil, fmt.Errorf("icon extraction not yet implemented on Windows")
}

// selfExePath returns the full path to the currently running executable.
func selfExePath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to determine executable path: %w", err)
	}
	return path, nil
}

// writeRegistryDefault creates a key (and parents) and sets the default (unnamed) value.
func writeRegistryDefault(keyPath, value string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to create registry key %s: %w", keyPath, err)
	}
	defer k.Close()

	if err := k.SetStringValue("", value); err != nil {
		return fmt.Errorf("failed to set default value for %s: %w", keyPath, err)
	}
	return nil
}

// writeRegistryString creates a key (and parents) and sets a named string value.
func writeRegistryString(keyPath, name, value string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to create registry key %s: %w", keyPath, err)
	}
	defer k.Close()

	if err := k.SetStringValue(name, value); err != nil {
		return fmt.Errorf("failed to set value %s for %s: %w", name, keyPath, err)
	}
	return nil
}

// progIDToName maps common Windows browser ProgIDs to friendly names.
func progIDToName(progID string) string {
	known := map[string]string{
		"MSEdgeHTM":                            "Microsoft Edge",
		"ChromeHTML":                           "Google Chrome",
		"FirefoxURL-308046B0AF4A39CB":          "Firefox",
		"FirefoxURL":                           "Firefox",
		"BraveHTML":                            "Brave Browser",
		"OperaStable":                          "Opera",
		"VivaldiHTM.VIVALDI":                   "Vivaldi",
		"AppXq0fevzme2pys62n3e0fbqa7peapykr8v": "Microsoft Edge (UWP)",
	}

	for prefix, name := range known {
		if strings.HasPrefix(progID, prefix) {
			return name
		}
	}

	return progID
}
