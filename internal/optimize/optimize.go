package optimize

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/discover"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/sidecar"
	"github.com/HanphoneJan/nodepulse/internal/subscription"
	"github.com/HanphoneJan/nodepulse/internal/tester"
)

// Request is a one-click optimization request.
type Request struct {
	API         string
	Secret      string
	Group       string
	ConfigPath  string
	MihomoPath  string
	URL         string
	Rounds      int
	Concurrency int
	TimeoutMs   int
	Switch      bool
}

// Result is what the UI shows after a one-click run.
type Result struct {
	Source     string         `json:"source"`
	Controller string         `json:"controller,omitempty"`
	Group      string         `json:"group,omitempty"`
	Current    string         `json:"current,omitempty"`
	Best       score.Result   `json:"best"`
	Results    []score.Result `json:"results"`
	Switched   bool           `json:"switched"`
	Message    string         `json:"message"`
}

// Run executes the one-click flow: resolve a controller (or sidecar), test,
// rank, and optionally select the best node.
func Run(ctx context.Context, req Request) (*Result, error) {
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

	res := &Result{Source: "sidecar"}
	var delayer tester.Delayer
	var controller *clash.Client
	var proxies map[string]clash.ProxyInfo

	api := req.API
	if api == "" {
		var authRequired string
		for _, c := range discover.Controllers(ctx) {
			if c.AuthRequired && req.Secret == "" {
				authRequired = c.URL
				continue
			}
			api = c.URL
			res.Message = "自动检测到客户端: " + c.URL
			break
		}
		if api == "" && authRequired != "" {
			res.Message = fmt.Sprintf("检测到 %s，但需要 secret；请在高级设置里填写", authRequired)
			return nil, fmt.Errorf("检测到 %s，但需要 secret", authRequired)
		}
	}
	if api != "" {
		controller = clash.New(api, req.Secret)
		res.Controller = api
		res.Source = "controller"
		var err error
		proxies, err = controller.Proxies(ctx)
		if err != nil {
			return nil, fmt.Errorf("连接客户端 %s 失败: %w", api, err)
		}
		delayer = controller
	}

	var nodes []config.Proxy
	if req.ConfigPath != "" {
		cfg, err := subscription.Load(req.ConfigPath)
		if err != nil {
			return nil, err
		}
		for _, n := range cfg.RealProxies() {
			if !config.IsInfoNode(n.Name) {
				nodes = append(nodes, n)
			}
		}
	} else if proxies != nil {
		for name, info := range proxies {
			if isTestable(info.Type) {
				nodes = append(nodes, config.Proxy{Name: name, Type: info.Type})
			}
		}
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("没有可测试的节点；请导入订阅或填写控制口")
	}

	if controller == nil {
		mihomoPath := req.MihomoPath
		if mihomoPath == "" {
			if p, err := discover.MihomoBinary(); err == nil {
				mihomoPath = p
			}
		}
		if req.ConfigPath == "" {
			return nil, fmt.Errorf("未检测到客户端，也没有订阅配置；请导入订阅或填写 mihomo 路径")
		}
		if mihomoPath == "" {
			return nil, fmt.Errorf("未检测到客户端，且找不到 mihomo；请在高级设置里填写 mihomo 路径")
		}
		cfg, err := subscription.Load(req.ConfigPath)
		if err != nil {
			return nil, err
		}
		sc, err := sidecar.Start(ctx, cfg, mihomoPath)
		if err != nil {
			return nil, err
		}
		defer sc.Close()
		delayer = clash.New(sc.BaseURL, sc.Secret)
		res.Source = "sidecar"
		res.Message = "使用 mihomo sidecar 测速（未连接客户端，不能自动切换）"
	}

	t := tester.New(delayer, tester.Options{
		Rounds:      req.Rounds,
		Concurrency: req.Concurrency,
		URL:         req.URL,
		Timeout:     time.Duration(req.TimeoutMs) * time.Millisecond,
	})
	results := score.Rank(t.Test(ctx, nodes))
	res.Results = results
	if len(results) == 0 {
		return nil, fmt.Errorf("测试没有返回结果")
	}
	best, ok := firstQualified(results)
	if !ok {
		best = results[0]
	}
	res.Best = best

	if controller == nil {
		if res.Message == "" {
			res.Message = "测速完成；未连接客户端，不能自动切换"
		}
		return res, nil
	}

	group := req.Group
	if group == "" {
		group = detectGroup(proxies)
	}
	if group == "" {
		res.Message = "测速完成；但没有找到 Selector 组，无法自动切换"
		return res, nil
	}
	res.Group = group
	if info, ok := proxies[group]; ok {
		res.Current = info.Now
	}

	if !req.Switch {
		res.Message = fmt.Sprintf("推荐 %s（评分 %.3f）", best.Name, best.Score)
		return res, nil
	}
	if best.Success == 0 {
		res.Message = "没有可用节点，未切换"
		return res, nil
	}
	if best.Name == res.Current {
		res.Message = fmt.Sprintf("当前已是最优节点 %s", best.Name)
		return res, nil
	}

	if err := controller.Switch(ctx, group, best.Name); err != nil {
		return res, fmt.Errorf("切换到 %s 失败: %w", best.Name, err)
	}
	time.Sleep(200 * time.Millisecond)
	now, err := controller.Current(ctx, group)
	if err != nil || now != best.Name {
		if res.Current != "" {
			_ = controller.Switch(ctx, group, res.Current)
		}
		return res, fmt.Errorf("切换验证失败（当前 %q），已尝试回滚", now)
	}
	res.Switched = true
	res.Current = now
	res.Message = fmt.Sprintf("已切换到 %s（评分 %.3f）", best.Name, best.Score)
	return res, nil
}

func firstQualified(results []score.Result) (score.Result, bool) {
	for _, r := range results {
		if r.Success > 0 && r.SuccessRate() >= 0.8 {
			return r, true
		}
	}
	return score.Result{}, false
}

func detectGroup(proxies map[string]clash.ProxyInfo) string {
	preferred := []string{"🚀节点选择", "节点选择", "Proxy", "PROXY", "一分机场", "赔钱机场", "GLOBAL"}
	for _, name := range preferred {
		if info, ok := proxies[name]; ok && strings.EqualFold(info.Type, "Selector") {
			return name
		}
	}
	names := make([]string, 0)
	for name, info := range proxies {
		if strings.EqualFold(info.Type, "Selector") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) > 0 {
		return names[0]
	}
	return ""
}

func isTestable(t string) bool {
	switch strings.ToLower(t) {
	case "selector", "urltest", "fallback", "loadbalance", "direct", "reject", "rejectdrop", "compatible", "pass", "dns":
		return false
	}
	return true
}
