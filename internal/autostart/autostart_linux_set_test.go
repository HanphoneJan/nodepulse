//go:build linux

package autostart

import (
	"os"
	"strings"
	"testing"
)

func TestSetEnableDisable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if Enabled() {
		t.Fatal("expected disabled by default")
	}
	if err := Set(true); err != nil {
		t.Fatal(err)
	}
	if !Enabled() {
		t.Fatal("expected enabled after Set(true)")
	}
	data, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "--background") {
		t.Fatalf("entry = %s", data)
	}
	if err := Set(false); err != nil {
		t.Fatal(err)
	}
	if Enabled() {
		t.Fatal("expected disabled after Set(false)")
	}
}
