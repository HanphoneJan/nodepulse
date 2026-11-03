package optimize

import (
	"context"
	"fmt"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/discover"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/selector"
	"github.com/HanphoneJan/nodepulse/internal/sidecar"
	"github.com/HanphoneJan/nodepulse/internal/subscription"
	"github.com/HanphoneJan/nodepulse/internal/tester"
)

// Request is a one-click optimization request.
type Request struct {
	API              string
	Secret           string
	Group            string
	ConfigPath       string
	MihomoPath       string
	URL              string
	Rounds           int
	Concurrency      int
	TimeoutMs        int
	Switch           bool
	AllSubscriptions bool
	OnProgress       func(done, total int, result score.Result)
}

// SourceInfo describes one subscription/profile used by a merged test.
type SourceInfo struct {
	Label  string `json:"label"`
	Client string `json:"client,omitempty"`
	Path   string `json:"path,omitempty"`
	Active bool   `json:"active,omitempty"`
	Nodes  int    `json:"nodes"`
}

// Result is what the UI shows after a one-click run.
type Result struct {
	Source            string         `json:"source"`
	Controller        string         `json:"controller,omitempty"`
	Group             string         `json:"group,omitempty"`
	Current           string         `json:"current,omitempty"`
	Best              score.Result   `json:"best"`
	Results           []score.Result `json:"results"`
	Switched          bool           `json:"switched"`
	Message           string         `json:"message"`
	ReasonCode        string         `json:"reason_code,omitempty"`
	Reason            string         `json:"reason,omitempty"`
	Confidence        string         `json:"confidence,omitempty"`
	Candidates        int            `json:"candidates,omitempty"`
	NestedSkipped     int            `json:"nested_skipped,omitempty"`
	Verified          bool           `json:"verified,omitempty"`
	RollbackAttempted bool           `json:"rollback_attempted,omitempty"`
	RollbackOK        bool           `json:"rollback_ok,omitempty"`
	Scope             string         `json:"scope,omitempty"`
	Sources           []SourceInfo   `json:"sources,omitempty"`
}

// Run executes the one-click flow: resolve a controller (or sidecar), test the
// managed group's direct nodes, rank, and optionally select the best node.
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
	if req.AllSubscriptions {
		return runAll(ctx, req)
	}

	res := &Result{Source: "sidecar", Confidence: selector.Confidence(req.Rounds)}
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

	var nodes []config.Proxy
	var meta selector.Meta
	var groupErr error

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

		group, err := selector.ResolveGroup(proxies, req.Group)
		groupErr = err
		if err == nil {
			res.Group = group
			if info, ok := proxies[group]; ok {
				res.Current = info.Now
			}
			nodes, meta, err = selector.Candidates(proxies, group)
			if err != nil {
				groupErr = err
				nodes = selector.AllTestable(proxies)
				res.Message = "管理组不可用，仅测试全部节点: " + err.Error()
			}
		} else {
			nodes = selector.AllTestable(proxies)
			res.Message = err.Error() + "；仅测速不切换"
		}
	} else {
		// Sidecar path: the subscription config is the node source.
		if req.ConfigPath == "" {
			return nil, fmt.Errorf("未检测到客户端，也没有订阅配置；请导入订阅或填写 mihomo 路径")
		}
		cfg, err := subscription.Load(req.ConfigPath)
		if err != nil {
			return nil, err
		}
		for _, n := range cfg.RealProxies() {
			if !config.IsInfoNode(n.Name) {
				nodes = append(nodes, n)
			}
		}
		res.Message = "使用 mihomo sidecar 测速（未连接客户端，不能自动切换）"
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("没有可测试的节点；请导入订阅或填写控制口")
	}
	res.Candidates = len(nodes)
	res.NestedSkipped = meta.NestedSkipped

	if controller == nil {
		mihomoPath := req.MihomoPath
		if mihomoPath == "" {
			if p, err := discover.MihomoBinary(); err == nil {
				mihomoPath = p
			}
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
	}

	t := tester.New(delayer, tester.Options{
		Rounds:      req.Rounds,
		Concurrency: req.Concurrency,
		URL:         req.URL,
		Timeout:     time.Duration(req.TimeoutMs) * time.Millisecond,
	})
	if req.OnProgress != nil {
		req.OnProgress(0, len(nodes), score.Result{})
		t.OnNode(req.OnProgress)
	}
	tested := t.Test(ctx, nodes)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	results := score.Rank(tested)
	res.Results = results
	if len(results) == 0 {
		return nil, fmt.Errorf("测试没有返回结果")
	}
	best, bestOK := selector.FirstQualified(results, selector.DefaultMinSuccess)
	if !bestOK {
		best = results[0]
	}
	res.Best = best

	if !bestOK || best.Success == 0 {
		res.ReasonCode = "no-qualified"
		res.Reason = "没有节点达到最低成功率"
		if controller != nil {
			base := selector.Baseline(ctx, controller, proxies, req.URL, time.Duration(req.TimeoutMs)*time.Millisecond)
			if base.Available && !base.OK {
				res.ReasonCode = "baseline-failed"
				res.Reason = fmt.Sprintf("本地 DIRECT 基线失败，可能是网络或测试 URL 问题: %v", base.Err)
			} else if best.LastError != "" {
				res.Reason = "没有节点达标；最近错误: " + best.LastError
			}
		}
		res.Message = res.Reason
		if controller == nil {
			res.Message = "测速完成；" + res.Reason
		}
		return res, nil
	}

	if controller == nil {
		res.Message = "测速完成；未连接客户端，不能自动切换"
		return res, nil
	}

	if groupErr != nil || res.Group == "" {
		res.ReasonCode = "group-error"
		res.Reason = res.Message
		if res.Message == "" {
			res.Message = "没有可管理的 Selector 组，仅测速不切换"
		}
		return res, nil
	}
	if info, ok := proxies[res.Group]; ok {
		res.Current = info.Now
	}

	if !req.Switch {
		res.ReasonCode = "recommendation"
		res.Reason = fmt.Sprintf("推荐 %s（评分 %.3f）", best.Name, best.Score)
		res.Message = res.Reason
		return res, nil
	}
	if best.Name == res.Current {
		res.ReasonCode = "already-best"
		res.Reason = fmt.Sprintf("当前已是最优节点 %s", best.Name)
		res.Message = res.Reason
		return res, nil
	}

	if err := controller.Switch(ctx, res.Group, best.Name); err != nil {
		res.ReasonCode = "switch-failed"
		res.Reason = fmt.Sprintf("切换到 %s 失败: %v", best.Name, err)
		return res, fmt.Errorf("%s", res.Reason)
	}
	time.Sleep(300 * time.Millisecond)
	now, currentErr := controller.Current(ctx, res.Group)
	verifyErr := currentErr
	if verifyErr == nil && now != best.Name {
		verifyErr = fmt.Errorf("控制器仍显示 %q", now)
	}
	if verifyErr == nil {
		if _, err := controller.Delay(ctx, best.Name, req.URL, time.Duration(req.TimeoutMs)*time.Millisecond); err != nil {
			verifyErr = fmt.Errorf("验证 %s 失败: %w", best.Name, err)
		}
	}
	if verifyErr != nil {
		res.ReasonCode = "verify-failed"
		res.Reason = fmt.Sprintf("切换验证失败: %v", verifyErr)
		res.RollbackAttempted = res.Current != ""
		if res.Current != "" {
			rollbackErr := controller.Switch(ctx, res.Group, res.Current)
			res.RollbackOK = rollbackErr == nil
			if rollbackErr != nil {
				res.ReasonCode = "rollback-failed"
				res.Reason = fmt.Sprintf("切换验证失败: %v；回滚也失败: %v", verifyErr, rollbackErr)
			} else {
				res.Reason = fmt.Sprintf("切换验证失败: %v；已回滚到 %s", verifyErr, res.Current)
			}
		}
		return res, fmt.Errorf("%s", res.Reason)
	}

	res.Switched = true
	res.Verified = true
	res.ReasonCode = "switched"
	res.Current = now
	res.Reason = fmt.Sprintf("已切换到 %s（评分 %.3f，%s置信）", best.Name, best.Score, confidenceLabel(res.Confidence))
	res.Message = res.Reason
	return res, nil
}

func runAll(ctx context.Context, req Request) (*Result, error) {
	res := &Result{
		Source:     "profiles",
		Scope:      "all",
		Confidence: selector.Confidence(req.Rounds),
		Message:    "",
	}
	var nodes []config.Proxy
	sources := make(map[string]string)
	seen := make(map[string]string)
	profiles := discover.Profiles()
	for _, profile := range profiles {
		cfg, err := subscription.Load(profile.Path)
		if err != nil {
			continue
		}
		real := cfg.RealProxies()
		if len(real) == 0 {
			continue
		}
		label := profile.Label
		if label == "" {
			label = profile.Path
		}
		if profile.Client != "" {
			label = profile.Client + " · " + label
		}
		info := SourceInfo{Label: label, Client: profile.Client, Path: profile.Path, Active: profile.Active}
		for _, node := range real {
			if config.IsInfoNode(node.Name) {
				continue
			}
			name := node.Name
			if owner, ok := seen[name]; ok {
				if owner == label {
					continue
				}
				name = fmt.Sprintf("%s · %s", node.Name, label)
				for i := 2; ; i++ {
					if _, exists := seen[name]; !exists {
						break
					}
					name = fmt.Sprintf("%s · %s #%d", node.Name, label, i)
				}
			}
			seen[name] = label
			sources[name] = label
			node.Name = name
			nodes = append(nodes, node)
			info.Nodes++
		}
		if info.Nodes > 0 {
			res.Sources = append(res.Sources, info)
		}
	}

	var delayer tester.Delayer
	var cleanup func()
	if len(nodes) > 0 {
		mihomoPath := req.MihomoPath
		if mihomoPath == "" {
			if path, err := discover.MihomoBinary(); err == nil {
				mihomoPath = path
			}
		}
		if mihomoPath == "" {
			return nil, fmt.Errorf("联合测速需要 mihomo 作为临时测速内核；请在设置里填写 mihomo 路径")
		}
		merged := &config.Config{Proxies: nodes}
		sc, err := sidecar.Start(ctx, merged, mihomoPath)
		if err != nil {
			return nil, err
		}
		delayer = clash.New(sc.BaseURL, sc.Secret)
		cleanup = func() { _ = sc.Close() }
	} else {
		// No local profiles: fall back to everything the active controller
		// exposes. This is still useful for clients that only ship one config.
		api := req.API
		if api == "" {
			for _, controller := range discover.Controllers(ctx) {
				if controller.AuthRequired && req.Secret == "" {
					continue
				}
				api = controller.URL
				break
			}
		}
		if api == "" {
			return nil, fmt.Errorf("没有找到本地订阅 profile，也没有可连接的控制口")
		}
		controller := clash.New(api, req.Secret)
		proxies, err := controller.Proxies(ctx)
		if err != nil {
			return nil, fmt.Errorf("读取控制口失败: %w", err)
		}
		for _, node := range selector.AllTestable(proxies) {
			nodes = append(nodes, node)
		}
		if len(nodes) == 0 {
			return nil, fmt.Errorf("控制口没有可测试节点")
		}
		res.Source = "controller"
		res.Message = "控制口只暴露当前激活订阅"
		res.Sources = append(res.Sources, SourceInfo{Label: "客户端当前配置", Client: "controller", Nodes: len(nodes), Active: true})
		delayer = controller
	}
	if cleanup != nil {
		defer cleanup()
	}

	if req.OnProgress != nil {
		req.OnProgress(0, len(nodes), score.Result{})
	}
	t := tester.New(delayer, tester.Options{
		Rounds:      req.Rounds,
		Concurrency: req.Concurrency,
		URL:         req.URL,
		Timeout:     time.Duration(req.TimeoutMs) * time.Millisecond,
	})
	if req.OnProgress != nil {
		t.OnNode(req.OnProgress)
	}
	tested := t.Test(ctx, nodes)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	for i := range tested {
		if source := sources[tested[i].Name]; source != "" {
			tested[i].Source = source
		}
	}
	results := score.Rank(tested)
	if len(results) == 0 {
		return nil, fmt.Errorf("联合测速没有返回结果")
	}
	best, ok := selector.FirstQualified(results, selector.DefaultMinSuccess)
	if !ok {
		best = results[0]
	}
	res.Best = best
	res.Results = results
	res.Candidates = len(results)
	if res.Message == "" {
		res.Message = fmt.Sprintf("已测速 %d 个订阅 / %d 个节点；跨订阅只推荐，未自动切换", len(res.Sources), len(results))
	} else {
		res.Message = fmt.Sprintf("%s；已测速 %d 个节点", res.Message, len(results))
	}
	res.ReasonCode = "all-recommendation"
	res.Reason = res.Message
	return res, nil
}

func confidenceLabel(confidence string) string {
	switch confidence {
	case "high":
		return "高"
	case "medium":
		return "中"
	default:
		return "低"
	}
}
