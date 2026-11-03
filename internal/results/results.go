// Package results stores the last ranked test results in a stable schema
// shared by the GUI and the local web UI.
package results

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HanphoneJan/nodepulse/internal/score"
)

// File is the on-disk results schema.
type File struct {
	GeneratedAt string         `json:"generated_at"`
	URL         string         `json:"url"`
	Rounds      int            `json:"rounds"`
	ConfigPath  string         `json:"config_path,omitempty"`
	Results     []score.Result `json:"results"`
}

// DefaultPath returns ~/.local/share/nodepulse/results.json.
func DefaultPath() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "nodepulse-results.json"
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "nodepulse", "results.json")
}

// Load reads a results file.
func Load(path string) (*File, error) {
	if path == "" {
		path = DefaultPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse results: %w", err)
	}
	if len(file.Results) == 0 {
		return nil, fmt.Errorf("results file is empty")
	}
	return &file, nil
}

// Save writes atomically so a crash cannot leave a truncated JSON file.
func Save(path string, file File) error {
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create results dir: %w", err)
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".results-*.json")
	if err != nil {
		return fmt.Errorf("create temporary results: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temporary results: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary results: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temporary results: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary results: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace results: %w", err)
	}
	_ = os.Chmod(path, 0o600)
	return nil
}

// Remove deletes the results file; a missing file is not an error.
func Remove(path string) error {
	if path == "" {
		path = DefaultPath()
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
