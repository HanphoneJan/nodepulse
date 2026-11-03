package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/autostart"
	"github.com/HanphoneJan/nodepulse/internal/backup"
	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/daemon"
	"github.com/HanphoneJan/nodepulse/internal/discover"
	"github.com/HanphoneJan/nodepulse/internal/export"
	"github.com/HanphoneJan/nodepulse/internal/health"
	"github.com/HanphoneJan/nodepulse/internal/lock"
	"github.com/HanphoneJan/nodepulse/internal/netfetch"
	"github.com/HanphoneJan/nodepulse/internal/optimize"
	"github.com/HanphoneJan/nodepulse/internal/progress"
	"github.com/HanphoneJan/nodepulse/internal/results"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/settings"
	"github.com/HanphoneJan/nodepulse/internal/sidecar"
	"github.com/HanphoneJan/nodepulse/internal/store"
	"github.com/HanphoneJan/nodepulse/internal/subscription"
	"github.com/HanphoneJan/nodepulse/internal/tester"
	"github.com/HanphoneJan/nodepulse/internal/version"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const defaultTestURL = "https://www.gstatic.com/generate_204"

// App is the Wails binding surface.
type App struct {
	ctx context.Context

	mu             sync.Mutex
	lastResults    []score.Result
	lastConfigPath string

	daemonCancel  context.CancelFunc
	daemonRunning bool
	daemonSeq     uint64
	daemonDone    chan struct{}
	daemonLock    *lock.Lock

	progress    *progress.Tracker
	testCancel  context.CancelFunc
	testRunning bool

	startHidden     bool
	windowHidden    bool
	quitting        bool
	lastErrorNotify time.Time
}

// NewApp creates the binding object.
func NewApp() *App {
	a := &App{progress: progress.New()}
	a.restoreResults()
	return a
}

func (a *App) restoreResults() {
	file, err := results.Load(results.DefaultPath())
	if err != nil {
		return
	}
	// Do not resurrect results from a different subscription/environment.
	if file.ConfigPath != "" {
		if s, err := settings.Load(""); err == nil && s.ConfigPath != "" && s.ConfigPath != file.ConfigPath {
			return
		}
	}
	a.lastResults = file.Results
	a.lastConfigPath = file.ConfigPath
}

func (a *App) persistResults(configPath, testURL string, rounds int) {
	if len(a.lastResults) == 0 {
		return
	}
	if testURL == "" {
		testURL = defaultTestURL
	}
	_ = results.Save(results.DefaultPath(), results.File{
		GeneratedAt: time.Now().Format(time.RFC3339),
		URL:         testURL,
		Rounds:      rounds,
		ConfigPath:  configPath,
		Results:     append([]score.Result(nil), a.lastResults...),
	})
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.startHidden {
		a.mu.Lock()
		a.windowHidden = true
		a.mu.Unlock()
		runtime.WindowHide(ctx)
	}
}

// beforeClose keeps the daemon alive in the tray unless the user explicitly
// quits from the tray or disabled the setting.
func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.Lock()
	quitting := a.quitting
	a.mu.Unlock()
	if quitting {
		return false
	}
	s, err := settings.Load("")
	if err != nil || !s.CloseToTray {
		return false
	}
	a.mu.Lock()
	a.windowHidden = true
	a.mu.Unlock()
	runtime.WindowHide(ctx)
	return true
}

func (a *App) showWindow() {
	a.mu.Lock()
	a.windowHidden = false
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.WindowUnminimise(a.ctx)
		runtime.WindowShow(a.ctx)
	}
}

func (a *App) quitApp() {
	a.mu.Lock()
	a.quitting = true
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

// Autostart reports whether NodePulse is enabled at login.
func (a *App) Autostart() bool {
	return autostart.Enabled()
}

// SetAutostart enables or disables the login item.
func (a *App) SetAutostart(enabled bool) error {
	if err := autostart.Set(enabled); err != nil {
		return err
	}
	setTrayAutostart(enabled)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "nodepulse:autostart", enabled)
	}
	return nil
}

// Version returns the GUI version.
func (a *App) Version() string {
	return version.Version
}

// ClearResults drops the last ranked results, used when switching environments
// so stale nodes cannot be selected or exported under a new subscription.
func (a *App) ClearResults() {
	a.mu.Lock()
	a.lastResults = nil
	a.lastConfigPath = ""
	a.mu.Unlock()
	_ = results.Remove(results.DefaultPath())
}

// Results returns the last ranked results held by this GUI process.
func (a *App) Results() []score.Result {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]score.Result(nil), a.lastResults...)
}

// History returns the most recent daemon events so the UI can show them after
// a restart instead of starting from an empty list.
func (a *App) History() ([]store.Event, error) {
	st, err := store.New(store.DefaultPath())
	if err != nil {
		return nil, err
	}
	return st.Recent(50)
}

// BootstrapResult is what the GUI needs on first paint.
type BootstrapResult struct {
	Controllers   []discover.Controller `json:"controllers"`
	Mihomo        string                `json:"mihomo,omitempty"`
	Subscriptions []string              `json:"subscriptions"`
	Settings      settings.Settings     `json:"settings"`
	Autostart     bool                  `json:"autostart"`
	MihomoError   string                `json:"mihomo_error,omitempty"`
}

// Bootstrap auto-detects the environment and returns the saved settings.
func (a *App) Bootstrap() (BootstrapResult, error) {
	s, err := settings.Load("")
	if err != nil {
		return BootstrapResult{}, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	controllers := discover.Controllers(ctx)
	if s.API == "" && len(controllers) > 0 {
		s.API = controllers[0].URL
	}
	mihomo, mihomoErr := discover.MihomoBinary()
	if s.MihomoPath == "" && mihomoErr == nil {
		s.MihomoPath = mihomo
	}
	if s.ConfigPath == "" {
		if files := discover.SubscriptionFiles(); len(files) > 0 {
			s.ConfigPath = files[0]
		}
	}
	s.SyncActiveProfile()
	out := BootstrapResult{
		Controllers:   controllers,
		Mihomo:        mihomo,
		Subscriptions: discover.SubscriptionFiles(),
		Settings:      s,
		Autostart:     autostart.Enabled(),
	}
	if mihomoErr != nil {
		out.MihomoError = mihomoErr.Error()
	}
	return out, nil
}

// Settings returns the saved settings.
func (a *App) Settings() (settings.Settings, error) {
	return settings.Load("")
}

// Health checks the current environment without modifying it.
func (a *App) Health(api, secret, group, configPath, mihomoPath, testURL string) health.Report {
	s := settings.Defaults()
	s.API = api
	s.Secret = secret
	s.Group = group
	s.ConfigPath = configPath
	s.MihomoPath = mihomoPath
	if testURL != "" {
		s.URL = testURL
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	return health.Run(ctx, s)
}

// SaveSettings persists the settings.
func (a *App) SaveSettings(s settings.Settings) error {
	return settings.Save("", s)
}

// RestoreSettings replaces settings from a backup. A running daemon is
// stopped first so the reloaded UI cannot race an old environment.
func (a *App) RestoreSettings(s settings.Settings) error {
	a.mu.Lock()
	running := a.testRunning
	a.mu.Unlock()
	if running {
		return fmt.Errorf("测速进行中，完成后再恢复设置")
	}
	old, _ := settings.Load("")
	_ = a.StopDaemon()
	if old.ConfigPath != s.ConfigPath {
		a.ClearResults()
	}
	return settings.Save("", s)
}

// ExportSettingsBackup writes a portable copy of settings, including profiles.
// The file contains the controller secret, so it stays 0600.
func (a *App) ExportSettingsBackup() (string, error) {
	s, err := settings.Load("")
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	out := fmt.Sprintf("nodepulse-settings-%s.json", time.Now().Format("20060102-150405"))
	if a.ctx != nil {
		chosen, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "导出 NodePulse 设置备份",
			DefaultFilename: out,
			Filters: []runtime.FileFilter{
				{DisplayName: "JSON (*.json)", Pattern: "*.json"},
			},
		})
		if err != nil {
			return "", err
		}
		if chosen == "" {
			return "", fmt.Errorf("已取消导出")
		}
		out = chosen
	}
	_, _ = backup.Snapshot(out, backup.DefaultKeep)
	if err := os.WriteFile(out, data, 0o600); err != nil {
		return "", err
	}
	_ = os.Chmod(out, 0o600)
	return out, nil
}

// Detect re-runs environment detection.
func (a *App) Detect() (BootstrapResult, error) {
	return a.Bootstrap()
}

// Progress returns the current manual test/optimization progress.
func (a *App) Progress() progress.Snapshot {
	return a.progress.Snapshot()
}

// CancelTest cancels the running manual test/optimization.
func (a *App) CancelTest() bool {
	a.mu.Lock()
	cancel := a.testCancel
	a.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (a *App) beginTest(kind string) (context.Context, func(error), error) {
	a.mu.Lock()
	if a.testRunning {
		a.mu.Unlock()
		return nil, nil, fmt.Errorf("已有测速任务在运行")
	}
	base := a.ctx
	if base == nil {
		base = context.Background()
	}
	ctx, cancel := context.WithCancel(base)
	a.testRunning = true
	a.testCancel = cancel
	a.mu.Unlock()
	a.progress.Begin(kind, cancel)
	finish := func(err error) {
		a.mu.Lock()
		a.testRunning = false
		a.testCancel = nil
		a.mu.Unlock()
		a.progress.Finish(err)
	}
	return ctx, finish, nil
}

// OneClick tests nodes and optionally switches to the best one.
func (a *App) OneClick(api, secret, group, configPath, mihomoPath, testURL string, rounds, concurrency, timeoutMs int, doSwitch bool) (res *optimize.Result, err error) {
	ctx, finish, err := a.beginTest("oneclick")
	if err != nil {
		return nil, err
	}
	defer func() { finish(err) }()

	req := optimize.Request{
		API:         api,
		Secret:      secret,
		Group:       group,
		ConfigPath:  configPath,
		MihomoPath:  mihomoPath,
		URL:         testURL,
		Rounds:      rounds,
		Concurrency: concurrency,
		TimeoutMs:   timeoutMs,
		Switch:      doSwitch,
		OnProgress: func(done, total int, result score.Result) {
			a.progress.Update(done, total, result.Name)
		},
	}
	res, err = optimize.Run(ctx, req)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.lastResults = res.Results
	a.lastConfigPath = configPath
	a.mu.Unlock()
	if len(res.Results) > 0 {
		a.persistResults(configPath, testURL, rounds)
	}

	// Persist whatever worked so the next launch is one click away.
	s, _ := settings.Load("")
	if api != "" {
		s.API = api
		s.Secret = secret
	}
	if group != "" {
		s.Group = group
	}
	if configPath != "" {
		s.ConfigPath = configPath
	}
	if mihomoPath != "" {
		s.MihomoPath = mihomoPath
	}
	if rounds > 0 {
		s.Rounds = rounds
	}
	_ = settings.Save("", s)
	return res, nil
}

// ImportSubscription downloads a subscription URL and saves it as YAML.
func (a *App) ImportSubscription(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("请填写订阅 URL")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 50*time.Second)
	defer cancel()
	data, err := netfetch.Download(ctx, rawURL)
	if err != nil {
		return "", fmt.Errorf("下载订阅失败: %w", err)
	}
	if _, err := subscription.LoadReader(strings.NewReader(string(data))); err != nil {
		return "", fmt.Errorf("订阅内容无法解析: %w", err)
	}

	dir := filepath.Join(filepath.Dir(settings.DefaultPath()), "subscriptions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	name := fmt.Sprintf("sub-%s.yaml", time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	s, _ := settings.Load("")
	s.ConfigPath = path
	_ = settings.Save("", s)
	return path, nil
}

// StartDaemon starts the background auto-switch loop.
func (a *App) StartDaemon(api, secret, group string, intervalSec, rounds int, margin float64, confirm int, minSuccess float64) error {
	if api == "" {
		return fmt.Errorf("自动优化需要先连接 Clash 控制口")
	}
	a.mu.Lock()
	if a.daemonRunning {
		a.mu.Unlock()
		return fmt.Errorf("后台自动优化已经在运行")
	}
	a.mu.Unlock()

	daemonLock, err := lock.Acquire(lock.DefaultPath())
	if err != nil {
		return err
	}

	a.mu.Lock()
	if a.daemonRunning {
		a.mu.Unlock()
		_ = daemonLock.Release()
		return fmt.Errorf("后台自动优化已经在运行")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.daemonSeq++
	id := a.daemonSeq
	done := make(chan struct{})
	a.daemonCancel = cancel
	a.daemonRunning = true
	a.daemonDone = done
	a.daemonLock = daemonLock
	a.mu.Unlock()

	st, err := store.New(store.DefaultPath())
	if err != nil {
		a.mu.Lock()
		if a.daemonSeq == id {
			a.daemonRunning = false
			a.daemonCancel = nil
			a.daemonDone = nil
		}
		close(done)
		a.mu.Unlock()
		cancel()
		_ = daemonLock.Release()
		return err
	}
	d := daemon.New(clash.New(api, secret), daemon.Options{
		Group:       group,
		Interval:    time.Duration(intervalSec) * time.Second,
		Rounds:      rounds,
		Concurrency: 12,
		TestURL:     defaultTestURL,
		Timeout:     5 * time.Second,
		Margin:      margin,
		Confirm:     confirm,
		MinSuccess:  minSuccess,
		Store:       st,
		OnEvent: func(e store.Event) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "nodepulse:daemon", e)
			}
			a.notifyEvent(e)
		},
	})
	go func() {
		_ = d.Run(ctx)
		a.mu.Lock()
		if a.daemonSeq == id {
			a.daemonRunning = false
			a.daemonCancel = nil
			a.daemonDone = nil
			a.daemonLock = nil
		}
		a.mu.Unlock()
		_ = daemonLock.Release()
		close(done)
	}()
	return nil
}

// StopDaemon stops the background loop and waits briefly for the run loop to
// exit so a subsequent start cannot overlap with the old one.
func (a *App) StopDaemon() error {
	a.mu.Lock()
	cancel := a.daemonCancel
	done := a.daemonDone
	daemonLock := a.daemonLock
	a.daemonCancel = nil
	a.daemonRunning = false
	a.daemonDone = nil
	a.daemonLock = nil
	a.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}
	_ = daemonLock.Release()
	return nil
}

// DaemonStatus reports whether the background loop is running.
func (a *App) DaemonStatus() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.daemonRunning
}

// GroupView is a controller group shown in the UI.
type GroupView struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Now     string `json:"now"`
	Members int    `json:"members"`
}

// Status returns groups known to the Clash controller.
func (a *App) Status(api, secret string) ([]GroupView, error) {
	if api == "" {
		return nil, fmt.Errorf("请先填写 Clash 控制口（如 http://127.0.0.1:9090）")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	proxies, err := clash.New(api, secret).Proxies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]GroupView, 0)
	for name, info := range proxies {
		if !isGroupType(info.Type) {
			continue
		}
		out = append(out, GroupView{Name: name, Type: info.Type, Now: info.Now, Members: len(info.All)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// TestAllSubscriptions merges locally discoverable profiles and tests every
// node with one sidecar. It only recommends; it never switches an inactive
// client profile.
func (a *App) TestAllSubscriptions(api, secret, mihomoPath, testURL string, rounds, concurrency, timeoutMs int) (res *optimize.Result, err error) {
	ctx, finish, err := a.beginTest("all")
	if err != nil {
		return nil, err
	}
	defer func() { finish(err) }()

	res, err = optimize.Run(ctx, optimize.Request{
		API:              api,
		Secret:           secret,
		MihomoPath:       mihomoPath,
		URL:              testURL,
		Rounds:           rounds,
		Concurrency:      concurrency,
		TimeoutMs:        timeoutMs,
		AllSubscriptions: true,
		OnProgress: func(done, total int, result score.Result) {
			a.progress.Update(done, total, result.Name)
		},
	})
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.lastResults = res.Results
	a.lastConfigPath = ""
	a.mu.Unlock()
	if len(res.Results) > 0 {
		a.persistResults("", testURL, rounds)
	}
	return res, nil
}

// DownloadMode tests the top candidates against a target file and switches
// once, before a download starts. It never changes nodes during the download.
func (a *App) DownloadMode(api, secret, group, configPath, mihomoPath, targetURL string, rounds, concurrency, timeoutMs int) (res *optimize.Result, err error) {
	ctx, finish, err := a.beginTest("download")
	if err != nil {
		return nil, err
	}
	defer func() { finish(err) }()

	res, err = optimize.Run(ctx, optimize.Request{
		API:          api,
		Secret:       secret,
		Group:        group,
		ConfigPath:   configPath,
		MihomoPath:   mihomoPath,
		TargetURL:    targetURL,
		Rounds:       rounds,
		Concurrency:  concurrency,
		TimeoutMs:    timeoutMs,
		DownloadMode: true,
		ProbeBytes:   3 << 20,
		ProbeSeconds: 5,
		ProbeTopK:    5,
		OnProgress: func(done, total int, result score.Result) {
			a.progress.Update(done, total, result.Name)
		},
	})
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.lastResults = res.Results
	a.lastConfigPath = configPath
	a.mu.Unlock()
	if len(res.Results) > 0 {
		a.persistResults(configPath, "", rounds)
	}
	return res, nil
}

// Test runs a stability test via controller or sidecar.
func (a *App) Test(api, secret, configPath, mihomoPath, testURL string, rounds, concurrency, timeoutMs int) (ranked []score.Result, err error) {
	ctx, finish, err := a.beginTest("test")
	if err != nil {
		return nil, err
	}
	defer func() { finish(err) }()

	if rounds <= 0 {
		rounds = 5
	}
	if concurrency <= 0 {
		concurrency = 8
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	if testURL == "" {
		testURL = defaultTestURL
	}

	var delayer tester.Delayer
	var cleanup func()
	if api != "" {
		delayer = clash.New(api, secret)
	} else {
		if configPath == "" || mihomoPath == "" {
			return nil, fmt.Errorf("未填写控制口时，需要同时提供订阅配置和 mihomo 路径")
		}
		cfg, err := subscription.Load(configPath)
		if err != nil {
			return nil, err
		}
		sc, err := sidecar.Start(ctx, cfg, mihomoPath)
		if err != nil {
			return nil, err
		}
		delayer = clash.New(sc.BaseURL, sc.Secret)
		cleanup = func() { _ = sc.Close() }
	}
	if cleanup != nil {
		defer cleanup()
	}

	var nodes []config.Proxy
	if configPath != "" {
		cfg, err := subscription.Load(configPath)
		if err != nil {
			return nil, err
		}
		nodes = cfg.RealProxies()
	} else {
		listCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		proxies, err := clash.New(api, secret).Proxies(listCtx)
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

	filtered := nodes[:0]
	for _, n := range nodes {
		if !config.IsInfoNode(n.Name) {
			filtered = append(filtered, n)
		}
	}
	nodes = filtered
	if len(nodes) == 0 {
		return nil, fmt.Errorf("没有可测试的节点")
	}

	t := tester.New(delayer, tester.Options{
		Rounds:      rounds,
		Concurrency: concurrency,
		URL:         testURL,
		Timeout:     time.Duration(timeoutMs) * time.Millisecond,
	})
	a.progress.Update(0, len(nodes), "")
	t.OnNode(func(done, total int, result score.Result) {
		a.progress.Update(done, total, result.Name)
	})
	results := t.Test(ctx, nodes)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	ranked = score.Rank(results)
	a.mu.Lock()
	a.lastResults = ranked
	a.lastConfigPath = configPath
	a.mu.Unlock()
	a.persistResults(configPath, testURL, rounds)
	return ranked, nil
}

// Switch selects a node in a Selector group.
func (a *App) Switch(api, secret, group, node string) error {
	if api == "" || group == "" || node == "" {
		return fmt.Errorf("控制口、组和节点都不能为空")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	return clash.New(api, secret).Switch(ctx, group, node)
}

// PreviewExport reports what an export would change without writing anything.
func (a *App) PreviewExport(minSuccess float64, minimal bool) (*export.Impact, error) {
	cfg, results, err := a.exportInputs()
	if err != nil {
		return nil, err
	}
	if minSuccess <= 0 {
		minSuccess = 0.8
	}
	return export.Preview(cfg, results, export.Options{
		MinSuccess: minSuccess,
		TestURL:    defaultTestURL,
		Preserve:   !minimal,
	})
}

// Export writes a filtered Clash config from the last test.
func (a *App) Export(out string, minSuccess float64, minimal bool) (string, error) {
	cfg, results, err := a.exportInputs()
	if err != nil {
		return "", err
	}
	if out == "" {
		out = "stable.yaml"
	}
	if minSuccess <= 0 {
		minSuccess = 0.8
	}
	sel, err := export.Stable(cfg, results, export.Options{
		MinSuccess: minSuccess,
		TestURL:    defaultTestURL,
		Preserve:   !minimal,
	})
	if err != nil {
		return "", err
	}
	if a.ctx != nil {
		chosen, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "导出稳定配置",
			DefaultFilename: filepath.Base(out),
			Filters: []runtime.FileFilter{
				{DisplayName: "Clash YAML (*.yaml)", Pattern: "*.yaml"},
			},
		})
		if err != nil {
			return "", err
		}
		if chosen == "" {
			return "", fmt.Errorf("已取消导出")
		}
		out = chosen
	}
	_, _ = backup.Snapshot(out, backup.DefaultKeep)
	if err := sel.Config.Write(out); err != nil {
		return "", err
	}
	mode := "保留原规则/分组"
	if !sel.Preserved {
		mode = "最小配置"
	}
	return fmt.Sprintf("已写入 %s（%s）：保留 %d 个，排除 %d 个", out, mode, len(sel.Included), len(sel.Excluded)), nil
}

func (a *App) exportInputs() (*config.Config, []score.Result, error) {
	a.mu.Lock()
	results := append([]score.Result(nil), a.lastResults...)
	configPath := a.lastConfigPath
	a.mu.Unlock()
	if len(results) == 0 {
		return nil, nil, fmt.Errorf("请先测试一次")
	}
	if configPath == "" {
		return nil, nil, fmt.Errorf("当前结果不是从订阅配置测试的，无法导出")
	}
	cfg, err := subscription.Load(configPath)
	if err != nil {
		return nil, nil, err
	}
	return cfg, results, nil
}

// ChooseConfig opens a file dialog for the subscription config.
func (a *App) ChooseConfig() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 Clash / mihomo 订阅配置",
		Filters: []runtime.FileFilter{
			{DisplayName: "YAML", Pattern: "*.yaml;*.yml"},
		},
	})
}

// ChooseMihomo opens a file dialog for the mihomo binary.
func (a *App) ChooseMihomo() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 mihomo 可执行文件",
	})
}

func isGroupType(t string) bool {
	switch strings.ToLower(t) {
	case "selector", "urltest", "fallback", "loadbalance":
		return true
	}
	return false
}

func isTestable(t string) bool {
	switch strings.ToLower(t) {
	case "selector", "urltest", "fallback", "loadbalance", "direct", "reject", "rejectdrop", "compatible", "pass", "dns":
		return false
	}
	return true
}
