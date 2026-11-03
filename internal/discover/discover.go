package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/config"
)

// Controller is a detected Clash-compatible control endpoint.
type Controller struct {
	URL          string `json:"url"`
	Version      string `json:"version,omitempty"`
	AuthRequired bool   `json:"auth_required"`
}

// Controllers probes common local Clash controller ports.
func Controllers(ctx context.Context) []Controller {
	ports := []int{9090, 9097, 9091, 9099, 9092, 9093}
	client := &http.Client{Timeout: 1200 * time.Millisecond}
	var out []Controller
	seen := map[string]bool{}
	for _, port := range ports {
		url := fmt.Sprintf("http://127.0.0.1:%d", port)
		if seen[url] {
			continue
		}
		seen[url] = true
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/version", nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var v struct {
				Version string `json:"version"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&v)
			out = append(out, Controller{URL: url, Version: v.Version})
		case http.StatusUnauthorized, http.StatusForbidden:
			out = append(out, Controller{URL: url, AuthRequired: true})
		}
		resp.Body.Close()
	}
	return out
}

// MihomoBinary finds a usable mihomo executable.
func MihomoBinary() (string, error) {
	candidates := []string{
		os.Getenv("NODEPULSE_MIHOMO"),
		"mihomo",
		"/usr/local/bin/mihomo",
		"/usr/bin/mihomo",
		"/opt/mihomo/mihomo",
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", "mihomo"),
			filepath.Join(home, "bin", "mihomo"),
		)
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("未找到 mihomo（可设置 NODEPULSE_MIHOMO，或放到 PATH）")
}

// Profile is one locally discoverable subscription/profile file.
type Profile struct {
	Path   string `json:"path"`
	Label  string `json:"label"`
	Client string `json:"client"`
	Active bool   `json:"active"`
}

// Profiles returns local profiles for supported clients, newest first. The
// controller only exposes the active profile, so inactive subscriptions must
// be read from these files.
func Profiles() []Profile {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var out []Profile
	seen := map[string]bool{}
	add := func(path, client, label string, active bool) {
		if path == "" || seen[path] {
			return
		}
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			return
		}
		seen[path] = true
		out = append(out, Profile{
			Path:   path,
			Label:  profileLabel(path, label),
			Client: client,
			Active: active,
		})
	}

	add(filepath.Join(home, ".local", "share", "com.follow.clash", "config.yaml"), "FlClash", "FlClash 当前订阅", true)
	flclashDirs := []string{
		filepath.Join(home, ".local", "share", "com.follow.clash", "profiles"),
		filepath.Join(home, ".config", "com.follow.clash", "profiles"),
	}
	for _, dir := range flclashDirs {
		addProfileDir(dir, "FlClash", add)
	}

	add(filepath.Join(home, ".config", "clash", "config.yaml"), "Clash", "Clash 当前配置", true)
	add(filepath.Join(home, ".config", "mihomo", "config.yaml"), "Mihomo", "Mihomo 当前配置", true)

	vergeDirs := []string{
		filepath.Join(home, ".local", "share", "io.github.clash-verge-rev.clash-verge-rev", "profiles"),
		filepath.Join(home, ".config", "clash-verge", "profiles"),
		filepath.Join(home, ".config", "clash-verge-rev", "profiles"),
	}
	for _, dir := range vergeDirs {
		addProfileDir(dir, "Clash Verge", add)
	}

	sort.SliceStable(out, func(i, j int) bool {
		si, _ := os.Stat(out[i].Path)
		sj, _ := os.Stat(out[j].Path)
		if si == nil || sj == nil {
			return out[i].Path < out[j].Path
		}
		return si.ModTime().After(sj.ModTime())
	})
	if len(out) > 20 {
		out = out[:20]
	}
	return out
}

func addProfileDir(dir, client string, add func(path, client, label string, active bool)) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if strings.EqualFold(entry.Name(), "profiles.yaml") || strings.EqualFold(entry.Name(), "profiles.yml") {
			continue
		}
		add(filepath.Join(dir, entry.Name()), client, strings.TrimSuffix(entry.Name(), ext), false)
	}
}

func profileLabel(path, fallback string) string {
	cfg, err := config.Load(path)
	if err == nil && len(cfg.ProxyGroups) > 0 && strings.TrimSpace(cfg.ProxyGroups[0].Name) != "" {
		return cfg.ProxyGroups[0].Name
	}
	return fallback
}

// SubscriptionFiles returns profile paths for backwards-compatible UI callers.
func SubscriptionFiles() []string {
	profiles := Profiles()
	out := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		out = append(out, profile.Path)
	}
	return out
}
