package clash

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientVersionProxiesAndDelay(t *testing.T) {
	var gotAuth, gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/version":
			fmt.Fprint(w, `{"version":"v1.19.0"}`)
		case "/proxies":
			fmt.Fprint(w, `{"proxies":{"Node A":{"type":"ss","now":"","all":[]}}}`)
		case "/proxies/Node A/delay":
			gotURL = r.URL.Query().Get("url")
			fmt.Fprint(w, `{"delay":123}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "secret")
	v, err := c.Version(context.Background())
	if err != nil || v != "v1.19.0" {
		t.Fatalf("Version = %q, %v", v, err)
	}
	ps, err := c.Proxies(context.Background())
	if err != nil || ps["Node A"].Type != "ss" {
		t.Fatalf("Proxies = %+v, %v", ps, err)
	}
	ms, err := c.Delay(context.Background(), "Node A", "https://example.com", time.Second)
	if err != nil || ms != 123 {
		t.Fatalf("Delay = %d, %v", ms, err)
	}
	if gotURL != "https://example.com" {
		t.Fatalf("test url = %q", gotURL)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("auth = %q", gotAuth)
	}
}

func TestDelayHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	if _, err := c.Delay(context.Background(), "x", "https://example.com", time.Second); err == nil {
		t.Fatal("expected an error")
	}
}
