package clash

import (
	"context"
	"encoding/json"
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

func TestSwitchAndCurrent(t *testing.T) {
	now := "B"
	var switched string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/proxies":
			fmt.Fprintf(w, `{"proxies":{"Auto":{"type":"Selector","now":%q,"all":["A","B"]}}}`, now)
		case r.URL.Path == "/proxies/Auto" && r.Method == http.MethodPut:
			var body struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			switched = body.Name
			now = body.Name
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "secret")
	cur, err := c.Current(context.Background(), "Auto")
	if err != nil || cur != "B" {
		t.Fatalf("Current = %q, %v", cur, err)
	}
	if err := c.Switch(context.Background(), "Auto", "A"); err != nil {
		t.Fatal(err)
	}
	if switched != "A" {
		t.Fatalf("switched = %q, want A", switched)
	}
	cur, err = c.Current(context.Background(), "Auto")
	if err != nil || cur != "A" {
		t.Fatalf("Current after switch = %q, %v", cur, err)
	}
}
