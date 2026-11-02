package subscription

import (
	"fmt"
	"io"
	"os"

	"github.com/HanphoneJan/nodepulse/internal/config"
)

// Load reads a Clash / mihomo YAML subscription from a local file.
func Load(path string) (*config.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open subscription: %w", err)
	}
	defer f.Close()
	return LoadReader(f)
}

// LoadReader parses a Clash / mihomo subscription from a reader.
func LoadReader(r io.Reader) (*config.Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read subscription: %w", err)
	}
	cfg, err := config.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse subscription: %w", err)
	}
	if len(cfg.Proxies) == 0 {
		return nil, fmt.Errorf("subscription has no proxies (only Clash / mihomo YAML is supported in this phase)")
	}
	return cfg, nil
}
