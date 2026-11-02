package subscription

import (
	"fmt"
	"io"
	"os"

	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/share"
)

// Load reads a Clash / mihomo YAML subscription or a base64 share-link
// subscription from a local file.
func Load(path string) (*config.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open subscription: %w", err)
	}
	defer f.Close()
	return LoadReader(f)
}

// LoadReader parses a subscription from a reader. Clash / mihomo YAML is tried
// first; if it yields no proxies, base64 / newline-separated v2ray share links
// are tried.
func LoadReader(r io.Reader) (*config.Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read subscription: %w", err)
	}
	cfg, parseErr := config.Parse(data)
	if parseErr == nil && len(cfg.Proxies) > 0 {
		return cfg, nil
	}
	if proxies, shareErr := share.ParseSubscription(data); shareErr == nil && len(proxies) > 0 {
		return &config.Config{Proxies: proxies}, nil
	}
	if parseErr != nil {
		return nil, fmt.Errorf("parse subscription: %w", parseErr)
	}
	return nil, fmt.Errorf("subscription has no proxies (supported: Clash / mihomo YAML, base64 share links)")
}
