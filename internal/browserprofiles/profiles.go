package browserprofiles

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	TypeChromium = "chromium"
	TypeFirefox  = "firefox"
)

// PrivateArgument returns the browser's supported private-window command line argument.
func PrivateArgument(browserName string) (string, bool) {
	name := strings.ToLower(browserName)
	switch {
	case strings.Contains(name, "firefox"):
		return "-private-window", true
	case strings.Contains(name, "chrome") || strings.Contains(name, "google") || strings.Contains(name, "chromium") ||
		strings.Contains(name, "edge") || strings.Contains(name, "brave"):
		return "--incognito", true
	default:
		return "", false
	}
}

// Profile describes a browser profile that can be selected as a launch target.
type Profile struct {
	Name      string
	ID        string
	Type      string
	IsDefault bool
}

type chromiumLocalState struct {
	Profile struct {
		InfoCache map[string]struct {
			Name string `json:"name"`
		} `json:"info_cache"`
	} `json:"profile"`
}

// Detect finds browser profiles in the standard configuration locations for the
// current platform. Unknown browsers and inaccessible profile files are ignored.
func Detect(browserName string) []Profile {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	return DetectForPlatform(browserName, runtime.GOOS, home, os.Getenv)
}

// DetectForPlatform discovers profiles using the supplied platform, home folder,
// and environment lookup. It is exported to keep platform-specific paths testable.
func DetectForPlatform(browserName, platform, home string, getenv func(string) string) []Profile {
	name := strings.ToLower(browserName)
	if strings.Contains(name, "firefox") {
		for _, dir := range firefoxProfileDirs(platform, home, getenv) {
			if profiles := detectFirefox(dir); len(profiles) > 0 {
				return profiles
			}
		}
		return nil
	}

	browserDirs := chromiumProfileDirs(name, platform, home, getenv)
	for _, dir := range browserDirs {
		profiles := detectChromium(dir)
		if len(profiles) > 0 {
			return profiles
		}
	}

	return nil
}

func chromiumProfileDirs(name, platform, home string, getenv func(string) string) []string {
	name = strings.ToLower(name)
	var relativeDir string
	switch {
	case strings.Contains(name, "edge"):
		switch platform {
		case "darwin":
			relativeDir = filepath.Join("Microsoft Edge")
		case "windows":
			relativeDir = filepath.Join("Microsoft", "Edge", "User Data")
		default:
			relativeDir = "microsoft-edge"
		}
	case strings.Contains(name, "brave"):
		switch platform {
		case "darwin":
			relativeDir = filepath.Join("BraveSoftware", "Brave-Browser")
		case "windows":
			relativeDir = filepath.Join("BraveSoftware", "Brave-Browser", "User Data")
		default:
			relativeDir = filepath.Join("BraveSoftware", "Brave-Browser")
		}
	case strings.Contains(name, "chromium"):
		switch platform {
		case "darwin":
			relativeDir = "Chromium"
		case "windows":
			relativeDir = filepath.Join("Chromium", "User Data")
		default:
			relativeDir = "chromium"
		}
	case strings.Contains(name, "chrome") || strings.Contains(name, "google"):
		switch platform {
		case "darwin":
			relativeDir = filepath.Join("Google", "Chrome")
		case "windows":
			relativeDir = filepath.Join("Google", "Chrome", "User Data")
		default:
			relativeDir = "google-chrome"
		}
	default:
		return nil
	}

	var roots []string
	switch platform {
	case "darwin":
		roots = []string{filepath.Join(home, "Library", "Application Support")}
	case "windows":
		roots = nonEmpty(getenv("LOCALAPPDATA"))
	default:
		configHome := getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(home, ".config")
		}
		roots = []string{configHome}
		if configHome != filepath.Join(home, ".config") {
			roots = append(roots, filepath.Join(home, ".config"))
		}
	}

	dirs := make([]string, 0, len(roots))
	for _, root := range roots {
		dirs = append(dirs, filepath.Join(root, relativeDir))
	}
	return dirs
}

func firefoxProfileDirs(platform, home string, getenv func(string) string) []string {
	switch platform {
	case "darwin":
		return []string{filepath.Join(home, "Library", "Application Support", "Firefox")}
	case "windows":
		roaming := getenv("APPDATA")
		if roaming == "" {
			roaming = filepath.Join(home, "AppData", "Roaming")
		}
		return []string{filepath.Join(roaming, "Mozilla", "Firefox")}
	default:
		profileHome := getenv("XDG_CONFIG_HOME")
		if profileHome == "" {
			profileHome = filepath.Join(home, ".config")
		}
		dirs := []string{filepath.Join(home, ".mozilla", "firefox"), filepath.Join(profileHome, "mozilla", "firefox")}
		return dirs
	}
}

func detectChromium(userDataDir string) []Profile {
	data, err := os.ReadFile(filepath.Join(userDataDir, "Local State"))
	if err != nil {
		return nil
	}

	var state chromiumLocalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}

	profileIDs := make([]string, 0, len(state.Profile.InfoCache))
	for id := range state.Profile.InfoCache {
		profileIDs = append(profileIDs, id)
	}
	sort.Slice(profileIDs, func(i, j int) bool {
		if profileIDs[i] == "Default" {
			return true
		}
		if profileIDs[j] == "Default" {
			return false
		}
		return profileIDs[i] < profileIDs[j]
	})

	var profiles []Profile
	for _, id := range profileIDs {
		info := state.Profile.InfoCache[id]
		if info.Name == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(userDataDir, id)); err != nil {
			continue
		}
		profiles = append(profiles, Profile{Name: info.Name, ID: id, Type: TypeChromium, IsDefault: id == "Default"})
	}
	return profiles
}

func detectFirefox(profileDir string) []Profile {
	file, err := os.Open(filepath.Join(profileDir, "profiles.ini"))
	if err != nil {
		return nil
	}
	defer file.Close()

	type profileEntry struct {
		name       string
		path       string
		isRelative bool
		isDefault  bool
		isProfile  bool
	}
	var entries []profileEntry
	var current *profileEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if !strings.HasPrefix(section, "Profile") {
				current = nil
				continue
			}
			entries = append(entries, profileEntry{isProfile: true, isRelative: true})
			current = &entries[len(entries)-1]
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if current == nil {
			continue
		}
		switch key {
		case "Name":
			current.name = value
		case "Path":
			current.path = value
		case "IsRelative":
			current.isRelative = value != "0"
		case "Default":
			current.isDefault = value == "1"
		}
	}
	if scanner.Err() != nil {
		return nil
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].isDefault && !entries[j].isDefault
	})

	var profiles []Profile
	for _, entry := range entries {
		if !entry.isProfile || entry.name == "" || entry.path == "" {
			continue
		}
		path := entry.path
		if entry.isRelative {
			path = filepath.Join(profileDir, filepath.FromSlash(path))
		}
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			continue
		}
		profiles = append(profiles, Profile{
			Name:      entry.name,
			ID:        path,
			Type:      TypeFirefox,
			IsDefault: entry.isDefault,
		})
	}
	return profiles
}

func nonEmpty(paths ...string) []string {
	var result []string
	for _, path := range paths {
		if path != "" {
			result = append(result, path)
		}
	}
	return result
}
