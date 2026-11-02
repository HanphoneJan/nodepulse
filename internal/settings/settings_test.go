package settings

import (
	"os"
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
	if len(got.Profiles) != 1 || got.Profiles[0].API != s.API {
		t.Fatalf("profiles = %+v", got.Profiles)
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
	if len(got.Profiles) != 1 || got.ActiveProfile == "" {
		t.Fatalf("default profile missing: %+v", got)
	}
}

func TestLegacySettingsGetDefaultProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	legacy := `{"api":"http://127.0.0.1:9090","secret":"s","group":"Proxy","config_path":"/tmp/a.yaml","rounds":3}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Profiles) != 1 || got.ActiveProfile != got.Profiles[0].ID {
		t.Fatalf("profiles = %+v active=%q", got.Profiles, got.ActiveProfile)
	}
	if got.Profiles[0].API != "http://127.0.0.1:9090" {
		t.Fatalf("profile api = %q", got.Profiles[0].API)
	}
}

func TestProfileSwitchPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := Defaults()
	s.Profiles = []Profile{
		{ID: "a", Name: "FlClash", API: "http://127.0.0.1:9090", ConfigPath: "/tmp/a.yaml", Rounds: 5},
		{ID: "b", Name: "Verge", API: "http://127.0.0.1:9097", ConfigPath: "/tmp/b.yaml", Rounds: 10},
	}
	s.ActiveProfile = "a"
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.ActivateProfile("b") {
		t.Fatal("profile b not found")
	}
	if got.API != "http://127.0.0.1:9097" || got.ConfigPath != "/tmp/b.yaml" || got.Rounds != 10 {
		t.Fatalf("active fields = %+v", got)
	}
	if err := Save(path, got); err != nil {
		t.Fatal(err)
	}
	again, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.ActiveProfile != "b" || again.API != "http://127.0.0.1:9097" {
		t.Fatalf("reloaded = %+v", again)
	}
}

func TestSaveWithoutProfilesPreservesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := Defaults()
	s.Profiles = []Profile{
		{ID: "a", Name: "A", API: "http://127.0.0.1:9090", ConfigPath: "/tmp/a.yaml"},
		{ID: "b", Name: "B", API: "http://127.0.0.1:9097", ConfigPath: "/tmp/b.yaml"},
	}
	s.ActiveProfile = "a"
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}

	oldClient := Defaults()
	oldClient.API = "http://127.0.0.1:9999"
	oldClient.Rounds = 5
	if err := Save(path, oldClient); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Profiles) != 2 {
		t.Fatalf("profiles were wiped: %+v", got.Profiles)
	}
	if got.Profiles[0].API != "http://127.0.0.1:9999" {
		t.Fatalf("active profile not synced: %+v", got.Profiles[0])
	}
}
