// Package backup keeps short-lived snapshots of files NodePulse is about to
// overwrite. It is deliberately not a version-control system: one directory,
// timestamped copies and a small retention window.
package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DefaultKeep is how many snapshots are kept per source file.
const DefaultKeep = 20

// DefaultDir returns ~/.local/share/nodepulse/backups.
func DefaultDir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "nodepulse-backups"
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "nodepulse", "backups")
}

// Snapshot copies an existing file into dir and returns the backup path. A
// missing source is not an error: there is nothing to protect yet.
func Snapshot(path string, keep int) (string, error) {
	return SnapshotTo(path, DefaultDir(), keep)
}

// SnapshotTo is Snapshot with an explicit directory, used by tests and callers
// that keep backups next to a portable config.
func SnapshotTo(path, dir string, keep int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read %s for backup: %w", path, err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}
	_ = os.Chmod(dir, 0o700)
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	ext := filepath.Ext(base)
	stamp := time.Now().Format("20060102-150405.000000000")
	out := filepath.Join(dir, fmt.Sprintf("%s-%s%s", name, stamp, ext))
	if err := os.WriteFile(out, data, 0o600); err != nil {
		return "", fmt.Errorf("write backup %s: %w", out, err)
	}
	_ = os.Chmod(out, 0o600)
	if keep <= 0 {
		keep = DefaultKeep
	}
	if err := rotate(dir, name+"-", ext, keep); err != nil {
		return out, err
	}
	return out, nil
}

// Entry is one backup file.
type Entry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// List returns backups newest first.
func List(dir string) ([]Entry, error) {
	if dir == "" {
		dir = DefaultDir()
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		out = append(out, Entry{
			Name:    entry.Name(),
			Path:    filepath.Join(dir, entry.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModTime.After(out[j].ModTime) })
	return out, nil
}

func rotate(dir, prefix, ext string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	type file struct {
		name string
		time time.Time
	}
	matches := make([]file, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) || filepath.Ext(entry.Name()) != ext {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		matches = append(matches, file{name: entry.Name(), time: info.ModTime()})
	}
	if len(matches) <= keep {
		return nil
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].time.After(matches[j].time) })
	for _, old := range matches[keep:] {
		_ = os.Remove(filepath.Join(dir, old.name))
	}
	return nil
}
