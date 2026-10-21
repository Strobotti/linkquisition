package browserprofiles

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	TypeChromium = "chromium"
	TypeFirefox  = "firefox"

	platformDarwin  = "darwin"
	platformWindows = "windows"

	chromiumDefaultProfileID = "Default"
)

// PrivateArgument returns the browser's supported private-window command line argument.
func PrivateArgument(browserName string) (string, bool) {
	name := strings.ToLower(browserName)
	switch {
	case strings.Contains(name, "firefox"):
		return "-private-window", true
	case strings.Contains(name, "edge"):
		return "--inprivate", true
	case strings.Contains(name, "chrome") || strings.Contains(name, "google") ||
		strings.Contains(name, "chromium") || strings.Contains(name, "brave"):
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

// chromiumFamily holds the user-data folder name of a Chromium-based browser on each platform.
type chromiumFamily struct {
	darwin  string
	windows string
	linux   string
}

func (f chromiumFamily) relativeDir(platform string) string {
	switch platform {
	case platformDarwin:
		return f.darwin
	case platformWindows:
		return f.windows
	default:
		return f.linux
	}
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

	for _, dir := range chromiumProfileDirs(name, platform, home, getenv) {
		if profiles := detectChromium(dir); len(profiles) > 0 {
			return profiles
		}
	}

	return nil
}

func chromiumFamilyFor(name string) (chromiumFamily, bool) {
	switch {
	case strings.Contains(name, "edge"):
		return chromiumFamily{
			darwin:  "Microsoft Edge",
			windows: filepath.Join("Microsoft", "Edge", "User Data"),
			linux:   "microsoft-edge",
		}, true
	case strings.Contains(name, "brave"):
		return chromiumFamily{
			darwin:  filepath.Join("BraveSoftware", "Brave-Browser"),
			windows: filepath.Join("BraveSoftware", "Brave-Browser", "User Data"),
			linux:   filepath.Join("BraveSoftware", "Brave-Browser"),
		}, true
	case strings.Contains(name, "chromium"):
		return chromiumFamily{
			darwin:  "Chromium",
			windows: filepath.Join("Chromium", "User Data"),
			linux:   "chromium",
		}, true
	case strings.Contains(name, "chrome") || strings.Contains(name, "google"):
		return chromiumFamily{
			darwin:  filepath.Join("Google", "Chrome"),
			windows: filepath.Join("Google", "Chrome", "User Data"),
			linux:   "google-chrome",
		}, true
	default:
		return chromiumFamily{}, false
	}
}

func chromiumRoots(platform, home string, getenv func(string) string) []string {
	switch platform {
	case platformDarwin:
		return []string{filepath.Join(home, "Library", "Application Support")}
	case platformWindows:
		return nonEmpty(getenv("LOCALAPPDATA"))
	default:
		defaultConfig := filepath.Join(home, ".config")
		configHome := getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			return []string{defaultConfig}
		}
		roots := []string{configHome}
		if configHome != defaultConfig {
			roots = append(roots, defaultConfig)
		}
		return roots
	}
}

func chromiumProfileDirs(name, platform, home string, getenv func(string) string) []string {
	family, ok := chromiumFamilyFor(strings.ToLower(name))
	if !ok {
		return nil
	}

	relativeDir := family.relativeDir(platform)
	roots := chromiumRoots(platform, home, getenv)
	dirs := make([]string, 0, len(roots))
	for _, root := range roots {
		dirs = append(dirs, filepath.Join(root, relativeDir))
	}
	return dirs
}

func firefoxProfileDirs(platform, home string, getenv func(string) string) []string {
	switch platform {
	case platformDarwin:
		return []string{filepath.Join(home, "Library", "Application Support", "Firefox")}
	case platformWindows:
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
		return []string{filepath.Join(home, ".mozilla", "firefox"), filepath.Join(profileHome, "mozilla", "firefox")}
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
		if profileIDs[i] == chromiumDefaultProfileID {
			return true
		}
		if profileIDs[j] == chromiumDefaultProfileID {
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
		profiles = append(profiles, Profile{
			Name:      info.Name,
			ID:        id,
			Type:      TypeChromium,
			IsDefault: id == chromiumDefaultProfileID,
		})
	}
	return profiles
}

// firefoxProfileEntry is one [ProfileN] section of Firefox's profiles.ini.
type firefoxProfileEntry struct {
	name       string
	path       string
	isRelative bool
	isDefault  bool
}

func detectFirefox(profileDir string) []Profile {
	file, err := os.Open(filepath.Join(profileDir, "profiles.ini"))
	if err != nil {
		return nil
	}
	defer file.Close()

	entries, err := parseFirefoxProfilesINI(file)
	if err != nil {
		return nil
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].isDefault && !entries[j].isDefault
	})

	var profiles []Profile
	for _, entry := range entries {
		path, ok := resolveFirefoxProfilePath(profileDir, entry)
		if !ok || entry.name == "" {
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

// parseFirefoxProfilesINI returns the [Profile...] sections of a profiles.ini file.
func parseFirefoxProfilesINI(r io.Reader) ([]firefoxProfileEntry, error) {
	var entries []firefoxProfileEntry
	inProfile := false

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if section, ok := parseINISection(line); ok {
			inProfile = strings.HasPrefix(section, "Profile")
			if inProfile {
				entries = append(entries, firefoxProfileEntry{isRelative: true})
			}
			continue
		}
		if !inProfile {
			continue
		}
		if key, value, ok := strings.Cut(line, "="); ok {
			applyFirefoxProfileKey(&entries[len(entries)-1], key, value)
		}
	}
	return entries, scanner.Err()
}

func parseINISection(line string) (string, bool) {
	if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
		return "", false
	}
	return line[1 : len(line)-1], true
}

func applyFirefoxProfileKey(entry *firefoxProfileEntry, key, value string) {
	switch key {
	case "Name":
		entry.name = value
	case "Path":
		entry.path = value
	case "IsRelative":
		entry.isRelative = value != "0"
	case "Default":
		entry.isDefault = value == "1"
	}
}

// resolveFirefoxProfilePath returns the absolute profile directory, if it exists.
func resolveFirefoxProfilePath(profileDir string, entry firefoxProfileEntry) (string, bool) {
	if entry.path == "" {
		return "", false
	}
	path := entry.path
	if entry.isRelative {
		path = filepath.Join(profileDir, filepath.FromSlash(path))
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return path, true
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
