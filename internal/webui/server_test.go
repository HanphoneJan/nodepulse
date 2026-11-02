package webui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDashboardAndAPI(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "sub.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
proxies:
  - {name: A, type: ss, server: 127.0.0.1, port: 1}
  - {name: B, type: ss, server: 127.0.0.1, port: 2}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	resultsPath := filepath.Join(dir, "results.json")
	historyPath := filepath.Join(dir, "history.jsonl")

	var switched string
	clashSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/proxies":
			fmt.Fprint(w, `{"proxies":{"Proxy":{"type":"Selector","now":"B","all":["A","B"]},"A":{"type":"ss"},"B":{"type":"ss"}}}`)
		case strings.HasPrefix(path, "/proxies/") && strings.HasSuffix(path, "/delay"):
			fmt.Fprint(w, `{"delay":120}`)
		case strings.HasPrefix(path, "/proxies/") && r.Method == http.MethodPut:
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			switched = body.Name
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer clashSrv.Close()

	s := New(Options{
		API:         clashSrv.URL,
		ResultsPath: resultsPath,
		HistoryPath: historyPath,
	})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	// Dashboard.
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("dashboard status = %d", resp.StatusCode)
	}

	// Test.
	payload := fmt.Sprintf(`{"config_path":%q,"rounds":1,"concurrency":2}`, cfgPath)
	resp, err = http.Post(srv.URL+"/api/test", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("test status = %d", resp.StatusCode)
	}
	var results []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}

	// Results file.
	resp, err = http.Get(srv.URL + "/api/results")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("results status = %d", resp.StatusCode)
	}

	// Status.
	resp, err = http.Get(srv.URL + "/api/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	// Switch.
	resp, err = http.Post(srv.URL+"/api/switch", "application/json", bytes.NewBufferString(`{"group":"Proxy","node":"A"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || switched != "A" {
		t.Fatalf("switch status = %d switched = %q", resp.StatusCode, switched)
	}
}
