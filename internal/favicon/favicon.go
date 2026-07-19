// Package favicon provides lazy favicon retrieval for URLs.
// It supports three strategies: direct (/favicon.ico), parsed (HTML link tag),
// and google (Google's favicon service).
package favicon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// StrategyDirect fetches /favicon.ico directly from the host.
	StrategyDirect = "direct"

	// StrategyParsed fetches the HTML page and parses the <link rel="icon"> tag.
	StrategyParsed = "parsed"

	// StrategyGoogle uses Google's favicon service (privacy concern: URL is sent to Google).
	StrategyGoogle = "google"

	fetchTimeout    = 5 * time.Second
	maxFaviconBytes = 256 * 1024 // 256 KB max favicon size
	cacheDirPerms   = 0755
	cacheFilePerms  = 0644
	cacheTTL        = 7 * 24 * time.Hour // 1 week

	googleFaviconURL = "https://t1.gstatic.com/faviconV2?client=SOCIAL&type=FAVICON&fallback_opts=TYPE,SIZE,URL&url=%s&size=32"
)

var linkIconRegex = regexp.MustCompile(
	`<link[^>]+rel=["'](?:icon|shortcut icon|apple-touch-icon)["'][^>]*href=["']([^"']+)["']`,
)

// Fetcher retrieves favicons using a configurable strategy.
type Fetcher struct {
	strategy string
	cacheDir string
	client   *http.Client
}

// NewFetcher creates a new favicon Fetcher with the given strategy and cache directory.
// If cacheDir is empty, caching is disabled.
func NewFetcher(strategy, cacheDir string) *Fetcher {
	return &Fetcher{
		strategy: strategy,
		cacheDir: cacheDir,
		client: &http.Client{
			Timeout: fetchTimeout,
		},
	}
}

// Fetch retrieves the favicon for the given URL.
// It checks the cache first (if enabled), then fetches using the configured strategy.
// Returns the image bytes or an error. The caller should handle errors gracefully
// (e.g. show a placeholder).
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("no host in URL")
	}

	// Check cache first
	if f.cacheDir != "" {
		if data, cacheErr := f.readCache(host); cacheErr == nil {
			return data, nil
		}
	}

	var data []byte

	switch f.strategy {
	case StrategyParsed:
		data, err = f.fetchParsed(ctx, parsed)
	case StrategyGoogle:
		data, err = f.fetchGoogle(ctx, rawURL)
	default:
		data, err = f.fetchDirect(ctx, parsed)
	}

	if err != nil {
		return nil, err
	}

	// Write to cache (best-effort)
	if f.cacheDir != "" {
		_ = f.writeCache(host, data)
	}

	return data, nil
}

// fetchDirect downloads {scheme}://{host}/favicon.ico
func (f *Fetcher) fetchDirect(ctx context.Context, parsed *url.URL) ([]byte, error) {
	faviconURL := fmt.Sprintf("%s://%s/favicon.ico", parsed.Scheme, parsed.Host)
	return f.download(ctx, faviconURL)
}

// fetchParsed downloads the HTML page and parses <link rel="icon"> to find the favicon URL.
func (f *Fetcher) fetchParsed(ctx context.Context, parsed *url.URL) ([]byte, error) {
	// First try to parse the HTML for a link icon
	pageURL := fmt.Sprintf("%s://%s/", parsed.Scheme, parsed.Host)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Linkquisition/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		// Fallback to direct if page fetch fails
		return f.fetchDirect(ctx, parsed)
	}
	defer resp.Body.Close()

	// Read a limited amount of HTML to find the icon link
	// (don't download the entire page for large sites)
	const maxHTMLBytes = 64 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTMLBytes))
	if err != nil {
		return f.fetchDirect(ctx, parsed)
	}

	matches := linkIconRegex.FindSubmatch(body)
	const minRegexGroups = 2
	if len(matches) < minRegexGroups {
		// No icon link found, fall back to /favicon.ico
		return f.fetchDirect(ctx, parsed)
	}

	iconHref := string(matches[1])
	iconURL := resolveIconURL(iconHref, parsed)

	if iconURL == "" {
		return f.fetchDirect(ctx, parsed)
	}

	data, err := f.download(ctx, iconURL)
	if err != nil {
		// Fall back to direct if the parsed icon URL fails
		return f.fetchDirect(ctx, parsed)
	}

	return data, nil
}

// fetchGoogle uses Google's favicon service.
func (f *Fetcher) fetchGoogle(ctx context.Context, rawURL string) ([]byte, error) {
	fetchURL := fmt.Sprintf(googleFaviconURL, url.QueryEscape(rawURL))
	return f.download(ctx, fetchURL)
}

// download performs the actual HTTP GET and returns the body bytes.
// It validates that the response contains image data (by content-type header
// and magic bytes) to avoid passing HTML or other non-image content to the renderer.
func (f *Fetcher) download(ctx context.Context, fetchURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Linkquisition/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching favicon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("favicon returned HTTP %d", resp.StatusCode)
	}

	// Reject responses that are clearly not images based on Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !isImageContentType(contentType) {
		return nil, fmt.Errorf("non-image content type: %s", contentType)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFaviconBytes))
	if err != nil {
		return nil, fmt.Errorf("reading favicon: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty favicon response")
	}

	// Double-check with magic bytes — some servers send wrong Content-Type
	if !isImageData(data) {
		return nil, fmt.Errorf("response data is not a recognized image format")
	}

	return data, nil
}

// cacheKey returns a filesystem-safe cache key for a hostname.
func cacheKey(host string) string {
	h := sha256.Sum256([]byte(host))
	return hex.EncodeToString(h[:8])
}

// readCache reads a cached favicon if it exists and is not expired.
// TODO: there is no proactive eviction of old cache entries — expired entries are only
// removed lazily when accessed. A startup pruner or max cache size could be added if needed.
func (f *Fetcher) readCache(host string) ([]byte, error) {
	path := filepath.Join(f.cacheDir, cacheKey(host))

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if time.Since(info.ModTime()) > cacheTTL {
		_ = os.Remove(path)
		return nil, fmt.Errorf("cache expired")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Validate cached data is still a recognized image format
	// (protects against previously cached invalid responses)
	if !isImageData(data) {
		_ = os.Remove(path)
		return nil, fmt.Errorf("cached data is not a valid image, evicting")
	}

	return data, nil
}

// writeCache writes favicon bytes to the cache directory.
func (f *Fetcher) writeCache(host string, data []byte) error {
	if err := os.MkdirAll(f.cacheDir, cacheDirPerms); err != nil {
		return err
	}

	path := filepath.Join(f.cacheDir, cacheKey(host))
	return os.WriteFile(path, data, cacheFilePerms)
}

// resolveIconURL resolves a potentially relative icon href to an absolute URL.
func resolveIconURL(href string, base *url.URL) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}

	// Already absolute
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	// Protocol-relative
	if strings.HasPrefix(href, "//") {
		return base.Scheme + ":" + href
	}

	// Relative path
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}

	return base.ResolveReference(ref).String()
}

// isImageContentType checks if a Content-Type header value indicates an image.
func isImageContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))

	// Split on ";" to ignore charset and other parameters
	if idx := strings.IndexByte(ct, ';'); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}

	return strings.HasPrefix(ct, "image/")
}

// isImageData checks the first bytes of data for known image magic bytes.
// Supports PNG, JPEG, GIF, ICO, BMP, WebP, and SVG.
func isImageData(data []byte) bool {
	if len(data) < 4 { //nolint:mnd
		return false
	}

	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return true
	}

	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true
	}

	// GIF: GIF87a or GIF89a
	if len(data) >= 6 && string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a" {
		return true
	}

	// ICO: 00 00 01 00
	if data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 && data[3] == 0x00 {
		return true
	}

	// BMP: 42 4D
	if data[0] == 0x42 && data[1] == 0x4D {
		return true
	}

	// WebP: RIFF....WEBP
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return true
	}

	// SVG: look for an <svg tag in the first 512 bytes (XML preamble may precede it)
	checkLen := len(data)
	if checkLen > 512 { //nolint:mnd
		checkLen = 512
	}
	header := strings.ToLower(string(data[:checkLen]))
	if strings.Contains(header, "<svg") {
		return true
	}

	return false
}
