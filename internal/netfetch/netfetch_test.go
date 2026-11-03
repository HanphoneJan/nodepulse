package netfetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownloadAllowsLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxies: []\n"))
	}))
	defer srv.Close()
	data, err := Download(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "proxies: []\n" {
		t.Fatalf("data = %q", data)
	}
}

func TestValidateURLRejectsUnsafeTargets(t *testing.T) {
	for _, raw := range []string{
		"file:///etc/passwd",
		"http://user:pass@example.com/sub",
		"http://169.254.169.254/latest/meta-data",
		"http://[fe80::1]/",
		"http://100.100.100.200/",
	} {
		if _, err := ValidateURL(raw); err == nil {
			t.Errorf("expected %s to be rejected", raw)
		}
	}
	if _, err := ValidateURL("http://10.0.0.2:8080/sub"); err != nil {
		t.Errorf("LAN URL should be allowed: %v", err)
	}
	if _, err := ValidateURL("http://127.0.0.1:8080/sub"); err != nil {
		t.Errorf("loopback URL should be allowed: %v", err)
	}
}

func TestDownloadRejectsRedirectToMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data", http.StatusFound)
	}))
	defer srv.Close()
	_, err := Download(context.Background(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "链路本地") {
		t.Fatalf("expected metadata redirect rejection, got %v", err)
	}
}

func TestDownloadRejectsOversize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", MaxBytes+1)))
	}))
	defer srv.Close()
	_, err := Download(context.Background(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "超过") {
		t.Fatalf("expected size error, got %v", err)
	}
}
