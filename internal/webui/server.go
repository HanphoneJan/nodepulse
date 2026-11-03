package webui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/lock"
	"github.com/HanphoneJan/nodepulse/internal/optimize"
	"github.com/HanphoneJan/nodepulse/internal/progress"
	"github.com/HanphoneJan/nodepulse/internal/results"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/sidecar"
	"github.com/HanphoneJan/nodepulse/internal/store"
	"github.com/HanphoneJan/nodepulse/internal/subscription"
	"github.com/HanphoneJan/nodepulse/internal/tester"
	"github.com/HanphoneJan/nodepulse/internal/uiassets"
)

// Options configures the local web UI.
type Options struct {
	Addr        string
	ResultsPath string
	HistoryPath string
	API         string
	Secret      string
	ConfigPath  string
	MihomoPath  string
	LockPath    string
}

// Server is the local web dashboard.
type Server struct {
	opts Options
	mux  *http.ServeMux

	mu             sync.Mutex
	lastResults    []score.Result
	lastConfigPath string
	daemonCancel   context.CancelFunc
	daemonRunning  bool
	daemonSeq      uint64
	daemonDone     chan struct{}

	progress    *progress.Tracker
	testCancel  context.CancelFunc
	testRunning bool

	daemonLock *lock.Lock
}

// New creates a server with defaults.
func New(opts Options) *Server {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:8099"
	}
	if opts.ResultsPath == "" {
		opts.ResultsPath = "results.json"
	}
	if opts.HistoryPath == "" {
		opts.HistoryPath = store.DefaultPath()
	}
	s := &Server{opts: opts, mux: http.NewServeMux(), progress: progress.New()}
	if data, err := readFile(opts.ResultsPath); err == nil {
		var rf resultsFile
		if json.Unmarshal(data, &rf) == nil && len(rf.Results) > 0 {
			s.lastResults = rf.Results
			s.lastConfigPath = rf.ConfigPath
		}
	}

	s.mux.Handle("/", http.FileServer(http.FS(uiassets.Dist())))
	s.mux.HandleFunc("/api/version", s.handleVersion)
	s.mux.HandleFunc("/api/progress", s.handleProgress)
	s.mux.HandleFunc("/api/cancel", s.handleCancel)
	s.mux.HandleFunc("/api/results", s.handleResults)
	s.mux.HandleFunc("/api/results/clear", s.handleResultsClear)
	s.mux.HandleFunc("/api/history", s.handleHistory)
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/test", s.handleTest)
	s.mux.HandleFunc("/api/test/all", s.handleTestAll)
	s.mux.HandleFunc("/api/switch", s.handleSwitch)
	s.mux.HandleFunc("/api/bootstrap", s.handleBootstrap)
	s.mux.HandleFunc("/api/settings", s.handleSettings)
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/settings/backup", s.handleSettingsBackup)
	s.mux.HandleFunc("/api/settings/restore", s.handleSettingsRestore)
	s.mux.HandleFunc("/api/oneclick", s.handleOneClick)
	s.mux.HandleFunc("/api/import", s.handleImport)
	s.mux.HandleFunc("/api/export", s.handleExport)
	s.mux.HandleFunc("/api/export/preview", s.handleExportPreview)
	s.mux.HandleFunc("/api/daemon/start", s.handleDaemonStart)
	s.mux.HandleFunc("/api/daemon/stop", s.handleDaemonStop)
	s.mux.HandleFunc("/api/daemon/status", s.handleDaemonStatus)
	return s
}

// Handler exposes the secured mux (used by tests).
func (s *Server) Handler() http.Handler { return secureHandler(s.mux) }

// secureHandler keeps the local management API local and same-origin. It also
// prevents API responses (settings/secret) from being cached.
func secureHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "no-referrer")
			if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
				if origin := r.Header.Get("Origin"); origin != "" {
					u, err := url.Parse(origin)
					if err != nil || !strings.EqualFold(u.Host, r.Host) {
						http.Error(w, "cross-origin request rejected", http.StatusForbidden)
						return
					}
				}
				if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
					http.Error(w, "cross-site request rejected", http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Run serves until the context is canceled.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{Addr: s.opts.Addr, Handler: s.Handler()}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.progress.Snapshot())
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]bool{"canceled": s.progress.Cancel()})
}

func (s *Server) beginTest(parent context.Context, kind string) (context.Context, func(error), error) {
	s.mu.Lock()
	if s.testRunning {
		s.mu.Unlock()
		return nil, nil, fmt.Errorf("已有测速任务在运行")
	}
	ctx, cancel := context.WithCancel(parent)
	s.testRunning = true
	s.testCancel = cancel
	s.mu.Unlock()
	s.progress.Begin(kind, cancel)
	finish := func(err error) {
		s.mu.Lock()
		s.testRunning = false
		s.testCancel = nil
		s.mu.Unlock()
		s.progress.Finish(err)
	}
	return ctx, finish, nil
}

func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := readFile(s.opts.ResultsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, map[string]any{"results": []any{}})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st, err := store.New(s.opts.HistoryPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	events, err := st.Recent(50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(events)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	api := queryOr(r, "api", s.opts.API)
	secret := s.opts.Secret
	switch r.Method {
	case http.MethodGet:
	case http.MethodPost:
		var req struct {
			API    string `json:"api"`
			Secret string `json:"secret"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		api = req.API
		secret = req.Secret
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if api == "" {
		writeError(w, http.StatusBadRequest, "api is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	proxies, err := clash.New(api, secret).Proxies(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	type group struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Now     string `json:"now"`
		Members int    `json:"members"`
	}
	var groups []group
	for name, info := range proxies {
		if !isGroupType(info.Type) {
			continue
		}
		groups = append(groups, group{Name: name, Type: info.Type, Now: info.Now, Members: len(info.All)})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	writeJSON(w, groups)
}

type testRequest struct {
	API         string `json:"api"`
	onProgress  func(done, total int, result score.Result)
	Secret      string `json:"secret"`
	ConfigPath  string `json:"config_path"`
	MihomoPath  string `json:"mihomo_path"`
	URL         string `json:"url"`
	Rounds      int    `json:"rounds"`
	Concurrency int    `json:"concurrency"`
	TimeoutMs   int    `json:"timeout_ms"`
}

func (s *Server) handleTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req testRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.API == "" {
		req.API = s.opts.API
		req.Secret = s.opts.Secret
	}
	if req.ConfigPath == "" {
		req.ConfigPath = s.opts.ConfigPath
	}
	if req.MihomoPath == "" {
		req.MihomoPath = s.opts.MihomoPath
	}
	ctx, finish, err := s.beginTest(r.Context(), "test")
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	req.onProgress = func(done, total int, result score.Result) {
		s.progress.Update(done, total, result.Name)
	}
	results, runErr := s.runTest(ctx, req)
	if runErr == nil && ctx.Err() != nil {
		runErr = ctx.Err()
	}
	finish(runErr)
	if errors.Is(runErr, context.Canceled) {
		writeJSON(w, map[string]bool{"canceled": true})
		return
	}
	if runErr != nil {
		writeError(w, http.StatusBadGateway, runErr.Error())
		return
	}
	s.mu.Lock()
	s.lastResults = results
	s.lastConfigPath = req.ConfigPath
	s.mu.Unlock()
	writeJSON(w, results)
}

func (s *Server) handleTestAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		API         string `json:"api"`
		Secret      string `json:"secret"`
		MihomoPath  string `json:"mihomo_path"`
		URL         string `json:"url"`
		Rounds      int    `json:"rounds"`
		Concurrency int    `json:"concurrency"`
		TimeoutMs   int    `json:"timeout_ms"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.API == "" {
		req.API = s.opts.API
		req.Secret = s.opts.Secret
	}
	if req.MihomoPath == "" {
		req.MihomoPath = s.opts.MihomoPath
	}
	ctx, finish, err := s.beginTest(r.Context(), "all")
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	res, runErr := optimize.Run(ctx, optimize.Request{
		API:              req.API,
		Secret:           req.Secret,
		MihomoPath:       req.MihomoPath,
		URL:              req.URL,
		Rounds:           req.Rounds,
		Concurrency:      req.Concurrency,
		TimeoutMs:        req.TimeoutMs,
		AllSubscriptions: true,
		OnProgress: func(done, total int, result score.Result) {
			s.progress.Update(done, total, result.Name)
		},
	})
	if runErr == nil && ctx.Err() != nil {
		runErr = ctx.Err()
	}
	finish(runErr)
	if errors.Is(runErr, context.Canceled) {
		writeJSON(w, map[string]bool{"canceled": true})
		return
	}
	if runErr != nil {
		writeError(w, http.StatusBadGateway, runErr.Error())
		return
	}
	s.mu.Lock()
	s.lastResults = res.Results
	s.lastConfigPath = ""
	s.mu.Unlock()
	if len(res.Results) > 0 {
		url := req.URL
		if url == "" {
			url = "https://www.gstatic.com/generate_204"
		}
		_ = results.Save(s.opts.ResultsPath, results.File{
			GeneratedAt: time.Now().Format(time.RFC3339),
			URL:         url,
			Rounds:      req.Rounds,
			Results:     res.Results,
		})
	}
	writeJSON(w, res)
}

func (s *Server) runTest(ctx context.Context, req testRequest) ([]score.Result, error) {
	if req.Rounds <= 0 {
		req.Rounds = 5
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 8
	}
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 5000
	}
	if req.URL == "" {
		req.URL = "https://www.gstatic.com/generate_204"
	}
	if req.ConfigPath == "" && req.API == "" {
		return nil, fmt.Errorf("api or config_path is required")
	}

	var cfg *config.Config
	var nodes []config.Proxy
	if req.ConfigPath != "" {
		loaded, err := subscription.Load(req.ConfigPath)
		if err != nil {
			return nil, err
		}
		cfg = loaded
		nodes = make([]config.Proxy, 0, len(cfg.Proxies))
		for _, n := range cfg.RealProxies() {
			if !config.IsInfoNode(n.Name) {
				nodes = append(nodes, n)
			}
		}
	}

	var delayer tester.Delayer
	var cleanup func()
	if req.API != "" {
		delayer = clash.New(req.API, req.Secret)
		if len(nodes) == 0 {
			proxies, err := clash.New(req.API, req.Secret).Proxies(ctx)
			if err != nil {
				return nil, err
			}
			for name, info := range proxies {
				if isGroupType(info.Type) || !isTestable(info.Type) {
					continue
				}
				nodes = append(nodes, config.Proxy{Name: name, Type: info.Type})
			}
			sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
		}
	} else {
		if req.MihomoPath == "" || cfg == nil {
			return nil, fmt.Errorf("api or mihomo_path is required")
		}
		sc, err := sidecar.Start(ctx, cfg, req.MihomoPath)
		if err != nil {
			return nil, err
		}
		delayer = clash.New(sc.BaseURL, sc.Secret)
		cleanup = func() { _ = sc.Close() }
	}
	if cleanup != nil {
		defer cleanup()
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no testable nodes")
	}

	t := tester.New(delayer, tester.Options{
		Rounds:      req.Rounds,
		Concurrency: req.Concurrency,
		URL:         req.URL,
		Timeout:     time.Duration(req.TimeoutMs) * time.Millisecond,
	})
	if req.onProgress != nil {
		req.onProgress(0, len(nodes), score.Result{})
		t.OnNode(req.onProgress)
	}
	ranked := score.Rank(t.Test(ctx, nodes))

	rf := resultsFile{
		GeneratedAt: time.Now().Format(time.RFC3339),
		URL:         req.URL,
		Rounds:      req.Rounds,
		Results:     ranked,
		ConfigPath:  req.ConfigPath,
	}
	if err := results.Save(s.opts.ResultsPath, rf); err != nil {
		return nil, err
	}
	return ranked, nil
}

func (s *Server) handleSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		API    string `json:"api"`
		Secret string `json:"secret"`
		Group  string `json:"group"`
		Node   string `json:"node"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.API == "" {
		req.API = s.opts.API
		req.Secret = s.opts.Secret
	}
	if req.API == "" || req.Group == "" || req.Node == "" {
		writeError(w, http.StatusBadRequest, "api, group and node are required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := clash.New(req.API, req.Secret).Switch(ctx, req.Group, req.Node); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, map[string]string{"ok": "switched", "group": req.Group, "node": req.Node})
}

type resultsFile = results.File

func readFile(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("file path is empty")
	}
	return readFileOS(path)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func queryOr(r *http.Request, key, fallback string) string {
	if v := strings.TrimSpace(r.URL.Query().Get(key)); v != "" {
		return v
	}
	return fallback
}

func isTestable(t string) bool {
	switch strings.ToLower(t) {
	case "selector", "urltest", "fallback", "loadbalance", "direct", "reject", "rejectdrop", "compatible", "pass", "dns":
		return false
	}
	return true
}

func isGroupType(t string) bool {
	switch strings.ToLower(t) {
	case "selector", "urltest", "fallback", "loadbalance":
		return true
	}
	return false
}
