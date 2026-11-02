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
	"time"
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

// SubscriptionFiles returns likely Clash/mihomo subscription files on this
// machine, newest first.
func SubscriptionFiles() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var out []string
	add := func(p string) {
		if p == "" {
			return
		}
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			out = append(out, p)
		}
	}
	add(filepath.Join(home, ".local", "share", "com.follow.clash", "config.yaml"))
	add(filepath.Join(home, ".config", "clash", "config.yaml"))
	add(filepath.Join(home, ".config", "mihomo", "config.yaml"))

	profileDirs := []string{
		filepath.Join(home, ".local", "share", "io.github.clash-verge-rev.clash-verge-rev", "profiles"),
		filepath.Join(home, ".config", "clash-verge", "profiles"),
		filepath.Join(home, ".config", "clash-verge-rev", "profiles"),
	}
	for _, dir := range profileDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := filepath.Ext(e.Name())
			if ext == ".yaml" || ext == ".yml" {
				add(filepath.Join(dir, e.Name()))
			}
		}
	}

	// De-duplicate and sort newest first.
	seen := map[string]bool{}
	var unique []string
	for _, p := range out {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}
	sort.SliceStable(unique, func(i, j int) bool {
		si, _ := os.Stat(unique[i])
		sj, _ := os.Stat(unique[j])
		if si == nil || sj == nil {
			return unique[i] < unique[j]
		}
		return si.ModTime().After(sj.ModTime())
	})
	if len(unique) > 20 {
		unique = unique[:20]
	}
	return unique
}
