//go:build darwin

package darwin

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	icns "github.com/jackmordaunt/icns/v3"
	"howett.net/plist"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/browserprofiles"
	"github.com/strobotti/linkquisition/internal/launchenv"
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
		browser := linkquisition.Browser{Name: entry.Name, Command: entry.BundleID}
		profiles := browserprofiles.Detect(browser.Name)
		if len(profiles) == 0 {
			browsers = append(browsers, browser)
			continue
		}
		for _, profile := range profiles {
			profiledBrowser := browser
			profiledBrowser.Name = browser.Name + " — " + profile.Name
			profiledBrowser.ProfileName = profile.Name
			profiledBrowser.ProfilePath = profile.ID
			profiledBrowser.ProfileType = profile.Type
			profiledBrowser.ProfileIsDefault = profile.IsDefault
			browsers = append(browsers, profiledBrowser)
		}
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
			// Note: we intentionally do NOT check entry.IsDir() here. On modern macOS,
			// system apps like Safari are "firmlinked" and os.ReadDir reports them as
			// non-directories despite being valid .app bundles.
			if !strings.HasSuffix(entry.Name(), ".app") {
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
		BundleID       string `plist:"CFBundleIdentifier"`
		DisplayName    string `plist:"CFBundleDisplayName"`
		BundleName     string `plist:"CFBundleName"`
		ExecutableName string `plist:"CFBundleExecutable"`
		URLTypes       []struct {
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

func getExecutableNameFromApp(appPath string) (string, error) {
	file, err := os.Open(filepath.Join(appPath, "Contents", "Info.plist"))
	if err != nil {
		return "", err
	}
	defer file.Close()

	var appInfo struct {
		ExecutableName string `plist:"CFBundleExecutable"`
	}
	if err := plist.NewDecoder(file).Decode(&appInfo); err != nil {
		return "", err
	}
	if appInfo.ExecutableName == "" {
		return "", fmt.Errorf("CFBundleExecutable is missing from %s", appPath)
	}
	return appInfo.ExecutableName, nil
}

func (b *BrowserService) OpenUrlWithDefaultBrowser(url string) error {
	cmd := exec.Command("open", url)
	cmd.Env = launchenv.SanitizeEnviron(cmd.Environ())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open URL %#q with default browser: %v", url, err)
	}
	return nil
}

func (b *BrowserService) OpenUrlWithBrowser(url string, browser *linkquisition.Browser) error {
	privateArgument := ""
	if browser.PrivateMode {
		var ok bool
		privateArgument, ok = browserprofiles.PrivateArgument(browser.Name)
		if !ok {
			return fmt.Errorf("private browsing is not supported for browser %q", browser.Name)
		}
	}

	if browser.ProfileType == browserprofiles.TypeFirefox {
		appPath, err := getAppPathForBundleID(browser.Command)
		if err != nil {
			return fmt.Errorf("failed to find browser app for profile launch: %w", err)
		}
		binaryName, err := getExecutableNameFromApp(appPath)
		if err != nil {
			return fmt.Errorf("failed to find browser executable for profile launch: %w", err)
		}
		if _, err := os.Stat(filepath.Join(appPath, "Contents", "MacOS", binaryName)); err != nil {
			return fmt.Errorf("browser executable not found: %w", err)
		}
		binaryPath := filepath.Join(appPath, "Contents", "MacOS", binaryName)
		args := []string{"-P", browser.ProfileName}
		if privateArgument != "" {
			args = append(args, privateArgument)
		} else {
			args = append(args, "-new-window")
		}
		args = append(args, url)
		cmd := exec.Command(binaryPath, args...)
		cmd.Env = launchenv.SanitizeEnviron(cmd.Environ())
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to open URL %#q with browser %#q: %v", url, browser.Name, err)
		}
		return nil
	}

	if browser.ProfileType == browserprofiles.TypeChromium {
		appPath, err := getAppPathForBundleID(browser.Command)
		if err != nil {
			return fmt.Errorf("failed to find browser app for profile launch: %w", err)
		}
		binaryName, err := getExecutableNameFromApp(appPath)
		if err != nil {
			return fmt.Errorf("failed to find browser executable for profile launch: %w", err)
		}
		binaryPath := filepath.Join(appPath, "Contents", "MacOS", binaryName)
		args := []string{"--profile-directory=" + browser.ProfilePath}
		if privateArgument != "" {
			args = append(args, privateArgument)
		}
		args = append(args, url)
		cmd := exec.Command(binaryPath, args...)
		cmd.Env = launchenv.SanitizeEnviron(cmd.Environ())
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to open URL %#q with browser %#q: %v", url, browser.Name, err)
		}
		return nil
	}

	if browser.PrivateMode {
		cmd := exec.Command("open", "-b", browser.Command, "--args", privateArgument, url)
		cmd.Env = launchenv.SanitizeEnviron(cmd.Environ())
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to open URL %#q with browser %#q: %v", url, browser.Name, err)
		}
		return nil
	}
	cmd := exec.Command("open", "-b", browser.Command, url)
	cmd.Env = launchenv.SanitizeEnviron(cmd.Environ())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open URL %#q with browser %#q: %v", url, browser.Name, err)
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

	// Decode the .icns file natively and encode as PNG
	f, err := os.Open(iconPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open icon file: %v", err)
	}
	defer f.Close()

	img, err := icns.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode icns: %v", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode icon as PNG: %v", err)
	}

	return buf.Bytes(), nil
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
