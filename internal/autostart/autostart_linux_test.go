//go:build linux

package autostart

import (
	"strings"
	"testing"
)

func TestDesktopEntryStartsInBackground(t *testing.T) {
	entry := desktopEntry(`/opt/Node Pulse/nodepulse-gui`)
	if !strings.Contains(entry, `Exec="/opt/Node Pulse/nodepulse-gui" --background`) {
		t.Fatalf("entry = %s", entry)
	}
	if !strings.Contains(entry, "X-GNOME-Autostart-enabled=true") {
		t.Fatalf("entry = %s", entry)
	}
}
