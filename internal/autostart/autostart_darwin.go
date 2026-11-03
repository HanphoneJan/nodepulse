//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Path returns the LaunchAgent plist for NodePulse.
func Path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "com.nodepulse.app.plist"
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.nodepulse.app.plist")
}

// Enabled reports whether the login item exists.
func Enabled() bool {
	_, err := os.Stat(Path())
	return err == nil
}

// Set creates or removes the login item. macOS applies it on next login.
func Set(enabled bool) error {
	path := Path()
	if !enabled {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	exe, err := executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.nodepulse.app</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>--background</string>
  </array>
  <key>RunAtLoad</key><true/>
</dict>
</plist>
`, strings.ReplaceAll(exe, "&", "&amp;"))
	return os.WriteFile(path, []byte(content), 0o600)
}
