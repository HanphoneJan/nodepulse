package subscription

import (
	"strings"
	"testing"
)

func TestLoadReader(t *testing.T) {
	cfg, err := LoadReader(strings.NewReader(`
proxies:
  - {name: A, type: ss, server: 127.0.0.1, port: 1}
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Proxies) != 1 || cfg.Proxies[0].Name != "A" {
		t.Fatalf("unexpected config: %+v", cfg.Proxies)
	}
}

func TestLoadReaderRejectsEmpty(t *testing.T) {
	if _, err := LoadReader(strings.NewReader("proxies: []\n")); err == nil {
		t.Fatal("expected an error for an empty subscription")
	}
}
