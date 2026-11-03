// Package probecache stores recent download probes per node and target so
// download mode does not repeatedly spend data on the same candidates.
package probecache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/probe"
)

const (
	// DefaultTTL is how long a successful probe is reused.
	DefaultTTL = 20 * time.Minute
	// FailureTTL keeps failed reachability checks short so a recovered node is
	// not excluded for the full success window.
	FailureTTL = 5 * time.Minute
	maxEntries = 500
)

// Cache is a small JSON-backed probe cache.
type Cache struct {
	mu      sync.Mutex
	path    string
	entries map[string]probe.Result
}

// DefaultPath returns ~/.local/share/nodepulse/probe-cache.json.
func DefaultPath() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "nodepulse-probe-cache.json"
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "nodepulse", "probe-cache.json")
}

// Open loads a cache; a missing or malformed file starts empty.
func Open(path string) (*Cache, error) {
	if path == "" {
		path = DefaultPath()
	}
	c := &Cache{path: path, entries: map[string]probe.Result{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	_ = json.Unmarshal(data, &c.entries)
	if c.entries == nil {
		c.entries = map[string]probe.Result{}
	}
	return c, nil
}

// Key identifies one node and target file path.
func Key(node, host, path string) string {
	return node + "|" + host + "|" + path
}

// Get returns a fresh cached probe. Errors use a shorter TTL than successes.
func (c *Cache) Get(key string) (probe.Result, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	result, ok := c.entries[key]
	if !ok {
		return probe.Result{}, false
	}
	ttl := DefaultTTL
	if !result.OK {
		ttl = FailureTTL
	}
	if time.Since(result.MeasuredAt) > ttl {
		delete(c.entries, key)
		return probe.Result{}, false
	}
	result.Cached = true
	return result, true
}

// Put stores one probe result.
func (c *Cache) Put(key string, result probe.Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if result.MeasuredAt.IsZero() {
		result.MeasuredAt = time.Now()
	}
	c.entries[key] = result
	if len(c.entries) > maxEntries {
		// Drop the oldest entries first.
		type pair struct {
			key string
			at  time.Time
		}
		all := make([]pair, 0, len(c.entries))
		for k, v := range c.entries {
			all = append(all, pair{k, v.MeasuredAt})
		}
		for len(all) > maxEntries {
			oldest := 0
			for i := 1; i < len(all); i++ {
				if all[i].at.Before(all[oldest].at) {
					oldest = i
				}
			}
			delete(c.entries, all[oldest].key)
			all = append(all[:oldest], all[oldest+1:]...)
		}
	}
}

// Save writes the cache atomically.
func (c *Cache) Save() error {
	c.mu.Lock()
	data, err := json.MarshalIndent(c.entries, "", "  ")
	path := c.path
	c.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".probe-cache-*.json")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace probe cache: %w", err)
	}
	return nil
}
