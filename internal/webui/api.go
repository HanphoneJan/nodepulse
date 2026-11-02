package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/daemon"
	"github.com/HanphoneJan/nodepulse/internal/discover"
	"github.com/HanphoneJan/nodepulse/internal/export"
	"github.com/HanphoneJan/nodepulse/internal/optimize"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/settings"
	"github.com/HanphoneJan/nodepulse/internal/store"
	"github.com/HanphoneJan/nodepulse/internal/subscription"
	"github.com/HanphoneJan/nodepulse/internal/version"
)

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]string{"version": version.Version})
}

type bootstrapResult struct {
	Controllers   []discover.Controller `json:"controllers"`
	Mihomo        string                `json:"mihomo,omitempty"`
	Subscriptions []string              `json:"subscriptions"`
	Settings      settings.Settings     `json:"settings"`
	MihomoError   string                `json:"mihomo_error,omitempty"`
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st, err := settings.Load("")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if st.API == "" {
		st.API = s.opts.API
	}
	if st.Secret == "" {
		st.Secret = s.opts.Secret
	}
	if st.ConfigPath == "" {
		st.ConfigPath = s.opts.ConfigPath
	}
	if st.MihomoPath == "" {
		st.MihomoPath = s.opts.MihomoPath
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	controllers := discover.Controllers(ctx)
	if st.API == "" && len(controllers) > 0 {
		st.API = controllers[0].URL
	}
	mihomo, mihomoErr := discover.MihomoBinary()
	if st.MihomoPath == "" && mihomoErr == nil {
		st.MihomoPath = mihomo
	}
	files := discover.SubscriptionFiles()
	if st.ConfigPath == "" && len(files) > 0 {
		st.ConfigPath = files[0]
	}
	st.SyncActiveProfile()
	out := bootstrapResult{
		Controllers:   controllers,
		Mihomo:        mihomo,
		Subscriptions: files,
		Settings:      st,
	}
	if mihomoErr != nil {
		out.MihomoError = mihomoErr.Error()
	}
	writeJSON(w, out)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		st, err := settings.Load("")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, st)
	case http.MethodPost:
		var st settings.Settings
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&st); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := settings.Save("", st); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]bool{"ok": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type oneClickRequest struct {
	API         string `json:"api"`
	Secret      string `json:"secret"`
	Group       string `json:"group"`
	ConfigPath  string `json:"config_path"`
	MihomoPath  string `json:"mihomo_path"`
	URL         string `json:"url"`
	Rounds      int    `json:"rounds"`
	Concurrency int    `json:"concurrency"`
	TimeoutMs   int    `json:"timeout_ms"`
	DoSwitch    bool   `json:"switch"`
}

func (s *Server) handleOneClick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req oneClickRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := optimize.Run(r.Context(), optimize.Request{
		API:         req.API,
		Secret:      req.Secret,
		Group:       req.Group,
		ConfigPath:  req.ConfigPath,
		MihomoPath:  req.MihomoPath,
		URL:         req.URL,
		Rounds:      req.Rounds,
		Concurrency: req.Concurrency,
		TimeoutMs:   req.TimeoutMs,
		Switch:      req.DoSwitch,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	s.mu.Lock()
	s.lastResults = res.Results
	s.lastConfigPath = req.ConfigPath
	s.mu.Unlock()
	if len(res.Results) > 0 {
		url := req.URL
		if url == "" {
			url = "https://www.gstatic.com/generate_204"
		}
		_ = writeJSONFile(s.opts.ResultsPath, resultsFile{
			GeneratedAt: time.Now().Format(time.RFC3339),
			URL:         url,
			Rounds:      req.Rounds,
			ConfigPath:  req.ConfigPath,
			Results:     res.Results,
		})
	}

	st, _ := settings.Load("")
	if req.API != "" {
		st.API = req.API
		st.Secret = req.Secret
	}
	if req.Group != "" {
		st.Group = req.Group
	}
	if req.ConfigPath != "" {
		st.ConfigPath = req.ConfigPath
	}
	if req.MihomoPath != "" {
		st.MihomoPath = req.MihomoPath
	}
	if req.Rounds > 0 {
		st.Rounds = req.Rounds
	}
	_ = settings.Save("", st)
	writeJSON(w, res)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rawURL := strings.TrimSpace(req.URL)
	if rawURL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		writeError(w, http.StatusBadRequest, "订阅 URL 必须是 http 或 https")
		return
	}
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, "下载订阅失败: "+err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("下载订阅失败: HTTP %d", resp.StatusCode))
		return
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if _, err := subscription.LoadReader(strings.NewReader(string(data))); err != nil {
		writeError(w, http.StatusBadRequest, "订阅内容无法解析: "+err.Error())
		return
	}
	dir := filepath.Join(filepath.Dir(settings.DefaultPath()), "subscriptions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	path := filepath.Join(dir, fmt.Sprintf("sub-%s.yaml", time.Now().Format("20060102-150405")))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	st, _ := settings.Load("")
	st.ConfigPath = path
	_ = settings.Save("", st)
	writeJSON(w, map[string]string{"path": path})
}

func (s *Server) handleResultsClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	s.lastResults = nil
	s.lastConfigPath = ""
	s.mu.Unlock()
	if err := os.Remove(s.opts.ResultsPath); err != nil && !os.IsNotExist(err) {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Out        string  `json:"out"`
		MinSuccess float64 `json:"min_success"`
		Minimal    bool    `json:"minimal"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req)
	s.mu.Lock()
	results := append([]score.Result(nil), s.lastResults...)
	configPath := s.lastConfigPath
	s.mu.Unlock()
	if configPath == "" {
		if st, err := settings.Load(""); err == nil {
			configPath = st.ConfigPath
		}
	}
	if len(results) == 0 || configPath == "" {
		writeError(w, http.StatusBadRequest, "请先测试一次")
		return
	}
	if req.Out == "" {
		req.Out = "stable.yaml"
	}
	if req.MinSuccess <= 0 {
		req.MinSuccess = 0.8
	}
	cfg, err := subscription.Load(configPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sel, err := export.Stable(cfg, results, export.Options{MinSuccess: req.MinSuccess, Preserve: !req.Minimal})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	dir := exportDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	name := filepath.Base(strings.TrimSpace(req.Out))
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "stable.yaml"
	}
	outPath := filepath.Join(dir, name)
	if err := sel.Config.Write(outPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	mode := "保留原规则和分组"
	if !sel.Preserved {
		mode = "最小配置"
	}
	writeJSON(w, map[string]string{
		"message": fmt.Sprintf("已写入 %s（%s）：保留 %d 个，排除 %d 个", outPath, mode, len(sel.Included), len(sel.Excluded)),
	})
}

type daemonRequest struct {
	API         string  `json:"api"`
	Secret      string  `json:"secret"`
	Group       string  `json:"group"`
	IntervalSec int     `json:"interval_sec"`
	Rounds      int     `json:"rounds"`
	Margin      float64 `json:"margin"`
	Confirm     int     `json:"confirm"`
	MinSuccess  float64 `json:"min_success"`
}

func (s *Server) handleDaemonStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req daemonRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.API == "" {
		writeError(w, http.StatusBadRequest, "api is required")
		return
	}
	s.mu.Lock()
	if s.daemonRunning {
		s.mu.Unlock()
		writeError(w, http.StatusConflict, "daemon already running")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.daemonSeq++
	id := s.daemonSeq
	done := make(chan struct{})
	s.daemonCancel = cancel
	s.daemonRunning = true
	s.daemonDone = done
	s.mu.Unlock()

	st, err := store.New(s.opts.HistoryPath)
	if err != nil {
		s.mu.Lock()
		if s.daemonSeq == id {
			s.daemonRunning = false
			s.daemonCancel = nil
			s.daemonDone = nil
		}
		close(done)
		s.mu.Unlock()
		cancel()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	d := daemon.New(clash.New(req.API, req.Secret), daemon.Options{
		Group:       req.Group,
		Interval:    time.Duration(req.IntervalSec) * time.Second,
		Rounds:      req.Rounds,
		Concurrency: 12,
		TestURL:     "https://www.gstatic.com/generate_204",
		Timeout:     5 * time.Second,
		Margin:      req.Margin,
		Confirm:     req.Confirm,
		MinSuccess:  req.MinSuccess,
		Store:       st,
	})
	go func() {
		_ = d.Run(ctx)
		s.mu.Lock()
		if s.daemonSeq == id {
			s.daemonRunning = false
			s.daemonCancel = nil
			s.daemonDone = nil
		}
		s.mu.Unlock()
		close(done)
	}()
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) handleDaemonStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	cancel := s.daemonCancel
	done := s.daemonDone
	if cancel == nil {
		s.mu.Unlock()
		writeJSON(w, map[string]bool{"ok": true})
		return
	}
	s.daemonCancel = nil
	s.daemonRunning = false
	s.daemonDone = nil
	s.mu.Unlock()

	cancel()
	if done != nil {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) handleDaemonStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	running := s.daemonRunning
	s.mu.Unlock()
	writeJSON(w, map[string]bool{"running": running})
}

func exportDir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "nodepulse-exports"
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "nodepulse", "exports")
}
