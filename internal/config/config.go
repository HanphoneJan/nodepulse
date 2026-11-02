package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Proxy is one proxy entry. Extra keeps every unknown field so that
// re-exporting a filtered config does not drop protocol-specific options
// (Reality keys, WS paths, Hysteria2 passwords, ...).
type Proxy struct {
	Name   string         `yaml:"name" json:"name"`
	Type   string         `yaml:"type" json:"type"`
	Server string         `yaml:"server,omitempty" json:"server,omitempty"`
	Port   int            `yaml:"port,omitempty" json:"port,omitempty"`
	Extra  map[string]any `yaml:",inline" json:"-"`
}

// Group is a proxy-group entry. Only the fields NodePulse needs are typed;
// everything else is preserved in Extra.
type Group struct {
	Name      string         `yaml:"name" json:"name"`
	Type      string         `yaml:"type" json:"type"`
	Proxies   []string       `yaml:"proxies,omitempty" json:"proxies,omitempty"`
	URL       string         `yaml:"url,omitempty" json:"url,omitempty"`
	Interval  int            `yaml:"interval,omitempty" json:"interval,omitempty"`
	Tolerance int            `yaml:"tolerance,omitempty" json:"tolerance,omitempty"`
	Extra     map[string]any `yaml:",inline" json:"-"`
}

// Config is the subset of a Clash / mihomo config that NodePulse understands.
type Config struct {
	MixedPort          int            `yaml:"mixed-port,omitempty" json:"mixed_port,omitempty"`
	Port               int            `yaml:"port,omitempty" json:"port,omitempty"`
	SocksPort          int            `yaml:"socks-port,omitempty" json:"socks_port,omitempty"`
	AllowLAN           *bool          `yaml:"allow-lan,omitempty" json:"allow_lan,omitempty"`
	Mode               string         `yaml:"mode,omitempty" json:"mode,omitempty"`
	LogLevel           string         `yaml:"log-level,omitempty" json:"log_level,omitempty"`
	ExternalController string         `yaml:"external-controller,omitempty" json:"external_controller,omitempty"`
	Secret             string         `yaml:"secret,omitempty" json:"secret,omitempty"`
	Proxies            []Proxy        `yaml:"proxies,omitempty" json:"proxies,omitempty"`
	ProxyGroups        []Group        `yaml:"proxy-groups,omitempty" json:"proxy_groups,omitempty"`
	Rules              []string       `yaml:"rules,omitempty" json:"rules,omitempty"`
	Extra              map[string]any `yaml:",inline" json:"-"`
}

// Load reads a Clash / mihomo YAML config from disk.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// Parse decodes a Clash / mihomo YAML config.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Marshal encodes the config back to YAML.
func (c *Config) Marshal() ([]byte, error) {
	return yaml.Marshal(c)
}

// Write saves the config as YAML with permissions 0600.
func (c *Config) Write(path string) error {
	data, err := c.Marshal()
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	_ = os.Chmod(path, 0o600)
	return nil
}

// RealProxies returns entries that can actually be dialed: non-empty name and
// server, and not a built-in outbound type.
func (c *Config) RealProxies() []Proxy {
	out := make([]Proxy, 0, len(c.Proxies))
	for _, p := range c.Proxies {
		if p.Name == "" || p.Server == "" {
			continue
		}
		switch strings.ToLower(p.Type) {
		case "direct", "reject", "rejectdrop", "pass":
			continue
		}
		out = append(out, p)
	}
	return out
}

// FindProxy looks a proxy up by exact name.
func (c *Config) FindProxy(name string) (Proxy, bool) {
	for _, p := range c.Proxies {
		if p.Name == name {
			return p, true
		}
	}
	return Proxy{}, false
}

var infoNodeMarkers = []string{
	"剩余流量", "套餐到期", "流量重置", "距离下次", "到期时间",
	"官网", "客服", "公告",
	"remaining traffic", "expire", "expiry", "traffic reset",
}

// IsInfoNode reports whether a proxy name looks like a provider info entry
// (traffic, expiry, announcements) rather than a real node.
func IsInfoNode(name string) bool {
	lower := strings.ToLower(name)
	for _, marker := range infoNodeMarkers {
		if strings.Contains(lower, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

// Clone returns a shallow copy of the config with slice fields copied so the
// caller can modify them without touching the original.
func (c *Config) Clone() *Config {
	out := *c
	out.Proxies = append([]Proxy(nil), c.Proxies...)
	out.ProxyGroups = append([]Group(nil), c.ProxyGroups...)
	out.Rules = append([]string(nil), c.Rules...)
	return &out
}

// DeepClone copies the whole config, including unknown inline fields such as
// DNS, TUN and rule-providers. Export preservation uses it so daily-use
// configs are not reduced to a minimal file.
func (c *Config) DeepClone() (*Config, error) {
	data, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	var out Config
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("unmarshal config copy: %w", err)
	}
	return &out, nil
}
