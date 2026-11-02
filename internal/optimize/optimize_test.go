package optimize

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSwitchesToBest(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "sub.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
proxies:
  - {name: A, type: ss, server: 127.0.0.1, port: 1}
  - {name: B, type: ss, server: 127.0.0.1, port: 2}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	now := "B"
	var switched string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/proxies":
			fmt.Fprintf(w, `{"proxies":{"Proxy":{"type":"Selector","now":%q,"all":["A","B"]},"A":{"type":"ss"},"B":{"type":"ss"}}}`, now)
		case strings.HasPrefix(path, "/proxies/") && strings.HasSuffix(path, "/delay"):
			if strings.Contains(path, "/A/") {
				fmt.Fprint(w, `{"delay":100}`)
			} else {
				fmt.Fprint(w, `{"delay":800}`)
			}
		case strings.HasPrefix(path, "/proxies/") && r.Method == http.MethodPut:
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			switched = body.Name
			now = body.Name
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	res, err := Run(context.Background(), Request{
		API: srv.URL, ConfigPath: cfgPath, Rounds: 1, Concurrency: 2, Switch: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Best.Name != "A" {
		t.Fatalf("best = %q, want A", res.Best.Name)
	}
	if !res.Switched || switched != "A" || now != "A" {
		t.Fatalf("switched=%v api=%q now=%q", res.Switched, switched, now)
	}
}

func TestRunRecommendOnly(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "sub.yaml")
	if err := os.WriteFile(cfgPath, []byte(`proxies:
  - {name: A, type: ss, server: 127.0.0.1, port: 1}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/proxies":
			fmt.Fprint(w, `{"proxies":{"Proxy":{"type":"Selector","now":"A","all":["A"]},"A":{"type":"ss"}}}`)
		case strings.HasPrefix(r.URL.Path, "/proxies/") && strings.HasSuffix(r.URL.Path, "/delay"):
			fmt.Fprint(w, `{"delay":90}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	res, err := Run(context.Background(), Request{
		API: srv.URL, ConfigPath: cfgPath, Rounds: 1, Concurrency: 1, Switch: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Switched {
		t.Fatal("should not switch when Switch=false")
	}
	if res.Best.Name != "A" {
		t.Fatalf("best = %q", res.Best.Name)
	}
}
