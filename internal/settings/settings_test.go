package settings

import (
	"path/filepath"
	"testing"
)

func TestSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := Defaults()
	s.API = "http://127.0.0.1:9090"
	s.Rounds = 10
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.API != s.API || got.Rounds != 10 {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Rounds != 5 || got.Concurrency != 8 {
		t.Fatalf("defaults = %+v", got)
	}
}
