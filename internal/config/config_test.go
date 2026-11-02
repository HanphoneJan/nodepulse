package config

import (
	"strings"
	"testing"
)

const sample = `
mixed-port: 7890
proxies:
  - name: "Node A"
    type: vless
    server: example.com
    port: 443
    uuid: abc
    reality-opts:
      public-key: key
  - name: "Node B"
    type: direct
  - name: "剩余流量：1 GB"
    type: ss
    server: 127.0.0.1
    port: 8388
proxy-groups:
  - name: Auto
    type: url-test
    proxies: ["Node A"]
rules:
  - MATCH,Auto
`

func TestParsePreservesUnknownProxyFields(t *testing.T) {
	cfg, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Proxies) != 3 {
		t.Fatalf("proxies = %d, want 3", len(cfg.Proxies))
	}
	p := cfg.Proxies[0]
	if p.Name != "Node A" || p.Type != "vless" || p.Server != "example.com" || p.Port != 443 {
		t.Fatalf("unexpected proxy: %+v", p)
	}
	if p.Extra["uuid"] != "abc" {
		t.Fatalf("uuid not preserved: %#v", p.Extra)
	}
	if _, ok := p.Extra["reality-opts"]; !ok {
		t.Fatalf("reality-opts not preserved: %#v", p.Extra)
	}
}

func TestMarshalKeepsExtraFields(t *testing.T) {
	cfg, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	data, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "public-key") {
		t.Fatalf("marshaled config lost reality-opts:\n%s", data)
	}
}

func TestRealProxiesFiltersBuiltinsAndEmpty(t *testing.T) {
	cfg := &Config{Proxies: []Proxy{
		{Name: "ok", Type: "ss", Server: "127.0.0.1", Port: 1},
		{Name: "direct", Type: "direct"},
		{Name: "no-server", Type: "ss"},
		{Name: "", Type: "ss", Server: "127.0.0.1"},
	}}
	got := cfg.RealProxies()
	if len(got) != 1 || got[0].Name != "ok" {
		t.Fatalf("RealProxies = %+v, want only ok", got)
	}
}

func TestIsInfoNode(t *testing.T) {
	cases := map[string]bool{
		"🇭🇰香港01":            false,
		"剩余流量：773 GB":       true,
		"套餐到期：长期有效":         true,
		"官网公告":              true,
		"Remaining Traffic": true,
	}
	for name, want := range cases {
		if got := IsInfoNode(name); got != want {
			t.Errorf("IsInfoNode(%q) = %v, want %v", name, got, want)
		}
	}
}
