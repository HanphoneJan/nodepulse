package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Settings is the GUI / web UI configuration.
type Settings struct {
	API         string `json:"api"`
	Secret      string `json:"secret"`
	Group       string `json:"group"`
	ConfigPath  string `json:"config_path"`
	MihomoPath  string `json:"mihomo_path"`
	URL         string `json:"url"`
	Rounds      int    `json:"rounds"`
	Concurrency int    `json:"concurrency"`
	TimeoutMs   int    `json:"timeout_ms"`
}

// DefaultPath is ~/.config/nodepulse/settings.json.
func DefaultPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "nodepulse-settings.json"
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "nodepulse", "settings.json")
}

// Defaults returns sensible starting values.
func Defaults() Settings {
	return Settings{
		URL:         "https://www.gstatic.com/generate_204",
		Rounds:      5,
		Concurrency: 8,
		TimeoutMs:   5000,
	}
}

// Load reads settings; a missing file returns defaults.
func Load(path string) (Settings, error) {
	if path == "" {
		path = DefaultPath()
	}
	s := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, fmt.Errorf("read settings: %w", err)
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("parse settings: %w", err)
	}
	if s.Rounds <= 0 {
		s.Rounds = 5
	}
	if s.Concurrency <= 0 {
		s.Concurrency = 8
	}
	if s.TimeoutMs <= 0 {
		s.TimeoutMs = 5000
	}
	if s.URL == "" {
		s.URL = "https://www.gstatic.com/generate_204"
	}
	return s, nil
}

// Save writes settings with 0600 permissions.
func Save(path string, s Settings) error {
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}
