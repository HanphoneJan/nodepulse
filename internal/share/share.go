package share

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/HanphoneJan/nodepulse/internal/config"
)

// ParseSubscription parses either a Clash-style YAML blob (handled elsewhere)
// or a base64 / newline separated list of v2ray share links.
func ParseSubscription(data []byte) ([]config.Proxy, error) {
	text := strings.TrimSpace(string(data))
	if text == "" {
		return nil, fmt.Errorf("empty subscription")
	}
	if strings.Contains(text, "://") {
		return ParseLinks(text)
	}
	decoded, err := decodeBase64(text)
	if err != nil {
		return nil, fmt.Errorf("not a share-link subscription: %w", err)
	}
	return ParseLinks(string(decoded))
}

// ParseLinks parses newline-separated share links.
func ParseLinks(text string) ([]config.Proxy, error) {
	var out []config.Proxy
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, err := ParseLink(line)
		if err != nil {
			continue // skip unsupported links rather than failing the whole subscription
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no supported share links found")
	}
	return out, nil
}

// ParseLink parses one share link.
func ParseLink(line string) (config.Proxy, error) {
	switch {
	case strings.HasPrefix(line, "vmess://"):
		return parseVMess(line)
	case strings.HasPrefix(line, "vless://"):
		return parseURLProxy(line, "vless")
	case strings.HasPrefix(line, "trojan://"):
		return parseURLProxy(line, "trojan")
	case strings.HasPrefix(line, "hysteria2://"):
		return parseURLProxy(line, "hysteria2")
	case strings.HasPrefix(line, "hy2://"):
		return parseURLProxy(strings.Replace(line, "hy2://", "hysteria2://", 1), "hysteria2")
	case strings.HasPrefix(line, "ss://"):
		return parseSS(line)
	default:
		return config.Proxy{}, fmt.Errorf("unsupported scheme: %s", line)
	}
}

func parseVMess(line string) (config.Proxy, error) {
	raw := strings.TrimPrefix(line, "vmess://")
	decoded, err := decodeBase64(raw)
	if err != nil {
		return config.Proxy{}, fmt.Errorf("vmess: decode: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(decoded, &m); err != nil {
		return config.Proxy{}, fmt.Errorf("vmess: json: %w", err)
	}
	name := str(m, "ps")
	server := str(m, "add")
	port := integer(m, "port")
	if server == "" || port == 0 {
		return config.Proxy{}, fmt.Errorf("vmess: missing server/port")
	}
	p := config.Proxy{Name: name, Type: "vmess", Server: server, Port: port, Extra: map[string]any{}}
	p.Extra["uuid"] = str(m, "id")
	if aid := integer(m, "aid"); aid > 0 {
		p.Extra["alterId"] = aid
	}
	if scy := str(m, "scy"); scy != "" {
		p.Extra["cipher"] = scy
	} else {
		p.Extra["cipher"] = "auto"
	}
	network := str(m, "net")
	if network == "" {
		network = "tcp"
	}
	p.Extra["network"] = network
	if strings.EqualFold(str(m, "tls"), "tls") {
		p.Extra["tls"] = true
		if sni := str(m, "sni"); sni != "" {
			p.Extra["servername"] = sni
		}
	}
	if network == "ws" {
		p.Extra["ws-opts"] = map[string]any{
			"path":    str(m, "path"),
			"headers": map[string]any{"Host": str(m, "host")},
		}
	}
	if name == "" {
		p.Name = fmt.Sprintf("%s:%d", server, port)
	}
	return p, nil
}

func parseURLProxy(line, typ string) (config.Proxy, error) {
	u, err := url.Parse(line)
	if err != nil {
		return config.Proxy{}, err
	}
	if u.User == nil {
		return config.Proxy{}, fmt.Errorf("%s: missing credentials", typ)
	}
	host, port := splitHostPort(u.Host)
	if host == "" || port == 0 {
		return config.Proxy{}, fmt.Errorf("%s: missing host/port", typ)
	}
	password := u.User.Username()
	q := u.Query()
	p := config.Proxy{Name: fragmentName(u), Type: typ, Server: host, Port: port, Extra: map[string]any{}}
	if p.Name == "" {
		p.Name = fmt.Sprintf("%s:%d", host, port)
	}

	switch typ {
	case "vless":
		p.Extra["uuid"] = password
		p.Extra["encryption"] = defaultString(q.Get("encryption"), "none")
		security := q.Get("security")
		if security == "tls" || security == "reality" {
			p.Extra["tls"] = true
			if sni := q.Get("sni"); sni != "" {
				p.Extra["servername"] = sni
			}
			if fp := q.Get("fp"); fp != "" {
				p.Extra["client-fingerprint"] = fp
			}
			if security == "reality" {
				p.Extra["reality-opts"] = map[string]any{
					"public-key": q.Get("pbk"),
					"short-id":   q.Get("sid"),
				}
			}
		}
		if flow := q.Get("flow"); flow != "" {
			p.Extra["flow"] = flow
		}
		applyTransport(p.Extra, q)
	case "trojan":
		p.Extra["password"] = password
		if sni := q.Get("sni"); sni != "" {
			p.Extra["sni"] = sni
		}
		if q.Get("allowInsecure") == "1" || strings.EqualFold(q.Get("skip-cert-verify"), "true") {
			p.Extra["skip-cert-verify"] = true
		}
		applyTransport(p.Extra, q)
	case "hysteria2":
		p.Extra["password"] = password
		if sni := q.Get("sni"); sni != "" {
			p.Extra["sni"] = sni
		}
		if q.Get("insecure") == "1" || strings.EqualFold(q.Get("allowInsecure"), "1") {
			p.Extra["skip-cert-verify"] = true
		}
		if obfs := q.Get("obfs"); obfs != "" {
			p.Extra["obfs"] = obfs
			p.Extra["obfs-password"] = q.Get("obfs-password")
		}
	}
	return p, nil
}

func parseSS(line string) (config.Proxy, error) {
	rest := strings.TrimPrefix(line, "ss://")
	name := ""
	if i := strings.LastIndex(rest, "#"); i >= 0 {
		name, _ = url.PathUnescape(rest[i+1:])
		rest = rest[:i]
	}
	if i := strings.LastIndex(rest, "?"); i >= 0 {
		rest = rest[:i] // drop plugin params for now
	}

	var method, password, hostport string
	if at := strings.LastIndex(rest, "@"); at >= 0 {
		cred := rest[:at]
		hostport = rest[at+1:]
		if decoded, err := decodeBase64(cred); err == nil && strings.Contains(string(decoded), ":") {
			cred = string(decoded)
		}
		parts := strings.SplitN(cred, ":", 2)
		if len(parts) == 2 {
			method, password = parts[0], parts[1]
		}
	} else {
		decoded, err := decodeBase64(rest)
		if err != nil {
			return config.Proxy{}, fmt.Errorf("ss: decode: %w", err)
		}
		text := string(decoded)
		at := strings.LastIndex(text, "@")
		if at < 0 {
			return config.Proxy{}, fmt.Errorf("ss: invalid payload")
		}
		cred, hp := text[:at], text[at+1:]
		parts := strings.SplitN(cred, ":", 2)
		if len(parts) != 2 {
			return config.Proxy{}, fmt.Errorf("ss: invalid credentials")
		}
		method, password = parts[0], parts[1]
		hostport = hp
	}

	host, port := splitHostPort(hostport)
	if host == "" || port == 0 || method == "" {
		return config.Proxy{}, fmt.Errorf("ss: missing host/port/method")
	}
	if name == "" {
		name = fmt.Sprintf("%s:%d", host, port)
	}
	return config.Proxy{
		Name: name, Type: "ss", Server: host, Port: port,
		Extra: map[string]any{"cipher": method, "password": password},
	}, nil
}

// EncodeProxy converts a proxy back into a share link.
func EncodeProxy(p config.Proxy) (string, error) {
	switch strings.ToLower(p.Type) {
	case "vmess":
		return encodeVMess(p)
	case "vless", "trojan", "hysteria2":
		return encodeURLProxy(p)
	case "ss":
		return encodeSS(p)
	default:
		return "", fmt.Errorf("unsupported proxy type %q", p.Type)
	}
}

func encodeVMess(p config.Proxy) (string, error) {
	m := map[string]any{
		"v": "2", "ps": p.Name, "add": p.Server, "port": strconv.Itoa(p.Port),
		"id": str(p.Extra, "uuid"), "aid": "0", "scy": "auto", "net": "tcp", "type": "none",
	}
	if v, ok := p.Extra["alterId"]; ok {
		m["aid"] = fmt.Sprint(v)
	}
	if v := str(p.Extra, "cipher"); v != "" {
		m["scy"] = v
	}
	if v := str(p.Extra, "network"); v != "" {
		m["net"] = v
	}
	if v, ok := p.Extra["tls"].(bool); ok && v {
		m["tls"] = "tls"
		m["sni"] = str(p.Extra, "servername")
	}
	if opts, ok := p.Extra["ws-opts"].(map[string]any); ok {
		m["path"] = str(opts, "path")
		if headers, ok := opts["headers"].(map[string]any); ok {
			m["host"] = str(headers, "Host")
		}
	}
	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return "vmess://" + base64.StdEncoding.EncodeToString(data), nil
}

func encodeURLProxy(p config.Proxy) (string, error) {
	typ := strings.ToLower(p.Type)
	q := url.Values{}
	var user string
	switch typ {
	case "vless":
		user = str(p.Extra, "uuid")
		q.Set("encryption", defaultString(str(p.Extra, "encryption"), "none"))
		if v, ok := p.Extra["tls"].(bool); ok && v {
			if ro, ok := p.Extra["reality-opts"].(map[string]any); ok {
				q.Set("security", "reality")
				q.Set("pbk", str(ro, "public-key"))
				q.Set("sid", str(ro, "short-id"))
			} else {
				q.Set("security", "tls")
			}
			if sni := str(p.Extra, "servername"); sni != "" {
				q.Set("sni", sni)
			}
			if fp := str(p.Extra, "client-fingerprint"); fp != "" {
				q.Set("fp", fp)
			}
		}
		if flow := str(p.Extra, "flow"); flow != "" {
			q.Set("flow", flow)
		}
	case "trojan":
		user = str(p.Extra, "password")
		if sni := str(p.Extra, "sni"); sni != "" {
			q.Set("sni", sni)
		}
		if v, ok := p.Extra["skip-cert-verify"].(bool); ok && v {
			q.Set("allowInsecure", "1")
		}
	case "hysteria2":
		user = str(p.Extra, "password")
		if sni := str(p.Extra, "sni"); sni != "" {
			q.Set("sni", sni)
		}
		if v, ok := p.Extra["skip-cert-verify"].(bool); ok && v {
			q.Set("insecure", "1")
		}
		if v := str(p.Extra, "obfs"); v != "" {
			q.Set("obfs", v)
			q.Set("obfs-password", str(p.Extra, "obfs-password"))
		}
	default:
		return "", fmt.Errorf("unsupported type %q", p.Type)
	}
	applyTransportEncode(q, p.Extra)
	u := url.URL{
		Scheme:   typ,
		User:     url.User(user),
		Host:     net.JoinHostPort(p.Server, strconv.Itoa(p.Port)),
		RawQuery: q.Encode(),
		Fragment: p.Name,
	}
	return u.String(), nil
}

func encodeSS(p config.Proxy) (string, error) {
	method := str(p.Extra, "cipher")
	password := str(p.Extra, "password")
	if method == "" || password == "" {
		return "", fmt.Errorf("ss: missing cipher/password")
	}
	cred := base64.StdEncoding.EncodeToString([]byte(method + ":" + password))
	host := net.JoinHostPort(p.Server, strconv.Itoa(p.Port))
	frag := ""
	if p.Name != "" {
		frag = "#" + url.PathEscape(p.Name)
	}
	return "ss://" + cred + "@" + host + frag, nil
}

// EncodeSubscription converts proxies into a base64 subscription body.
func EncodeSubscription(proxies []config.Proxy) ([]byte, error) {
	var lines []string
	var lastErr error
	for _, p := range proxies {
		link, err := EncodeProxy(p)
		if err != nil {
			lastErr = err
			continue
		}
		lines = append(lines, link)
	}
	if len(lines) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("no encodable proxies")
	}
	return []byte(base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))), nil
}

func applyTransport(extra map[string]any, q url.Values) {
	switch q.Get("type") {
	case "ws":
		extra["network"] = "ws"
		extra["ws-opts"] = map[string]any{
			"path":    q.Get("path"),
			"headers": map[string]any{"Host": q.Get("host")},
		}
	case "grpc":
		extra["network"] = "grpc"
		extra["grpc-opts"] = map[string]any{"grpc-service-name": q.Get("serviceName")}
	}
}

func applyTransportEncode(q url.Values, extra map[string]any) {
	network := str(extra, "network")
	switch network {
	case "ws":
		q.Set("type", "ws")
		if opts, ok := extra["ws-opts"].(map[string]any); ok {
			q.Set("path", str(opts, "path"))
			if headers, ok := opts["headers"].(map[string]any); ok {
				q.Set("host", str(headers, "Host"))
			}
		}
	case "grpc":
		q.Set("type", "grpc")
		if opts, ok := extra["grpc-opts"].(map[string]any); ok {
			q.Set("serviceName", str(opts, "grpc-service-name"))
		}
	}
}

func fragmentName(u *url.URL) string {
	if u.Fragment == "" {
		return ""
	}
	name, err := url.PathUnescape(u.Fragment)
	if err != nil {
		return u.Fragment
	}
	return name
}

func splitHostPort(hostport string) (string, int) {
	hostport = strings.TrimSpace(hostport)
	if host, portStr, err := net.SplitHostPort(hostport); err == nil {
		port, _ := strconv.Atoi(portStr)
		return host, port
	}
	if i := strings.LastIndex(hostport, ":"); i >= 0 {
		port, _ := strconv.Atoi(hostport[i+1:])
		return strings.Trim(hostport[:i], "[]"), port
	}
	return "", 0
}

func decodeBase64(s string) ([]byte, error) {
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		if data, err := enc.DecodeString(s); err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("invalid base64")
}

func str(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprint(x)
	}
}

func integer(m map[string]any, key string) int {
	s := str(m, key)
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
