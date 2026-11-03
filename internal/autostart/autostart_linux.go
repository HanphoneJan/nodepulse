//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

// Path returns the XDG autostart file for NodePulse.
func Path() string {
	return filepath.Join(configHome(), "autostart", "nodepulse.desktop")
}

// Enabled reports whether the login item exists.
func Enabled() bool {
	_, err := os.Stat(Path())
	return err == nil
}

// Set creates or removes the login item.
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
	return os.WriteFile(path, []byte(desktopEntry(exe)), 0o600)
}

func desktopEntry(exe string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=NodePulse
Comment=Keep the selected proxy node stable
Exec=%s --background
Terminal=false
X-GNOME-Autostart-enabled=true
`, quoteDesktop(exe))
}
