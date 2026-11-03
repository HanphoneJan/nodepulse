package localapi

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

func TestDaemonLockRejectsSecondServer(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "daemon.lock")
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/proxies":
			fmt.Fprint(w, `{"proxies":{"Proxy":{"type":"Selector","now":"A","all":["A","B"]},"A":{"type":"ss"},"B":{"type":"ss"}}}`)
		case strings.HasSuffix(r.URL.Path, "/delay"):
			fmt.Fprint(w, `{"delay":120}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer fake.Close()

	newServer := func(name string) *httptest.Server {
		s := New(Options{
			ResultsPath: filepath.Join(dir, name+"-results.json"),
			HistoryPath: filepath.Join(dir, name+"-history.jsonl"),
			LockPath:    lockPath,
		})
		return httptest.NewServer(s.Handler())
	}
	first := newServer("first")
	defer first.Close()
	second := newServer("second")
	defer second.Close()

	body, _ := json.Marshal(daemonRequest{API: fake.URL, Group: "Proxy", IntervalSec: 3600, Rounds: 1})
	post := func(base string) int {
		resp, err := http.Post(base+"/api/daemon/start", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}
	if code := post(first.URL); code != http.StatusOK {
		t.Fatalf("first start = %d", code)
	}
	if code := post(second.URL); code != http.StatusConflict {
		t.Fatalf("second start = %d, want 409", code)
	}
	resp, err := http.Post(first.URL+"/api/daemon/stop", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("lock should be released, got %v", err)
	}
}
