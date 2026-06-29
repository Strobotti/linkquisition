//go:build darwin

package darwin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"

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
	// Use the Swift helper via 'open' heuristics and system_profiler won't work cleanly.
	// Instead, use LSCopyAllHandlersForURLScheme via a small Swift script, or parse
	// the output of `lsregister -dump` which is too complex.
	// Pragmatic approach: use defaults + known browser bundle IDs + check what's installed.
	cmd := exec.Command("/usr/bin/python3", "-c", `
import json, subprocess, plistlib, os, glob

browsers = []
app_dirs = ["/Applications", os.path.expanduser("~/Applications")]

for app_dir in app_dirs:
    if not os.path.isdir(app_dir):
        continue
    for app in glob.glob(os.path.join(app_dir, "*.app")):
        plist_path = os.path.join(app, "Contents", "Info.plist")
        if not os.path.exists(plist_path):
            continue
        try:
            with open(plist_path, "rb") as f:
                plist = plistlib.load(f)
            url_types = plist.get("CFBundleURLTypes", [])
            for url_type in url_types:
                schemes = [s.lower() for s in url_type.get("CFBundleURLSchemes", [])]
                if "http" in schemes or "https" in schemes:
                    bundle_id = plist.get("CFBundleIdentifier", "")
                    name = plist.get("CFBundleDisplayName") or plist.get("CFBundleName") or os.path.basename(app).replace(".app", "")
                    browsers.append({"bundleId": bundle_id, "name": name, "path": app})
                    break
        except Exception:
            pass

print(json.dumps(browsers))
`)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to discover browsers: %v", err)
	}

	var results []struct {
		BundleID string `json:"bundleId"`
		Name     string `json:"name"`
		Path     string `json:"path"`
	}

	if err := json.Unmarshal(out.Bytes(), &results); err != nil {
		return nil, fmt.Errorf("failed to parse browser list: %v", err)
	}

	var entries []lsRegisterEntry
	for _, r := range results {
		entries = append(entries, lsRegisterEntry{
			BundleID: r.BundleID,
			Name:     r.Name,
			Path:     r.Path,
		})
	}

	return entries, nil
}

func (b *BrowserService) GetDefaultBrowser() (linkquisition.Browser, error) {
	// Read the default HTTP handler
	cmd := exec.Command("/usr/bin/python3", "-c", `
import subprocess, re
result = subprocess.run(["defaults", "read", "com.apple.LaunchServices/com.apple.launchservices.secure", "LSHandlers"], capture_output=True, text=True)
# Fallback: use the x-scheme-handler approach
import Foundation
from Foundation import NSBundle
# Actually simplest: use open command
import subprocess as sp
r = sp.run(["plutil", "-convert", "json", "-o", "-", "/Users/"+__import__("os").environ.get("USER","")+"//Library/Preferences/com.apple.LaunchServices/com.apple.launchservices.secure.plist"], capture_output=True, text=True)
`)
	// The above is too fragile. Use a simpler approach:
	cmd = exec.Command("/usr/bin/python3", "-c", `
import json, subprocess
# On macOS, we can get the default browser bundle ID from LaunchServices
# Using the 'defaults' command to read the handler for https
import plistlib, os

plist_path = os.path.expanduser("~/Library/Preferences/com.apple.LaunchServices/com.apple.launchservices.secure.plist")
bundle_id = "com.apple.Safari"  # default fallback

try:
    with open(plist_path, "rb") as f:
        data = plistlib.load(f)
    for handler in data.get("LSHandlers", []):
        if handler.get("LSHandlerURLScheme") == "https":
            bundle_id = handler.get("LSHandlerRoleAll", bundle_id)
            break
except Exception:
    pass

print(bundle_id)
`)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// Fallback to Safari
		return linkquisition.Browser{Name: "Safari", Command: "com.apple.Safari"}, nil
	}

	bundleID := strings.TrimSpace(out.String())
	if bundleID == "" {
		bundleID = "com.apple.Safari"
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
	cmd := exec.Command("mdfind", fmt.Sprintf("kMDItemCFBundleIdentifier == '%s'", bundleID))
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
	// Read Info.plist to find icon file name
	cmd := exec.Command("/usr/bin/python3", "-c", fmt.Sprintf(`
import plistlib, os
plist_path = "%s/Contents/Info.plist"
try:
    with open(plist_path, "rb") as f:
        data = plistlib.load(f)
    icon = data.get("CFBundleIconFile", "")
    if icon and not icon.endswith(".icns"):
        icon += ".icns"
    print(icon)
except Exception:
    print("")
`, appPath))

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}

	iconFile := strings.TrimSpace(out.String())
	if iconFile == "" {
		return ""
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
