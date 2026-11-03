package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/config"
)

// Client is a Clash-compatible desktop app found on this machine. Detection is
// local only: it looks at installed paths, running processes and the runtime
// config the client writes, not at the network.
type Client struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Running    bool   `json:"running"`
	Installed  bool   `json:"installed"`
	Controller string `json:"controller,omitempty"`
	Secret     string `json:"secret,omitempty"`
	ConfigPath string `json:"config_path,omitempty"`
	Hint       string `json:"hint,omitempty"`
}

type clientSpec struct {
	id        string
	name      string
	dirs      []string
	processes []string
}

func knownClients() []clientSpec {
	home, _ := os.UserHomeDir()
	roots := configRoots(home)
	join := func(names ...string) []string {
		var out []string
		for _, root := range roots {
			for _, name := range names {
				out = append(out, filepath.Join(root, name))
			}
		}
		return out
	}
	return []clientSpec{
		{
			id:        "clash-verge",
			name:      "Clash Verge",
			dirs:      join("io.github.clash-verge-rev.clash-verge-rev", "clash-verge-rev", "clash-verge"),
			processes: []string{"clash verge", "clash-verge", "clash-verge-rev"},
		},
		{
			id:        "flclash",
			name:      "FlClash",
			dirs:      join("com.follow.clash", "FlClash", "flclash"),
			processes: []string{"flclash", "fl clash"},
		},
		{
			id:        "mihomo-party",
			name:      "Mihomo Party",
			dirs:      join("mihomo-party", "Mihomo Party"),
			processes: []string{"mihomo-party", "mihomo party"},
		},
		{
			id:        "clash-nyanpasu",
			name:      "Clash Nyanpasu",
			dirs:      join("clash-nyanpasu", "Clash Nyanpasu"),
			processes: []string{"clash-nyanpasu", "clash nyanpasu"},
		},
		{
			id:        "clash-for-windows",
			name:      "Clash for Windows",
			dirs:      join("clash", "Clash for Windows"),
			processes: []string{"clash for windows", "clash-win64", "clash-win32"},
		},
	}
}

// configRoots lists the per-user directories desktop clients use for app data.
func configRoots(home string) []string {
	var roots []string
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			roots = append(roots, appData)
		}
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			roots = append(roots, localAppData)
		}
		return roots
	}
	if runtime.GOOS == "darwin" {
		roots = append(roots, filepath.Join(home, "Library", "Application Support"))
		return roots
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		roots = append(roots, xdg)
	} else {
		roots = append(roots, filepath.Join(home, ".local", "share"))
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		roots = append(roots, xdg)
	} else {
		roots = append(roots, filepath.Join(home, ".config"))
	}
	return roots
}

func installRoots(home string) []string {
	switch runtime.GOOS {
	case "windows":
		var out []string
		for _, env := range []string{"LOCALAPPDATA", "PROGRAMFILES", "PROGRAMFILES(X86)"} {
			if base := os.Getenv(env); base != "" {
				out = append(out, filepath.Join(base, "Programs"), base)
			}
		}
		return out
	case "darwin":
		return []string{"/Applications", filepath.Join(home, "Applications")}
	default:
		return []string{"/opt", filepath.Join(home, ".local", "bin"), "/usr/local/bin", "/usr/bin"}
	}
}

// installedClient reports where a client's executable, app bundle or app data
// directory can be found.
func installedClient(spec clientSpec, home string) (dir string, ok bool) {
	for _, candidate := range spec.dirs {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true
		}
	}
	for _, root := range installRoots(home) {
		for _, base := range []string{spec.name, strings.ToLower(strings.ReplaceAll(spec.name, " ", "-")), spec.id} {
			path := filepath.Join(root, base)
			if _, err := os.Stat(path); err == nil {
				return path, true
			}
			if info, err := os.Stat(path + ".exe"); err == nil && !info.IsDir() {
				return path + ".exe", true
			}
			if info, err := os.Stat(path + ".app"); err == nil && info.IsDir() {
				return path + ".app", true
			}
		}
	}
	return "", false
}

func processRunning(processes []string, running []string) bool {
	for _, name := range running {
		name = strings.ToLower(strings.TrimSuffix(filepath.Base(name), ".exe"))
		for _, want := range processes {
			if name == want || strings.Contains(name, want) {
				return true
			}
		}
	}
	return false
}

// Clients returns locally installed or running Clash-compatible desktop apps.
func Clients() []Client {
	home, _ := os.UserHomeDir()
	running := runningProcesses()
	var out []Client
	for _, spec := range knownClients() {
		dir, installed := installedClient(spec, home)
		runningNow := processRunning(spec.processes, running)
		if !installed && !runningNow {
			continue
		}
		client := Client{
			ID:        spec.id,
			Name:      spec.name,
			Running:   runningNow,
			Installed: installed,
		}
		configPath := ""
		if dir != "" {
			candidate := filepath.Join(dir, "config.yaml")
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				configPath = candidate
			}
		}
		if configPath == "" {
			for _, candidate := range spec.dirs {
				path := filepath.Join(candidate, "config.yaml")
				if info, err := os.Stat(path); err == nil && !info.IsDir() {
					configPath = path
					break
				}
			}
		}
		if configPath != "" {
			client.ConfigPath = configPath
			controller, secret := controllerFromConfig(configPath)
			client.Controller = controller
			client.Secret = secret
		}
		if !runningNow {
			client.Hint = "未运行"
		} else if client.Controller == "" {
			if client.ConfigPath == "" {
				client.Hint = "运行中，但没有找到配置文件"
			} else {
				client.Hint = "外部控制未开启，需要在客户端里打开外部控制"
			}
		}
		out = append(out, client)
	}
	return out
}

// controllerFromConfig reads the runtime config a client writes and extracts
// the external controller endpoint and secret. Empty values mean the client
// disabled its external controller.
func controllerFromConfig(path string) (string, string) {
	cfg, err := config.Load(path)
	if err != nil {
		return "", ""
	}
	raw := strings.TrimSpace(cfg.ExternalController)
	if raw == "" {
		return "", ""
	}
	endpoint, err := normalizeControllerURL(raw)
	if err != nil {
		return "", ""
	}
	return endpoint, cfg.Secret
}

func normalizeControllerURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, ":") {
		value = "127.0.0.1" + value
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid controller %q", raw)
	}
	host := parsed.Hostname()
	port := parsed.Port()
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "9090"
	}
	parsed.Host = net.JoinHostPort(host, port)
	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimSuffix(parsed.String(), "/"), nil
}

// Scan detects local clients and probes every controller they advertise plus
// the usual default ports. The client list is returned even when the client
// has no reachable controller, so the UI can explain why.
func Scan(ctx context.Context) ([]Client, []Controller) {
	clients := Clients()
	var candidates []controllerCandidate
	seenClient := map[string]bool{}
	for _, client := range clients {
		if client.Controller == "" || seenClient[client.Controller] {
			continue
		}
		seenClient[client.Controller] = true
		candidates = append(candidates, controllerCandidate{url: client.Controller, secret: client.Secret, client: client.Name})
	}
	controllers := probeControllers(ctx, candidates)
	reachable := map[string]bool{}
	for _, controller := range controllers {
		reachable[controller.URL] = true
	}
	for index := range clients {
		if clients[index].Running && clients[index].Controller != "" && !reachable[clients[index].Controller] {
			clients[index].Hint = "控制口未响应，请检查客户端是否已启动"
		}
	}
	return clients, controllers
}

type controllerCandidate struct {
	url    string
	secret string
	client string
}

// Controllers probes common local Clash controller ports and any controller
// advertised by a detected client config.
func Controllers(ctx context.Context) []Controller {
	var candidates []controllerCandidate
	for _, client := range Clients() {
		if client.Controller == "" {
			continue
		}
		candidates = append(candidates, controllerCandidate{url: client.Controller, secret: client.Secret, client: client.Name})
	}
	return probeControllers(ctx, candidates)
}

func probeControllers(ctx context.Context, candidates []controllerCandidate) []Controller {
	ports := []int{9090, 9097, 9091, 9099, 9092, 9093}
	for _, port := range ports {
		candidates = append(candidates, controllerCandidate{url: fmt.Sprintf("http://127.0.0.1:%d", port)})
	}

	client := &http.Client{Timeout: 1200 * time.Millisecond}
	var out []Controller
	seen := map[string]int{}
	for _, candidate := range candidates {
		url := strings.TrimSuffix(candidate.url, "/")
		if url == "" {
			continue
		}
		if index, ok := seen[url]; ok {
			if out[index].Secret == "" && candidate.secret != "" {
				out[index].Secret = candidate.secret
			}
			if out[index].Client == "" && candidate.client != "" {
				out[index].Client = candidate.client
			}
			continue
		}
		controller, ok := probeController(ctx, client, url, candidate.secret)
		if !ok {
			continue
		}
		controller.Client = candidate.client
		seen[url] = len(out)
		out = append(out, controller)
	}
	return out
}

func probeController(ctx context.Context, client *http.Client, rawURL, secret string) (Controller, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL+"/version", nil)
	if err != nil {
		return Controller{}, false
	}
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Controller{}, false
	}
	defer resp.Body.Close()

	controller := Controller{URL: rawURL, Secret: secret}
	switch resp.StatusCode {
	case http.StatusOK:
		var v struct {
			Version string `json:"version"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&v)
		controller.Version = v.Version
		return controller, true
	case http.StatusUnauthorized, http.StatusForbidden:
		controller.AuthRequired = true
		if secret != "" {
			// The config secret did not work; do not persist a broken value.
			controller.Secret = ""
		}
		return controller, true
	default:
		return Controller{}, false
	}
}
