package webui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
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
	s := &Server{opts: opts, mux: http.NewServeMux()}

	s.mux.Handle("/", http.FileServer(http.FS(uiassets.Dist())))
	s.mux.HandleFunc("/api/version", s.handleVersion)
	s.mux.HandleFunc("/api/results", s.handleResults)
	s.mux.HandleFunc("/api/history", s.handleHistory)
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/test", s.handleTest)
	s.mux.HandleFunc("/api/switch", s.handleSwitch)
	s.mux.HandleFunc("/api/bootstrap", s.handleBootstrap)
	s.mux.HandleFunc("/api/settings", s.handleSettings)
	s.mux.HandleFunc("/api/oneclick", s.handleOneClick)
	s.mux.HandleFunc("/api/import", s.handleImport)
	s.mux.HandleFunc("/api/export", s.handleExport)
	s.mux.HandleFunc("/api/daemon/start", s.handleDaemonStart)
	s.mux.HandleFunc("/api/daemon/stop", s.handleDaemonStop)
	s.mux.HandleFunc("/api/daemon/status", s.handleDaemonStatus)
	return s
}

// Handler exposes the mux (used by tests).
func (s *Server) Handler() http.Handler { return s.mux }

// Run serves until the context is canceled.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{Addr: s.opts.Addr, Handler: s.mux}
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
	secret := queryOr(r, "secret", s.opts.Secret)
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
	results, err := s.runTest(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	s.mu.Lock()
	s.lastResults = results
	s.lastConfigPath = req.ConfigPath
	s.mu.Unlock()
	writeJSON(w, results)
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
	if req.ConfigPath == "" {
		return nil, fmt.Errorf("config_path is required")
	}
	cfg, err := subscription.Load(req.ConfigPath)
	if err != nil {
		return nil, err
	}
	nodes := make([]config.Proxy, 0, len(cfg.Proxies))
	for _, n := range cfg.RealProxies() {
		if !config.IsInfoNode(n.Name) {
			nodes = append(nodes, n)
		}
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no testable nodes")
	}

	var delayer tester.Delayer
	var cleanup func()
	if req.API != "" {
		delayer = clash.New(req.API, req.Secret)
	} else {
		if req.MihomoPath == "" {
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

	t := tester.New(delayer, tester.Options{
		Rounds:      req.Rounds,
		Concurrency: req.Concurrency,
		URL:         req.URL,
		Timeout:     time.Duration(req.TimeoutMs) * time.Millisecond,
	})
	ranked := score.Rank(t.Test(ctx, nodes))

	rf := resultsFile{
		GeneratedAt: time.Now().Format(time.RFC3339),
		URL:         req.URL,
		Rounds:      req.Rounds,
		Results:     ranked,
	}
	if err := writeJSONFile(s.opts.ResultsPath, rf); err != nil {
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

type resultsFile struct {
	GeneratedAt string         `json:"generated_at"`
	URL         string         `json:"url"`
	Rounds      int            `json:"rounds"`
	Results     []score.Result `json:"results"`
}

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

func isGroupType(t string) bool {
	switch strings.ToLower(t) {
	case "selector", "urltest", "fallback", "loadbalance":
		return true
	}
	return false
}
