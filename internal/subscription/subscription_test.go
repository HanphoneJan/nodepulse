package subscription

import (
	"encoding/base64"
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

func TestLoadReaderBase64ShareLinks(t *testing.T) {
	body := base64.StdEncoding.EncodeToString([]byte("hysteria2://pass@example.com:443#Node H\n"))
	cfg, err := LoadReader(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Proxies) != 1 || cfg.Proxies[0].Name != "Node H" || cfg.Proxies[0].Type != "hysteria2" {
		t.Fatalf("unexpected proxies: %+v", cfg.Proxies)
	}
}
