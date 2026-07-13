package safety

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	cacheDirPerms    = 0o755
	cacheFilePerms   = 0o600
	securityCacheDir = "security_cache"
)

// Cache provides file-based caching of security check results, separated by provider.
type Cache struct {
	dir string
	ttl time.Duration
}

type cachedEntry struct {
	Result   CheckResult `json:"result"`
	CachedAt time.Time   `json:"cachedAt"`
}

// NewCache creates a new security result cache for the given provider.
// The cache directory is: <configDir>/security_cache/<provider>/
func NewCache(configDir, provider string, ttl time.Duration) *Cache {
	return &Cache{
		dir: filepath.Join(configDir, securityCacheDir, provider),
		ttl: ttl,
	}
}

// Get retrieves a cached result for the given URL.
// Returns the result and true if a valid (non-expired) entry exists, or nil and false otherwise.
func (c *Cache) Get(url string) (*CheckResult, bool) {
	path := filepath.Join(c.dir, cacheKey(url))

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var entry cachedEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Corrupted cache entry — remove it
		_ = os.Remove(path)
		return nil, false
	}

	if time.Since(entry.CachedAt) > c.ttl {
		_ = os.Remove(path)
		return nil, false
	}

	return &entry.Result, true
}

// Put stores a check result in the cache for the given URL.
func (c *Cache) Put(url string, result *CheckResult) error {
	if err := os.MkdirAll(c.dir, cacheDirPerms); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	entry := cachedEntry{
		Result:   *result,
		CachedAt: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling cache entry: %w", err)
	}

	path := filepath.Join(c.dir, cacheKey(url))

	return os.WriteFile(path, data, cacheFilePerms)
}

// Clear removes all cached entries for this provider.
func (c *Cache) Clear() error {
	return os.RemoveAll(c.dir)
}

// PruneExpired removes all expired cache entries for this provider.
// Intended to be called in the background to keep the cache directory tidy.
func (c *Cache) PruneExpired() {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(c.dir, entry.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var cached cachedEntry
		if err := json.Unmarshal(data, &cached); err != nil {
			// Corrupted — remove it
			_ = os.Remove(path)
			continue
		}

		if time.Since(cached.CachedAt) > c.ttl {
			_ = os.Remove(path)
		}
	}
}

// ClearAll removes the entire security cache directory (all providers).
func ClearAll(configDir string) error {
	return os.RemoveAll(filepath.Join(configDir, securityCacheDir))
}

// cacheKey returns a filesystem-safe key for a URL.
func cacheKey(url string) string {
	h := sha256.Sum256([]byte(url))
	return hex.EncodeToString(h[:]) + ".json"
}
