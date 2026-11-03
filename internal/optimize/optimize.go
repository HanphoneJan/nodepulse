package optimize

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/discover"
	"github.com/HanphoneJan/nodepulse/internal/probe"
	"github.com/HanphoneJan/nodepulse/internal/probecache"
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
	DownloadMode     bool
	TargetURL        string
	ProbeBytes       int64
	ProbeSeconds     int
	ProbeTopK        int
	OnProgress       func(done, total int, result score.Result)
}

// DownloadProbe is one candidate's target-site download measurement.
type DownloadProbe struct {
	probe.Result
	DownloadScore float64 `json:"download_score"`
	BaseScore     float64 `json:"base_score"`
}

// DownloadInfo is the download-mode report.
type DownloadInfo struct {
	TargetHost string          `json:"target_host"`
	TargetPath string          `json:"target_path"`
	TopK       int             `json:"top_k"`
	CacheHits  int             `json:"cache_hits"`
	Excluded   int             `json:"excluded"`
	Fallback   bool            `json:"fallback,omitempty"`
	Probes     []DownloadProbe `json:"probes"`
	Best       DownloadProbe   `json:"best"`
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
	Download          *DownloadInfo  `json:"download,omitempty"`
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
	if req.DownloadMode {
		return runDownload(ctx, req)
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

func runDownload(ctx context.Context, req Request) (*Result, error) {
	target, err := probe.ParseTarget(req.TargetURL)
	if err != nil {
		return nil, err
	}
	if req.ProbeBytes <= 0 {
		req.ProbeBytes = 3 << 20
	}
	if req.ProbeSeconds <= 0 {
		req.ProbeSeconds = 5
	}
	if req.ProbeTopK <= 0 {
		req.ProbeTopK = 5
	}
	if req.ConfigPath == "" {
		return nil, fmt.Errorf("下载模式需要本地订阅配置来构建测速 sidecar；请先在设置里选择订阅文件")
	}
	cfg, err := subscription.Load(req.ConfigPath)
	if err != nil {
		return nil, err
	}

	// First run the normal latency/stability test. It gives the candidate pool
	// and keeps browser-oriented scoring unchanged.
	baseReq := req
	baseReq.DownloadMode = false
	baseReq.TargetURL = ""
	baseReq.Switch = false
	base, err := Run(ctx, baseReq)
	if err != nil {
		return nil, err
	}
	if base.Controller == "" || base.Group == "" {
		return nil, fmt.Errorf("下载模式需要连接客户端并选定一个可管理的 Selector 组")
	}
	baseByName := make(map[string]score.Result, len(base.Results))
	for _, result := range base.Results {
		baseByName[result.Name] = result
	}
	candidateLimit := req.ProbeTopK * 2
	candidates := make([]score.Result, 0, candidateLimit)
	for _, result := range base.Results {
		if result.Success == 0 || result.SuccessRate() < selector.DefaultMinSuccess {
			continue
		}
		candidates = append(candidates, result)
		if len(candidates) >= candidateLimit {
			break
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("没有达到最低成功率的候选节点，无法进行下载模式")
	}
	proxies := make([]config.Proxy, 0, len(candidates))
	for _, candidate := range candidates {
		proxy, ok := cfg.FindProxy(candidate.Name)
		if ok {
			proxies = append(proxies, proxy)
		}
	}
	if len(proxies) == 0 {
		return nil, fmt.Errorf("控制口节点在本地订阅中找不到对应定义，无法做真实吞吐探测")
	}

	cache, _ := probecache.Open(probecache.DefaultPath())
	missing := cache == nil
	if cache != nil {
		for _, proxy := range proxies {
			key := probecache.Key(proxy.Name, target.Host, target.Path)
			if _, ok := cache.Get(key); !ok {
				missing = true
				break
			}
		}
	}
	var sc *sidecar.Sidecar
	if missing {
		mihomoPath := req.MihomoPath
		if mihomoPath == "" {
			if path, err := discover.MihomoBinary(); err == nil {
				mihomoPath = path
			}
		}
		if mihomoPath == "" {
			return nil, fmt.Errorf("下载模式需要 mihomo 作为临时探测内核；请在设置里填写 mihomo 路径")
		}
		sc, err = sidecar.StartProbe(ctx, proxies, mihomoPath)
		if err != nil {
			return nil, err
		}
		defer sc.Close()
	}

	probes := make([]DownloadProbe, 0, len(proxies))
	cacheHits := 0
	excluded := 0
	probeOne := func(proxy config.Proxy) DownloadProbe {
		key := probecache.Key(proxy.Name, target.Host, target.Path)
		var result probe.Result
		cached := false
		if cache != nil {
			if hit, ok := cache.Get(key); ok {
				result = hit
				cached = true
				cacheHits++
			}
		}
		if !cached {
			if sc == nil {
				result = probe.Result{Node: proxy.Name, Err: "probe sidecar unavailable"}
			} else if err := sc.Select(ctx, proxy.Name); err != nil {
				result = probe.Result{Node: proxy.Name, Err: err.Error()}
			} else {
				time.Sleep(200 * time.Millisecond)
				result, err = probe.Measure(ctx, fmt.Sprintf("127.0.0.1:%d", sc.MixedPort), target, req.ProbeBytes, time.Duration(req.ProbeSeconds)*time.Second)
				if err != nil {
					result.Err = err.Error()
				}
			}
			result.Node = proxy.Name
			if cache != nil {
				cache.Put(key, result)
			}
		}
		result.Node = proxy.Name
		if !result.OK {
			excluded++
		}
		baseScore := baseByName[proxy.Name].Score
		probeResult := DownloadProbe{Result: result, BaseScore: baseScore}
		if result.OK {
			probeResult.DownloadScore = downloadScore(result, baseScore)
		}
		return probeResult
	}
	probeBatch := func(batch []config.Proxy) []DownloadProbe {
		if len(batch) == 0 {
			return nil
		}
		if req.OnProgress != nil {
			req.OnProgress(0, len(batch), score.Result{})
		}
		out := make([]DownloadProbe, 0, len(batch))
		for index, proxy := range batch {
			out = append(out, probeOne(proxy))
			if req.OnProgress != nil {
				req.OnProgress(index+1, len(batch), score.Result{Name: "探测 " + proxy.Name})
			}
		}
		return out
	}
	probed := 0
	firstEnd := req.ProbeTopK
	if firstEnd > len(proxies) {
		firstEnd = len(proxies)
	}
	probes = append(probes, probeBatch(proxies[:firstEnd])...)
	probed = len(probes)
	fallback := false
	anyOK := false
	for _, result := range probes {
		if result.OK {
			anyOK = true
			break
		}
	}
	if !anyOK && len(proxies) > firstEnd {
		fallback = true
		probes = append(probes, probeBatch(proxies[firstEnd:])...)
		probed = len(probes)
	}
	if cache != nil {
		_ = cache.Save()
	}
	sort.SliceStable(probes, func(i, j int) bool {
		if probes[i].OK != probes[j].OK {
			return probes[i].OK
		}
		return probes[i].DownloadScore > probes[j].DownloadScore
	})

	info := &DownloadInfo{
		TargetHost: target.Host,
		TargetPath: target.Path,
		TopK:       probed,
		CacheHits:  cacheHits,
		Excluded:   excluded,
		Fallback:   fallback,
		Probes:     probes,
	}
	if len(probes) > 0 {
		info.Best = probes[0]
	}
	base.Download = info
	base.Scope = "download"
	res := base
	if len(probes) == 0 || !probes[0].OK {
		res.Best = base.Best
		res.ReasonCode = "download-unreachable"
		res.Reason = "所有候选节点都无法访问目标站点（TLS/EOF/超时/HTTP 错误）"
		res.Message = res.Reason
		return res, nil
	}
	best := probes[0]
	bestBase := baseByName[best.Node]
	if bestBase.Name == "" {
		bestBase = score.Result{Name: best.Node, Success: 1, Total: 1, Score: best.BaseScore}
	}
	res.Best = bestBase
	res.ReasonCode = "download-selected"
	summary := fmt.Sprintf("p25 %.0f KiB/s，首字节 %dms", best.P25KiBps, best.TTFBMs)
	if best.Cached {
		summary += "，缓存命中"
	}
	if excluded > 0 {
		summary += fmt.Sprintf("，淘汰 %d 个不可达节点", excluded)
	}
	if fallback {
		summary += "，已补测下一批候选"
	}
	if best.Node == res.Current {
		res.Reason = "当前节点 " + best.Node + " 已是下载模式最优（" + summary + "）"
		res.Message = res.Reason
		return res, nil
	}

	controller := clash.New(res.Controller, req.Secret)
	if err := controller.Switch(ctx, res.Group, best.Node); err != nil {
		res.ReasonCode = "switch-failed"
		res.Reason = fmt.Sprintf("切换到 %s 失败: %v", best.Node, err)
		return res, fmt.Errorf("%s", res.Reason)
	}
	time.Sleep(300 * time.Millisecond)
	now, currentErr := controller.Current(ctx, res.Group)
	verifyErr := currentErr
	if verifyErr == nil && now != best.Node {
		verifyErr = fmt.Errorf("控制器仍显示 %q", now)
	}
	if verifyErr == nil {
		if _, err := controller.Delay(ctx, best.Node, req.URL, time.Duration(req.TimeoutMs)*time.Millisecond); err != nil {
			verifyErr = fmt.Errorf("验证 %s 失败: %w", best.Node, err)
		}
	}
	if verifyErr != nil {
		res.ReasonCode = "verify-failed"
		res.Reason = fmt.Sprintf("下载模式切换验证失败: %v", verifyErr)
		res.RollbackAttempted = res.Current != ""
		if res.Current != "" {
			rollbackErr := controller.Switch(ctx, res.Group, res.Current)
			res.RollbackOK = rollbackErr == nil
			if rollbackErr != nil {
				res.ReasonCode = "rollback-failed"
				res.Reason = fmt.Sprintf("下载模式切换验证失败: %v；回滚也失败: %v", verifyErr, rollbackErr)
			} else {
				res.Reason = fmt.Sprintf("下载模式切换验证失败: %v；已回滚到 %s", verifyErr, res.Current)
			}
		}
		return res, fmt.Errorf("%s", res.Reason)
	}
	res.Switched = true
	res.Verified = true
	res.Current = now
	res.Reason = fmt.Sprintf("下载模式：已选择 %s（%s）", best.Node, summary)
	res.Message = res.Reason
	return res, nil
}

func downloadScore(result probe.Result, base float64) float64 {
	throughput := 0.0
	if result.P25KiBps > 0 {
		throughput = math.Log1p(result.P25KiBps) / math.Log1p(1024*1024)
	}
	throughput = math.Max(0, math.Min(1, throughput))
	stability := 1.0
	if result.MedianKiBps > 0 {
		stability = math.Max(0, math.Min(1, result.P25KiBps/result.MedianKiBps))
	}
	ttfb := 1 / (1 + float64(result.TTFBMs)/1000)
	return 0.60*throughput + 0.15*stability + 0.15*base + 0.10*ttfb
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
