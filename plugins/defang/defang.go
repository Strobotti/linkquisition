package main

import (
	"bufio"
	"context"
	"errors"
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
	"github.com/strobotti/linkquisition/internal/defangconfig"
)

// defaultSources is the list of blocklist URLs used when no custom sources are configured
var defaultSources = defangconfig.DefaultSources()

// DefangPluginSettings holds the configuration for the defang plugin
type DefangPluginSettings struct {
	// Sources is a list of URLs to download blocklists from (hosts-file format)
	Sources []string `json:"sources,omitempty"`

	// UpdateInterval is how often the blocklists should be refreshed (default: "24h" = 1 day)
	UpdateInterval string `json:"updateInterval,omitempty"`

	// Action determines what happens when a blocked URL is detected: "warn", "block", or "log"
	Action string `json:"action,omitempty"`
}

const (
	actionBlock = "block"
	actionWarn  = "warn"
	actionLog   = "log"

	cacheDirPerms = 0700

	etagHeader            = "ETag"
	lastModifiedHeader    = "Last-Modified"
	ifNoneMatchHeader     = "If-None-Match"
	ifModifiedSinceHeader = "If-Modified-Since"
	httpRequestFailed     = "HTTP request failed"
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
		Author:      "Juha Jantunen",
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
				Description: "How often to refresh the cached blocklists (Go duration format, default: 24 hours)",
				Type:        linkquisition.SettingTypeDuration,
				Default:     "24h",
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
	p.updateInterval = 24 * time.Hour // 1 day
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

// loadCachedLists atomically replaces the in-memory list with the current sources.
// Legacy index-based files are retained as a fallback until each source is fetched.
func (p *defang) loadCachedLists() {
	domains := make(map[string]struct{})
	for i, source := range p.sources {
		path, _, _ := p.sourcePaths(source)
		file, err := os.Open(path)
		if os.IsNotExist(err) {
			path = filepath.Join(p.cacheDir, fmt.Sprintf("source_%d.txt", i))
			file, err = os.Open(path)
		}
		if err != nil {
			if !os.IsNotExist(err) {
				p.serviceProvider.GetLogger().Warn("error opening hosts file", "path", path, "error", err.Error(), "plugin", "defang")
			}
			continue
		}
		parseHostsReader(file, domains)
		_ = file.Close()
	}
	p.mu.Lock()
	p.blockedDomains = domains
	p.mu.Unlock()
	p.serviceProvider.GetLogger().Debug(fmt.Sprintf("loaded %d blocked domains from cache", len(domains)), "plugin", "defang")
}

// parseHostsReader parses a hosts-format reader and adds domains to the blocklist.
func parseHostsReader(reader io.Reader, domains map[string]struct{}) {
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

		domains[domain] = struct{}{}
	}
}

// maybeStartUpdate schedules a background pass; each source is checked independently.
func (p *defang) maybeStartUpdate() {
	p.mu.Lock()
	if p.updating {
		p.mu.Unlock()
		return
	}
	p.updating = true
	p.done = make(chan struct{})
	p.mu.Unlock()
	go p.fetchUpdates()
}

func (p *defang) fetchUpdates() {
	defer func() {
		p.mu.Lock()
		p.updating = false
		close(p.done)
		p.mu.Unlock()
	}()

	if err := os.MkdirAll(p.cacheDir, cacheDirPerms); err != nil {
		p.serviceProvider.GetLogger().Warn("error creating cache directory", "error", err.Error(), "plugin", "defang")
		return
	}
	client := &http.Client{Timeout: 30 * time.Second}
	for _, source := range p.sources {
		p.updateSource(client, source)
	}
	p.loadCachedLists()
}

func (p *defang) updateSource(client *http.Client, source string) {
	cachePath, statePath, lockPath := p.sourcePaths(source)
	lock, err := trySourceLock(lockPath)
	if err != nil {
		// Another process may already be updating this source.
		return
	}
	defer unlockSource(lock)
	state, err := readSourceState(statePath)
	if err != nil {
		p.serviceProvider.GetLogger().Warn("error reading blocklist state", "error", err.Error(), "plugin", "defang")
		return
	}
	now := time.Now()
	if !sourceDue(&state, cachePath, p.updateInterval, now) {
		return
	}
	// Persist the attempt before networking, so an interrupted process doesn't
	// immediately retry on every link open.
	state.LastAttempt = now
	state.Failures++
	state.NextAttempt = now.Add(retryDelay(state.Failures))
	if writeErr := writeSourceState(statePath, &state); writeErr != nil {
		p.serviceProvider.GetLogger().Warn("error recording blocklist attempt", "error", writeErr.Error(), "plugin", "defang")
		return
	}
	var etag, lastModified string
	if _, statErr := os.Stat(cachePath); statErr == nil {
		etag, lastModified = state.ETag, state.LastModified
	}
	result, err := p.downloadSource(client, source, cachePath, etag, lastModified)
	if err != nil {
		var httpErr *sourceHTTPError
		if errors.As(err, &httpErr) {
			state.HTTPStatus = httpErr.status
			state.Gone = httpErr.status == http.StatusGone
		} else {
			state.HTTPStatus = 0
		}
		if stateErr := writeSourceState(statePath, &state); stateErr != nil {
			p.serviceProvider.GetLogger().Warn("error recording blocklist failure", "error", stateErr.Error(), "plugin", "defang")
		}
		// Network errors may embed the URL, including credentials or query parameters.
		p.serviceProvider.GetLogger().Warn(
			"error downloading blocklist",
			"source_id", sourceID(source),
			"http_status", state.HTTPStatus,
			"plugin", "defang",
		)
		return
	}
	state.LastSuccess = time.Now()
	state.Failures = 0
	state.HTTPStatus = 0
	state.Gone = false
	state.NextAttempt = time.Time{}
	if result.NotModified {
		if result.ETag != "" {
			state.ETag = result.ETag
		}
		if result.LastModified != "" {
			state.LastModified = result.LastModified
		}
	} else {
		state.ETag = result.ETag
		state.LastModified = result.LastModified
	}
	if writeErr := writeSourceState(statePath, &state); writeErr != nil {
		p.serviceProvider.GetLogger().Warn("error recording blocklist success", "error", writeErr.Error(), "plugin", "defang")
	}
}

// downloadSource fetches a source, using conditional requests when cached validators are available.
func (p *defang) downloadSource(client *http.Client, sourceURL, destPath, etag, lastModified string) (sourceDownloadResult, error) {
	req, err := newSourceRequest(sourceURL, etag, lastModified)
	if err != nil {
		return sourceDownloadResult{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return sourceDownloadResult{}, fmt.Errorf(httpRequestFailed+": %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		result := sourceDownloadResult{
			NotModified:  true,
			ETag:         resp.Header.Get(etagHeader),
			LastModified: resp.Header.Get(lastModifiedHeader),
		}
		_ = resp.Body.Close()
		if _, statErr := os.Stat(destPath); statErr == nil {
			return result, nil
		}
		// A validator without a corresponding cache is unusable; retry unconditionally.
		req, err = newSourceRequest(sourceURL, "", "")
		if err != nil {
			return sourceDownloadResult{}, err
		}
		resp, err = client.Do(req)
		if err != nil {
			return sourceDownloadResult{}, fmt.Errorf(httpRequestFailed+": %w", err)
		}
	}
	return saveSourceResponse(resp, destPath)
}

func newSourceRequest(sourceURL, etag, lastModified string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, sourceURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	if etag != "" {
		req.Header.Set(ifNoneMatchHeader, etag)
	}
	if lastModified != "" {
		req.Header.Set(ifModifiedSinceHeader, lastModified)
	}
	return req, nil
}

func saveSourceResponse(resp *http.Response, destPath string) (sourceDownloadResult, error) {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return sourceDownloadResult{}, &sourceHTTPError{status: resp.StatusCode}
	}
	tmpPath := destPath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return sourceDownloadResult{}, fmt.Errorf("error creating temp file: %w", err)
	}
	if _, err := io.Copy(file, resp.Body); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return sourceDownloadResult{}, fmt.Errorf("error writing file: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return sourceDownloadResult{}, fmt.Errorf("error closing file: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return sourceDownloadResult{}, fmt.Errorf("error renaming file: %w", err)
	}
	return sourceDownloadResult{ETag: resp.Header.Get(etagHeader), LastModified: resp.Header.Get(lastModifiedHeader)}, nil
}

// Plugin is the exported symbol loaded by the plugin system.
// plugin.Lookup("Plugin") returns *defang which satisfies linkquisition.Plugin (pointer receiver methods).
var Plugin defang

// NewForTesting creates a fresh defang instance for use in tests (avoids mutex copy issues with the global Plugin var).
func NewForTesting() linkquisition.Plugin {
	return &defang{}
}
