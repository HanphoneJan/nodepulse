package results

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HanphoneJan/nodepulse/internal/score"
)

func TestSaveLoadAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "results.json")
	file := File{GeneratedAt: "2026-11-03T00:00:00Z", URL: "https://example.com", Rounds: 5, ConfigPath: "/tmp/sub.yaml", Results: []score.Result{{Name: "A", Success: 5, Total: 5}}}
	if err := Save(path, file); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.ConfigPath != file.ConfigPath || len(got.Results) != 1 || got.Results[0].Name != "A" {
		t.Fatalf("loaded = %+v", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	if err := Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); !os.IsNotExist(err) {
		t.Fatalf("expected missing file, got %v", err)
	}
}
