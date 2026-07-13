package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/strobotti/linkquisition"
)

// defaultSources is the list of blocklist URLs used when no custom sources are configured
var defaultSources = []string{
	"https://urlhaus.abuse.ch/downloads/hostfile/",
	"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
}

// DefangPluginSettings holds the configuration for the defang plugin
type DefangPluginSettings struct {
	// Sources is a list of URLs to download blocklists from (hosts-file format)
	Sources []string `json:"sources,omitempty"`

	// UpdateInterval is how often the blocklists should be refreshed (default: "168h" = 7 days)
	UpdateInterval string `json:"updateInterval,omitempty"`

	// Action determines what happens when a blocked URL is detected: "warn", "block", or "log"
	Action string `json:"action,omitempty"`
}

const (
	actionBlock = "block"
	actionWarn  = "warn"
	actionLog   = "log"

	cacheDirPerms = 0700
)

var _ linkquisition.Plugin = (*defang)(nil)

// defang is a plugin that checks URLs against known-malicious domain blocklists
type defang struct {
	settings        DefangPluginSettings
	sources         []string
	updateInterval  time.Duration
	action          string
	blockedDomains  map[string]struct{}
	cacheDir        string
	serviceProvider linkquisition.PluginServiceProvider

	mu       sync.Mutex
	updating bool
	done     chan struct{}
}

func (p *defang) Metadata() linkquisition.PluginMetadata {
	return linkquisition.PluginMetadata{
		Name:        "Defang",
		Description: "Checks URLs against known-malicious domain blocklists and blocks or warns before opening",
		Author:      "Strobotti",
		Version:     "2.0.0",
		URL:         "https://github.com/Strobotti/linkquisition",
		Settings: []linkquisition.PluginSettingDescriptor{
			{
				Key:         "sources",
				Label:       "Blocklist Sources",
				Description: "URLs to download hosts-format blocklists from",
				Type:        linkquisition.SettingTypeStringList,
				Default:     defaultSources,
			},
			{
				Key:         "updateInterval",
				Label:       "Update Interval",
				Description: "How often to refresh the cached blocklists (Go duration format, default: 7 days)",
				Type:        linkquisition.SettingTypeDuration,
				Default:     "168h",
			},
			{
				Key:         "action",
				Label:       "Action",
				Description: "What to do when a blocked domain is detected",
				Type:        linkquisition.SettingTypeChoice,
				Default:     actionBlock,
				Options:     []string{actionBlock, actionWarn, actionLog},
			},
		},
	}
}

func (p *defang) Setup(serviceProvider linkquisition.PluginServiceProvider, config map[string]interface{}) error {
	p.serviceProvider = serviceProvider
	p.blockedDomains = make(map[string]struct{})
	p.action = actionBlock
	p.updateInterval = 168 * time.Hour // 7 days
	p.sources = defaultSources

	var settings DefangPluginSettings
	if err := mapstructure.Decode(config, &settings); err != nil {
		return fmt.Errorf("error decoding settings: %w", err)
	}

	p.settings = settings

	if len(settings.Sources) > 0 {
		p.sources = settings.Sources
	}

	if settings.UpdateInterval != "" {
		d, err := time.ParseDuration(settings.UpdateInterval)
		if err != nil {
			return fmt.Errorf("invalid updateInterval %q: %w", settings.UpdateInterval, err)
		}
		p.updateInterval = d
	}

	if settings.Action != "" {
		switch settings.Action {
		case actionWarn, actionBlock, actionLog:
			p.action = settings.Action
		default:
			return fmt.Errorf("unknown action: %q (must be %q, %q, or %q)", settings.Action, actionBlock, actionWarn, actionLog)
		}
	}

	// Set up cache directory next to config
	p.cacheDir = filepath.Join(serviceProvider.GetConfigFolderPath(), "defang")

	// Load cached blocklists from disk
	p.loadCachedLists()

	// Check if any lists need updating and start background fetch
	p.maybeStartUpdate()

	return nil
}

func (p *defang) ProcessURL(_ context.Context, address string) linkquisition.PluginResult {
	parsed, err := url.Parse(address)
	if err != nil {
		return linkquisition.PluginResult{URL: address, Action: linkquisition.ActionContinue, ContinueChain: true}
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return linkquisition.PluginResult{URL: address, Action: linkquisition.ActionContinue, ContinueChain: true}
	}

	if !p.isDomainBlocked(host) {
		return linkquisition.PluginResult{URL: address, Action: linkquisition.ActionContinue, ContinueChain: true}
	}

	p.serviceProvider.GetLogger().Warn(
		fmt.Sprintf("blocked URL: %s (domain %s is on blocklist)", address, host),
		"plugin", "defang",
		"action", p.action,
	)

	switch p.action {
	case actionBlock:
		return linkquisition.PluginResult{
			URL:     address,
			Action:  linkquisition.ActionBlock,
			Message: fmt.Sprintf("The domain %q was found on a malware/phishing blocklist.\n\nThe URL has been blocked.", host),
		}
	case actionWarn:
		return linkquisition.PluginResult{
			URL:     address,
			Action:  linkquisition.ActionWarn,
			Message: fmt.Sprintf("The domain %q was found on a malware/phishing blocklist.\n\nDo you want to open it anyway?", host),
		}
	case actionLog:
		// Log but still open the URL
		return linkquisition.PluginResult{URL: address, Action: linkquisition.ActionContinue, ContinueChain: true}
	default:
		return linkquisition.PluginResult{
			URL:     address,
			Action:  linkquisition.ActionBlock,
			Message: fmt.Sprintf("The domain %q was found on a malware/phishing blocklist.\n\nThe URL has been blocked.", host),
		}
	}
}

func (p *defang) Shutdown(ctx context.Context) {
	p.mu.Lock()
	updating := p.updating
	done := p.done
	p.mu.Unlock()

	if !updating || done == nil {
		return
	}

	p.serviceProvider.GetLogger().Debug("waiting for blocklist update to finish", "plugin", "defang")

	select {
	case <-done:
		p.serviceProvider.GetLogger().Debug("blocklist update finished", "plugin", "defang")
	case <-ctx.Done():
		p.serviceProvider.GetLogger().Warn("blocklist update timed out during shutdown", "plugin", "defang")
	}
}

// isDomainBlocked checks if a domain or any of its parent domains are in the blocklist
func (p *defang) isDomainBlocked(host string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check the exact domain and all parent domains
	parts := strings.Split(host, ".")
	for i := range parts {
		domain := strings.Join(parts[i:], ".")
		if _, ok := p.blockedDomains[domain]; ok {
			return true
		}
	}

	return false
}

// loadCachedLists loads blocklists from the cache directory
func (p *defang) loadCachedLists() {
	if _, err := os.Stat(p.cacheDir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(p.cacheDir)
	if err != nil {
		p.serviceProvider.GetLogger().Warn("error reading cache directory", "error", err.Error(), "plugin", "defang")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}

		filePath := filepath.Join(p.cacheDir, entry.Name())
		p.loadHostsFile(filePath)
	}

	p.serviceProvider.GetLogger().Debug(
		fmt.Sprintf("loaded %d blocked domains from cache", len(p.blockedDomains)),
		"plugin", "defang",
	)
}

// loadHostsFile parses a hosts-format file and adds domains to the blocklist
func (p *defang) loadHostsFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		p.serviceProvider.GetLogger().Warn("error opening hosts file", "path", path, "error", err.Error(), "plugin", "defang")
		return
	}
	defer file.Close()

	p.parseHostsReader(file)
}

// parseHostsReader parses a hosts-format reader and adds domains to the blocklist
func (p *defang) parseHostsReader(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Hosts file format: "0.0.0.0 domain.com" or "127.0.0.1 domain.com"
		fields := strings.Fields(line)
		if len(fields) < 2 { //nolint:mnd
			continue
		}

		ip := fields[0]
		if ip != "0.0.0.0" && ip != "127.0.0.1" {
			continue
		}

		domain := strings.ToLower(fields[1])

		// Skip localhost entries
		if domain == "localhost" || domain == "localhost.localdomain" ||
			domain == "local" || domain == "broadcasthost" ||
			domain == "ip6-localhost" || domain == "ip6-loopback" {
			continue
		}

		p.blockedDomains[domain] = struct{}{}
	}
}

// maybeStartUpdate checks if any cached list is stale and starts a background update
func (p *defang) maybeStartUpdate() {
	needsUpdate := false

	if _, err := os.Stat(p.cacheDir); os.IsNotExist(err) {
		// No cache at all — need to fetch
		needsUpdate = true
	} else {
		// Check if any source file is missing or older than updateInterval
		for i := range p.sources {
			fileName := fmt.Sprintf("source_%d.txt", i)
			filePath := filepath.Join(p.cacheDir, fileName)

			info, err := os.Stat(filePath)
			if err != nil || time.Since(info.ModTime()) > p.updateInterval {
				needsUpdate = true
				break
			}
		}
	}

	if !needsUpdate {
		return
	}

	p.mu.Lock()
	p.updating = true
	p.done = make(chan struct{})
	p.mu.Unlock()

	go p.fetchUpdates()
}

// fetchUpdates downloads all configured blocklist sources to the cache directory
func (p *defang) fetchUpdates() {
	defer func() {
		p.mu.Lock()
		p.updating = false
		close(p.done)
		p.mu.Unlock()
	}()

	p.serviceProvider.GetLogger().Debug("starting blocklist update", "plugin", "defang")

	if err := os.MkdirAll(p.cacheDir, cacheDirPerms); err != nil {
		p.serviceProvider.GetLogger().Warn("error creating cache directory", "error", err.Error(), "plugin", "defang")
		return
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for i, source := range p.sources {
		fileName := fmt.Sprintf("source_%d.txt", i)
		filePath := filepath.Join(p.cacheDir, fileName)

		if err := p.downloadSource(client, source, filePath); err != nil {
			p.serviceProvider.GetLogger().Warn(
				fmt.Sprintf("error downloading blocklist from %s", source),
				"error", err.Error(),
				"plugin", "defang",
			)
			continue
		}

		// Load the newly downloaded list into memory
		p.loadHostsFile(filePath)
	}

	p.mu.Lock()
	domainCount := len(p.blockedDomains)
	p.mu.Unlock()

	p.serviceProvider.GetLogger().Debug(
		fmt.Sprintf("blocklist update complete, %d domains blocked", domainCount),
		"plugin", "defang",
	)
}

// downloadSource downloads a single blocklist source to a file
func (p *defang) downloadSource(client *http.Client, sourceURL, destPath string) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, sourceURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, sourceURL)
	}

	tmpPath := destPath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("error writing file: %w", err)
	}

	file.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("error renaming temp file: %w", err)
	}

	return nil
}

// Plugin is the exported symbol loaded by the plugin system.
// plugin.Lookup("Plugin") returns *defang which satisfies linkquisition.Plugin (pointer receiver methods).
var Plugin defang

// NewForTesting creates a fresh defang instance for use in tests (avoids mutex copy issues with the global Plugin var)
func NewForTesting() linkquisition.Plugin {
	return &defang{}
}
