package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotAndRotate(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(src, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	backups := filepath.Join(dir, "backups")
	for i := 0; i < 4; i++ {
		if _, err := SnapshotTo(src, backups, 2); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := List(backups)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("kept %d backups, want 2", len(entries))
	}
	data, err := os.ReadFile(entries[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"a":1}` {
		t.Fatalf("backup content = %q", data)
	}
}

func TestSnapshotMissingFileIsNoop(t *testing.T) {
	path, err := SnapshotTo(filepath.Join(t.TempDir(), "missing.json"), t.TempDir(), 2)
	if err != nil || path != "" {
		t.Fatalf("path=%q err=%v", path, err)
	}
}
