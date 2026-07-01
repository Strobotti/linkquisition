//go:build darwin

package darwin

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"howett.net/plist"

	"github.com/strobotti/linkquisition"
)

var _ linkquisition.BrowserService = (*BrowserService)(nil)

type BrowserService struct{}

type lsRegisterEntry struct {
	BundleID   string
	Name       string
	Path       string
	URLSchemes []string
}

func (b *BrowserService) GetAvailableBrowsers() ([]linkquisition.Browser, error) {
	entries, err := b.getHTTPHandlers()
	if err != nil {
		return nil, err
	}

	var browsers []linkquisition.Browser
	seen := make(map[string]bool)

	for _, entry := range entries {
		if seen[entry.BundleID] {
			continue
		}
		// Skip ourselves
		if strings.Contains(entry.BundleID, "linkquisition") {
			continue
		}
		seen[entry.BundleID] = true
		browsers = append(browsers, linkquisition.Browser{
			Name:    entry.Name,
			Command: entry.BundleID,
		})
	}

	return browsers, nil
}

func (b *BrowserService) getHTTPHandlers() ([]lsRegisterEntry, error) {
	homeDir, _ := os.UserHomeDir()
	appDirs := []string{"/Applications", filepath.Join(homeDir, "Applications")}

	var entries []lsRegisterEntry
	seen := make(map[string]bool)

	for _, appDir := range appDirs {
		dirEntries, err := os.ReadDir(appDir)
		if err != nil {
			continue
		}

		for _, entry := range dirEntries {
			if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".app") {
				continue
			}

			appPath := filepath.Join(appDir, entry.Name())
			plistPath := filepath.Join(appPath, "Contents", "Info.plist")

			if _, err := os.Stat(plistPath); err != nil {
				continue
			}

			bundleID, name, isHTTPHandler := parseBrowserPlist(plistPath, entry.Name())
			if !isHTTPHandler || seen[bundleID] {
				continue
			}

			seen[bundleID] = true
			entries = append(entries, lsRegisterEntry{
				BundleID: bundleID,
				Name:     name,
				Path:     appPath,
			})
		}
	}

	return entries, nil
}

// parseBrowserPlist extracts bundle ID, display name, and whether the app handles HTTP/HTTPS URLs.
// Parses the binary plist directly using the howett.net/plist library.
func parseBrowserPlist(plistPath, appDirName string) (bundleID, name string, isHTTPHandler bool) {
	f, err := os.Open(plistPath)
	if err != nil {
		return "", "", false
	}
	defer f.Close()

	var plistData struct {
		BundleID    string `plist:"CFBundleIdentifier"`
		DisplayName string `plist:"CFBundleDisplayName"`
		BundleName  string `plist:"CFBundleName"`
		URLTypes    []struct {
			Schemes []string `plist:"CFBundleURLSchemes"`
		} `plist:"CFBundleURLTypes"`
	}

	decoder := plist.NewDecoder(f)
	if err := decoder.Decode(&plistData); err != nil {
		return "", "", false
	}

	// Check if the app handles http or https
	for _, urlType := range plistData.URLTypes {
		for _, scheme := range urlType.Schemes {
			lower := strings.ToLower(scheme)
			if lower == "http" || lower == "https" {
				// Determine the display name
				name = plistData.DisplayName
				if name == "" {
					name = plistData.BundleName
				}
				if name == "" {
					name = strings.TrimSuffix(appDirName, ".app")
				}
				return plistData.BundleID, name, true
			}
		}
	}

	return "", "", false
}

func (b *BrowserService) GetDefaultBrowser() (linkquisition.Browser, error) {
	safariDefault := linkquisition.Browser{Name: "Safari", Command: "com.apple.Safari"}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return safariDefault, nil
	}

	plistPath := filepath.Join(homeDir, "Library", "Preferences",
		"com.apple.LaunchServices", "com.apple.launchservices.secure.plist")

	f, err := os.Open(plistPath)
	if err != nil {
		return safariDefault, nil
	}
	defer f.Close()

	var data struct {
		LSHandlers []struct {
			URLScheme      string `plist:"LSHandlerURLScheme"`
			HandlerRoleAll string `plist:"LSHandlerRoleAll"`
		} `plist:"LSHandlers"`
	}

	decoder := plist.NewDecoder(f)
	if err := decoder.Decode(&data); err != nil {
		return safariDefault, nil
	}

	bundleID := "com.apple.Safari"
	for _, handler := range data.LSHandlers {
		if handler.URLScheme == "https" && handler.HandlerRoleAll != "" {
			bundleID = handler.HandlerRoleAll
			break
		}
	}

	name := bundleIDToName(bundleID)
	return linkquisition.Browser{Name: name, Command: bundleID}, nil
}

func (b *BrowserService) OpenUrlWithDefaultBrowser(url string) error {
	cmd := exec.Command("open", url)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open URL `%s` with default browser: %v", url, err)
	}
	return nil
}

func (b *BrowserService) OpenUrlWithBrowser(url string, browser *linkquisition.Browser) error {
	cmd := exec.Command("open", "-b", browser.Command, url)
	if err := cmd.Run(); err != nil {
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
	// On macOS, setting the default browser requires the app to be a registered URL handler.
	// The OS will show a confirmation dialog. We use LSSetDefaultHandlerForURLScheme via Swift.
	script := `
import Foundation
import CoreServices
let bundleID = "com.strobotti.linkquisition" as CFString
LSSetDefaultHandlerForURLScheme("http" as CFString, bundleID)
LSSetDefaultHandlerForURLScheme("https" as CFString, bundleID)
`
	cmd := exec.Command("swift", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set as default browser: %v", err)
	}
	return nil
}

func (b *BrowserService) GetIconForBrowser(browser linkquisition.Browser) ([]byte, error) {
	// Find the app bundle by bundle ID and extract its icon
	appPath, err := getAppPathForBundleID(browser.Command)
	if err != nil {
		return nil, err
	}

	iconPath := getIconPathFromApp(appPath)
	if iconPath == "" {
		return nil, fmt.Errorf("no icon found for %s", browser.Name)
	}

	// Convert .icns to PNG using sips
	cmd := exec.Command("sips", "-s", "format", "png", iconPath, "--out", "/tmp/linkquisition-icon-tmp.png")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to convert icon: %v", err)
	}

	icon, err := fyne.LoadResourceFromPath("/tmp/linkquisition-icon-tmp.png")
	if err != nil {
		return nil, err
	}

	return icon.Content(), nil
}

func getAppPathForBundleID(bundleID string) (string, error) {
	// Pass the query as a single argument to mdfind — bundleID is not interpolated into a shell
	query := "kMDItemCFBundleIdentifier == '" + bundleID + "'"
	cmd := exec.Command("mdfind", query)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to find app for bundle ID %s: %v", bundleID, err)
	}

	path := strings.TrimSpace(strings.Split(out.String(), "\n")[0])
	if path == "" {
		return "", fmt.Errorf("app not found for bundle ID %s", bundleID)
	}

	return path, nil
}

func getIconPathFromApp(appPath string) string {
	plistPath := filepath.Join(appPath, "Contents", "Info.plist")

	f, err := os.Open(plistPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	var data struct {
		IconFile string `plist:"CFBundleIconFile"`
	}

	decoder := plist.NewDecoder(f)
	if err := decoder.Decode(&data); err != nil {
		return ""
	}

	iconFile := data.IconFile
	if iconFile == "" {
		return ""
	}

	if !strings.HasSuffix(iconFile, ".icns") {
		iconFile += ".icns"
	}

	return filepath.Join(appPath, "Contents", "Resources", iconFile)
}

func bundleIDToName(bundleID string) string {
	known := map[string]string{
		"com.apple.Safari":           "Safari",
		"com.google.Chrome":          "Google Chrome",
		"org.mozilla.firefox":        "Firefox",
		"com.microsoft.edgemac":      "Microsoft Edge",
		"com.brave.Browser":          "Brave Browser",
		"com.operasoftware.Opera":    "Opera",
		"com.vivaldi.Vivaldi":        "Vivaldi",
		"company.thebrowser.Browser": "Arc",
		"com.nickvision.Midori":      "Midori",
	}

	if name, ok := known[bundleID]; ok {
		return name
	}

	// Try to look it up
	appPath, err := getAppPathForBundleID(bundleID)
	if err == nil {
		return filepath.Base(strings.TrimSuffix(appPath, ".app"))
	}

	// Last resort: use the last part of the bundle ID
	parts := strings.Split(bundleID, ".")
	return parts[len(parts)-1]
}
