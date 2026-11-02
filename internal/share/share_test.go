package share

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/HanphoneJan/nodepulse/internal/config"
)

func mk(typ, name, server string, port int, extra map[string]any) config.Proxy {
	return config.Proxy{Name: name, Type: typ, Server: server, Port: port, Extra: extra}
}

func TestParseVLESSReality(t *testing.T) {
	line := "vless://11111111-1111-1111-1111-111111111111@example.com:443?encryption=none&security=reality&sni=www.apple.com&fp=chrome&pbk=pubkey&sid=abcd&flow=xtls-rprx-vision&type=ws&path=%2Fws&host=cdn.example.com#Node%20A"
	p, err := ParseLink(line)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Node A" || p.Type != "vless" || p.Server != "example.com" || p.Port != 443 {
		t.Fatalf("unexpected proxy: %+v", p)
	}
	if p.Extra["uuid"] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("uuid = %v", p.Extra["uuid"])
	}
	if _, ok := p.Extra["reality-opts"]; !ok {
		t.Fatalf("reality-opts missing: %#v", p.Extra)
	}
	if p.Extra["network"] != "ws" {
		t.Fatalf("network = %v", p.Extra["network"])
	}
}

func TestParseTrojanSSHysteria2(t *testing.T) {
	cases := []struct {
		line string
		typ  string
		name string
	}{
		{"trojan://pass@example.com:443?security=tls&sni=example.com#T1", "trojan", "T1"},
		{"hysteria2://pass@example.com:8443?sni=example.com&insecure=1#H1", "hysteria2", "H1"},
		{"hy2://pass@example.com:8443#H2", "hysteria2", "H2"},
	}
	for _, tc := range cases {
		p, err := ParseLink(tc.line)
		if err != nil {
			t.Fatalf("%s: %v", tc.line, err)
		}
		if p.Type != tc.typ || p.Name != tc.name {
			t.Fatalf("%s -> %+v", tc.line, p)
		}
		if p.Extra["password"] != "pass" {
			t.Fatalf("%s credentials = %v", tc.line, p.Extra["password"])
		}
	}
}

func TestParseSS(t *testing.T) {
	cred := base64.StdEncoding.EncodeToString([]byte("aes-128-gcm:secret"))
	p, err := ParseLink("ss://" + cred + "@example.com:8388#SS%20A")
	if err != nil {
		t.Fatal(err)
	}
	if p.Type != "ss" || p.Name != "SS A" || p.Server != "example.com" || p.Port != 8388 {
		t.Fatalf("unexpected: %+v", p)
	}
	if p.Extra["cipher"] != "aes-128-gcm" || p.Extra["password"] != "secret" {
		t.Fatalf("creds = %#v", p.Extra)
	}
}

func TestParseVMess(t *testing.T) {
	payload := map[string]any{
		"v": "2", "ps": "VM", "add": "example.com", "port": "443", "id": "uuid",
		"aid": "0", "net": "ws", "path": "/ws", "host": "cdn.example.com", "tls": "tls", "sni": "example.com",
	}
	data, _ := json.Marshal(payload)
	line := "vmess://" + base64.StdEncoding.EncodeToString(data)
	p, err := ParseLink(line)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "VM" || p.Type != "vmess" || p.Server != "example.com" || p.Port != 443 {
		t.Fatalf("unexpected: %+v", p)
	}
	if p.Extra["uuid"] != "uuid" {
		t.Fatalf("uuid = %v", p.Extra["uuid"])
	}
	if p.Extra["network"] != "ws" {
		t.Fatalf("network = %v", p.Extra["network"])
	}
}

func TestRoundTrip(t *testing.T) {
	items := []config.Proxy{
		mk("vless", "V", "example.com", 443, map[string]any{
			"uuid": "u", "tls": true, "servername": "sni", "network": "ws",
			"ws-opts": map[string]any{"path": "/ws", "headers": map[string]any{"Host": "h"}},
		}),
		mk("trojan", "T", "example.com", 443, map[string]any{"password": "p", "sni": "sni"}),
		mk("hysteria2", "H", "example.com", 8443, map[string]any{"password": "p", "sni": "sni"}),
		mk("ss", "S", "example.com", 8388, map[string]any{"cipher": "aes-128-gcm", "password": "p"}),
		mk("vmess", "M", "example.com", 443, map[string]any{"uuid": "u", "network": "ws"}),
	}
	for _, p := range items {
		link, err := EncodeProxy(p)
		if err != nil {
			t.Fatalf("%s encode: %v", p.Type, err)
		}
		got, err := ParseLink(link)
		if err != nil {
			t.Fatalf("%s parse %q: %v", p.Type, link, err)
		}
		if got.Type != p.Type || got.Server != p.Server || got.Port != p.Port || got.Name != p.Name {
			t.Fatalf("%s round trip: got %+v", p.Type, got)
		}
	}
}

func TestParseBase64Subscription(t *testing.T) {
	cred := base64.StdEncoding.EncodeToString([]byte("aes-128-gcm:secret"))
	lines := "ss://" + cred + "@example.com:8388#A\nhysteria2://p@example.com:443#B"
	body := []byte(base64.StdEncoding.EncodeToString([]byte(lines)))
	proxies, err := ParseSubscription(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(proxies) != 2 {
		t.Fatalf("proxies = %d, want 2", len(proxies))
	}
}

func TestEncodeSubscription(t *testing.T) {
	body, err := EncodeSubscription([]config.Proxy{
		mk("ss", "A", "example.com", 8388, map[string]any{"cipher": "aes-128-gcm", "password": "p"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	proxies, err := ParseSubscription(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(proxies) != 1 || proxies[0].Name != "A" {
		t.Fatalf("proxies = %+v", proxies)
	}
}
