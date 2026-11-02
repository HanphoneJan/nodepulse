package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/export"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/sidecar"
	"github.com/HanphoneJan/nodepulse/internal/subscription"
	"github.com/HanphoneJan/nodepulse/internal/tester"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const defaultTestURL = "https://www.gstatic.com/generate_204"

// App is the Wails binding surface.
type App struct {
	ctx            context.Context
	lastResults    []score.Result
	lastConfigPath string
}

// NewApp creates the binding object.
func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Version returns the GUI version.
func (a *App) Version() string {
	return "0.2.0"
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

// Test runs a stability test, either through an existing controller or a
// temporary mihomo sidecar. Results are ranked and kept for Export.
func (a *App) Test(api, secret, configPath, mihomoPath, testURL string, rounds, concurrency, timeoutMs int) ([]score.Result, error) {
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
		sc, err := sidecar.Start(a.ctx, cfg, mihomoPath)
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
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		proxies, err := clash.New(api, secret).Proxies(ctx)
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
	results := t.Test(a.ctx, nodes)
	ranked := score.Rank(results)
	a.lastResults = ranked
	a.lastConfigPath = configPath
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

// Export writes a filtered Clash config from the last test.
func (a *App) Export(out string, minSuccess float64) (string, error) {
	if len(a.lastResults) == 0 {
		return "", fmt.Errorf("请先测试一次")
	}
	if a.lastConfigPath == "" {
		return "", fmt.Errorf("当前结果不是从订阅配置测试的，无法导出")
	}
	if out == "" {
		out = "stable.yaml"
	}
	if minSuccess <= 0 {
		minSuccess = 0.8
	}
	cfg, err := subscription.Load(a.lastConfigPath)
	if err != nil {
		return "", err
	}
	sel, err := export.Stable(cfg, a.lastResults, export.Options{MinSuccess: minSuccess, TestURL: defaultTestURL})
	if err != nil {
		return "", err
	}
	if err := sel.Config.Write(out); err != nil {
		return "", err
	}
	return fmt.Sprintf("已写入 %s：保留 %d 个，排除 %d 个", out, len(sel.Included), len(sel.Excluded)), nil
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
