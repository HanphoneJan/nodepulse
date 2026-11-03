package health

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/HanphoneJan/nodepulse/internal/settings"
)

func TestCheckControllerAndSubscription(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "sub.yaml")
	if err := os.WriteFile(cfgPath, []byte("proxies:\n  - {name: A, type: ss, server: 127.0.0.1, port: 1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"proxies":{"Proxy":{"type":"Selector","now":"A","all":["A"]},"A":{"type":"ss"}}}`)
	}))
	defer srv.Close()

	s := settings.Defaults()
	s.API = srv.URL
	s.ConfigPath = cfgPath
	s.Group = "Proxy"
	report := Run(context.Background(), s)
	if !report.OK {
		t.Fatalf("expected healthy report: %+v", report)
	}
	status := map[string]string{}
	for _, check := range report.Checks {
		status[check.ID] = check.Status
	}
	if status["controller"] != StatusOK || status["group"] != StatusOK || status["subscription"] != StatusOK {
		t.Fatalf("statuses = %v", status)
	}
}

func TestCheckReportsUnreachableController(t *testing.T) {
	s := settings.Defaults()
	s.API = "http://127.0.0.1:1"
	s.MihomoPath = "/bin/true"
	report := Run(context.Background(), s)
	if report.OK {
		t.Fatalf("expected unhealthy report: %+v", report)
	}
}
