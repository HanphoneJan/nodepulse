// Package selector contains the safe group resolution, candidate discovery and
// switch decision used by both the background daemon and one-click optimize.
package selector

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/score"
)

const (
	GlobalGroup       = "GLOBAL"
	DefaultMargin     = 0.2
	DefaultConfirm    = 2
	DefaultMinSuccess = 0.8
)

// Meta describes how candidates were discovered.
type Meta struct {
	Total         int
	Direct        int
	NestedSkipped int
	Skipped       int
}

// IsSelector reports whether a proxy entry is a Selector group.
func IsSelector(info clash.ProxyInfo) bool {
	return strings.EqualFold(info.Type, "Selector")
}

// IsGroupType reports whether a proxy entry is a group rather than a node.
func IsGroupType(t string) bool {
	switch strings.ToLower(t) {
	case "selector", "select", "urltest", "url-test", "fallback", "loadbalance", "load-balance", "relay":
		return true
	}
	return false
}

// IsTestable reports whether a proxy entry can be delayed directly.
func IsTestable(t string) bool {
	if IsGroupType(t) {
		return false
	}
	switch strings.ToLower(t) {
	case "direct", "reject", "rejectdrop", "compatible", "pass", "dns":
		return false
	}
	return true
}

// ResolveGroup picks the group NodePulse is allowed to manage. It never picks
// GLOBAL. A requested group must exist and be a Selector.
func ResolveGroup(proxies map[string]clash.ProxyInfo, requested string) (string, error) {
	if requested != "" {
		if strings.EqualFold(requested, GlobalGroup) {
			return "", fmt.Errorf("GLOBAL 不能被自动管理，请选择一个 Selector 组")
		}
		info, ok := proxies[requested]
		if !ok {
			return "", fmt.Errorf("找不到组 %q", requested)
		}
		if !IsSelector(info) {
			return "", fmt.Errorf("组 %q 不是 Selector（当前 %s）", requested, info.Type)
		}
		return requested, nil
	}

	names := make([]string, 0, len(proxies))
	for name, info := range proxies {
		if strings.EqualFold(name, GlobalGroup) || !IsSelector(info) {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return "", fmt.Errorf("没有找到可管理的 Selector 组；请在设置里选择组")
	}

	// Prefer the conventional names without baking personal subscription names
	// into the product.
	sort.SliceStable(names, func(i, j int) bool {
		si, sj := groupPreference(names[i]), groupPreference(names[j])
		if si != sj {
			return si > sj
		}
		return names[i] < names[j]
	})
	return names[0], nil
}

func groupPreference(name string) int {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(name, "节点选择"):
		return 3
	case lower == "proxy" || lower == "proxies" || lower == "selector":
		return 2
	case strings.Contains(lower, "proxy"):
		return 1
	default:
		return 0
	}
}

// Candidates returns the direct, testable nodes inside a group. It does not
// fall back to every proxy: switching a node outside the managed group is worse
// than reporting that the group is empty.
func Candidates(proxies map[string]clash.ProxyInfo, groupName string) ([]config.Proxy, Meta, error) {
	info, ok := proxies[groupName]
	if !ok {
		return nil, Meta{}, fmt.Errorf("找不到组 %q", groupName)
	}
	meta := Meta{Total: len(info.All)}
	seen := map[string]bool{}
	out := make([]config.Proxy, 0, len(info.All))
	for _, name := range info.All {
		if seen[name] {
			continue
		}
		seen[name] = true
		member, ok := proxies[name]
		if !ok {
			meta.Skipped++
			continue
		}
		if IsGroupType(member.Type) {
			meta.NestedSkipped++
			continue
		}
		if !IsTestable(member.Type) {
			meta.Skipped++
			continue
		}
		out = append(out, config.Proxy{Name: name, Type: member.Type})
	}
	meta.Direct = len(out)
	if len(out) == 0 {
		return nil, meta, fmt.Errorf("组 %q 没有可直接测速的节点（嵌套组 %d，跳过 %d）", groupName, meta.NestedSkipped, meta.Skipped)
	}
	return out, meta, nil
}

// AllTestable returns every direct node known to the controller, sorted. This
// is for diagnostics/one-click testing when no managed group exists; it must
// not be used to switch a group.
func AllTestable(proxies map[string]clash.ProxyInfo) []config.Proxy {
	out := make([]config.Proxy, 0, len(proxies))
	for name, info := range proxies {
		if !IsTestable(info.Type) || config.IsInfoNode(name) {
			continue
		}
		out = append(out, config.Proxy{Name: name, Type: info.Type})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Confidence labels how much the sample can be trusted.
func Confidence(rounds int) string {
	if rounds < 3 {
		return "low"
	}
	if rounds < 5 {
		return "medium"
	}
	return "high"
}

// ResultFor finds a node in a result set.
func ResultFor(results []score.Result, name string) (score.Result, bool) {
	for _, result := range results {
		if result.Name == name {
			return result, true
		}
	}
	return score.Result{Name: name}, false
}

// Decision is the outcome of comparing the current node with ranked candidates.
type Decision struct {
	Switch     bool
	Best       score.Result
	BestOK     bool
	Streak     int
	ReasonCode string
	Reason     string
}

// Decide applies success threshold, hysteresis margin and consecutive
// confirmation. A manual one-click can ignore the confirmation by passing
// Confirm=1.
func Decide(current string, currentResult score.Result, ranked []score.Result, opts Options) Decision {
	opts = normalizeOptions(opts)
	best, ok := FirstQualified(ranked, opts.MinSuccess)
	if !ok {
		return Decision{ReasonCode: "no-qualified", Reason: "没有节点达到最低成功率"}
	}
	d := Decision{Best: best, BestOK: true}
	if current == "" {
		d.Switch = true
		d.ReasonCode = "no-current"
		d.Reason = "当前没有选中节点"
		return d
	}
	if currentResult.Success == 0 || currentResult.SuccessRate() < opts.MinSuccess {
		d.Switch = true
		d.ReasonCode = "current-unhealthy"
		d.Reason = fmt.Sprintf("当前节点成功率 %d/%d，低于 %.0f%%", currentResult.Success, currentResult.Total, opts.MinSuccess*100)
		return d
	}
	if best.Name == current {
		d.ReasonCode = "already-best"
		d.Reason = "当前已经是达标节点中最优的"
		return d
	}
	if best.Score > currentResult.Score*(1+opts.Margin) {
		d.Streak = opts.Streak + 1
		if d.Streak >= opts.Confirm {
			confirmed := d.Streak
			d.Switch = true
			d.Streak = 0
			d.ReasonCode = "better-candidate"
			d.Reason = fmt.Sprintf("候选评分 %.3f 高于当前 %.3f，超过 %.0f%% 余量，连续 %d 轮确认", best.Score, currentResult.Score, opts.Margin*100, confirmed)
			return d
		}
		d.ReasonCode = "better-candidate-confirming"
		d.Reason = fmt.Sprintf("候选更好，等待确认 %d/%d", d.Streak, opts.Confirm)
		return d
	}
	d.ReasonCode = "within-margin"
	d.Reason = fmt.Sprintf("候选优势低于 %.0f%% 余量，保持当前节点", opts.Margin*100)
	return d
}

// FirstQualified returns the first ranked node that meets the success floor.
func FirstQualified(ranked []score.Result, minSuccess float64) (score.Result, bool) {
	for _, result := range ranked {
		if result.Success > 0 && result.SuccessRate() >= minSuccess {
			return result, true
		}
	}
	return score.Result{}, false
}

// Options controls the decision.
type Options struct {
	MinSuccess float64
	Margin     float64
	Confirm    int
	Streak     int
}

func normalizeOptions(opts Options) Options {
	if opts.MinSuccess <= 0 {
		opts.MinSuccess = DefaultMinSuccess
	}
	if opts.Margin <= 0 {
		opts.Margin = DefaultMargin
	}
	if opts.Confirm <= 0 {
		opts.Confirm = DefaultConfirm
	}
	return opts
}

// Delayer is the node-delay capability needed for a baseline check.
type Delayer interface {
	Delay(ctx context.Context, name, testURL string, timeout time.Duration) (int, error)
}

// BaselineResult tells the UI whether a failure is local (DIRECT also fails)
// or node-specific.
type BaselineResult struct {
	Available bool
	OK        bool
	LatencyMs int
	Err       error
}

// Baseline tests DIRECT when the controller exposes it.
func Baseline(ctx context.Context, delayer Delayer, proxies map[string]clash.ProxyInfo, testURL string, timeout time.Duration) BaselineResult {
	if _, ok := proxies["DIRECT"]; !ok {
		return BaselineResult{}
	}
	ms, err := delayer.Delay(ctx, "DIRECT", testURL, timeout)
	return BaselineResult{Available: true, OK: err == nil, LatencyMs: ms, Err: err}
}
