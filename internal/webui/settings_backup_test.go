package webui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/HanphoneJan/nodepulse/internal/settings"
)

func TestSettingsBackupAndRestore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))

	s := New(Options{
		ResultsPath: filepath.Join(dir, "results.json"),
		HistoryPath: filepath.Join(dir, "history.jsonl"),
	})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	payload := settings.Defaults()
	payload.API = "http://127.0.0.1:9999"
	payload.Secret = "restored-secret"
	payload.ConfigPath = filepath.Join(dir, "sub.yaml")
	payload.Rounds = 9
	payload.CloseToTray = false
	payload.NormalizeProfiles()

	restore := func(value settings.Settings) {
		body, _ := json.Marshal(value)
		resp, err := http.Post(srv.URL+"/api/settings/restore", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("restore status = %d", resp.StatusCode)
		}
	}
	restore(payload)
	payload.Rounds = 11
	restore(payload)

	loaded, err := settings.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.API != payload.API || loaded.Rounds != 11 || loaded.CloseToTray {
		t.Fatalf("restored settings = %+v", loaded)
	}

	entries, err := os.ReadDir(filepath.Join(dir, "data", "nodepulse", "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected a settings backup before the second restore")
	}

	resp, err := http.Get(srv.URL + "/api/settings/backup")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("backup status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Disposition"); got == "" {
		t.Fatal("backup response is not an attachment")
	}
	var downloaded settings.Settings
	if err := json.NewDecoder(resp.Body).Decode(&downloaded); err != nil {
		t.Fatal(err)
	}
	if downloaded.API != payload.API || downloaded.Rounds != 11 {
		t.Fatalf("download = %+v", downloaded)
	}
}
