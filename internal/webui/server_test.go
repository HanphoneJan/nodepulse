package webui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HanphoneJan/nodepulse/internal/score"
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

func TestSecurityMiddlewareAndMethodChecks(t *testing.T) {
	s := New(Options{})
	handler := s.Handler()

	// Cross-origin state-changing request is rejected.
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/settings", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin status = %d, want 403", rec.Code)
	}

	// Cross-site fetch metadata is rejected.
	req = httptest.NewRequest(http.MethodPost, "http://example.com/api/settings", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site status = %d, want 403", rec.Code)
	}

	// Same-origin requests reach the handler (empty body is a 400, not 403).
	req = httptest.NewRequest(http.MethodPost, "http://example.com/api/settings", nil)
	req.Header.Set("Origin", "http://example.com")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Fatalf("same-origin request was rejected")
	}

	// Daemon stop must be POST.
	req = httptest.NewRequest(http.MethodGet, "http://example.com/api/daemon/stop", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET daemon stop = %d, want 405", rec.Code)
	}

	// Subscription import only accepts http(s).
	req = httptest.NewRequest(http.MethodPost, "http://example.com/api/import", bytes.NewBufferString(`{"url":"file:///etc/passwd"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("file import = %d, want 400", rec.Code)
	}
}

func TestExportDirUsesXDGDataHome(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_DATA_HOME", base)
	want := filepath.Join(base, "nodepulse", "exports")
	if got := exportDir(); got != want {
		t.Fatalf("exportDir() = %q, want %q", got, want)
	}
}

func TestRunTestWithControllerOnly(t *testing.T) {
	clashSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/proxies":
			fmt.Fprint(w, `{"proxies":{"Proxy":{"type":"Selector","now":"A","all":["A","B"]},"A":{"type":"ss"},"B":{"type":"ss"}}}`)
		case strings.HasPrefix(r.URL.Path, "/proxies/") && strings.HasSuffix(r.URL.Path, "/delay"):
			fmt.Fprint(w, `{"delay":120}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer clashSrv.Close()

	s := New(Options{
		API:         clashSrv.URL,
		ResultsPath: filepath.Join(t.TempDir(), "results.json"),
		HistoryPath: filepath.Join(t.TempDir(), "history.jsonl"),
	})
	results, err := s.runTest(context.Background(), testRequest{API: clashSrv.URL, Rounds: 1, Concurrency: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("controller-only results = %d, want 2", len(results))
	}
}

func TestExportAfterRestartUsesSavedResults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
	cfgPath := filepath.Join(dir, "sub.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
mixed-port: 7897
proxies:
  - {name: A, type: ss, server: 127.0.0.1, port: 1}
  - {name: B, type: ss, server: 127.0.0.1, port: 2}
proxy-groups:
  - {name: PROXY, type: select, proxies: [A, B]}
rules:
  - MATCH,PROXY
dns:
  enable: true
`), 0o600); err != nil {
		t.Fatal(err)
	}
	resultsPath := filepath.Join(dir, "results.json")
	rf := resultsFile{
		GeneratedAt: "2026-01-01T00:00:00Z",
		URL:         "https://example.com",
		Rounds:      1,
		ConfigPath:  cfgPath,
		Results: []score.Result{
			{Name: "A", Type: "ss", Success: 10, Total: 10, MedianMs: 100, P90Ms: 100, Score: 1},
		},
	}
	data, err := json.Marshal(rf)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultsPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(Options{ResultsPath: resultsPath, HistoryPath: filepath.Join(dir, "history.jsonl")})
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/export", bytes.NewBufferString(`{"out":"stable.yaml"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export after restart = %d: %s", rec.Code, rec.Body.String())
	}
	outPath := filepath.Join(exportDir(), "stable.yaml")
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("export file missing: %v", err)
	}
	out, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "MATCH,PROXY") || !strings.Contains(text, "dns:") || !strings.Contains(text, "NodePulse") {
		t.Fatalf("preserved export missing expected keys:\n%s", text)
	}
}

func TestClearResultsRemovesSavedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "results.json")
	if err := os.WriteFile(path, []byte(`{"results":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Options{ResultsPath: path, HistoryPath: filepath.Join(dir, "history.jsonl")})
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/results/clear", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("clear results = %d", rec.Code)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("results file still exists: %v", err)
	}
}
