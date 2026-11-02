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
	OnProgress  func(done, total int, result score.Result)
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
