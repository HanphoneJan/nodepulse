package daemon

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/store"
	"github.com/HanphoneJan/nodepulse/internal/tester"
)

// API is the subset of a Clash controller used by the daemon.
type API interface {
	Proxies(ctx context.Context) (map[string]clash.ProxyInfo, error)
	Delay(ctx context.Context, name, testURL string, timeout time.Duration) (int, error)
	Current(ctx context.Context, group string) (string, error)
	Switch(ctx context.Context, group, node string) error
}

// Options controls the daemon.
type Options struct {
	Group       string
	Interval    time.Duration
	Rounds      int
	Concurrency int
	TestURL     string
	Timeout     time.Duration
	Margin      float64
	Confirm     int
	MinSuccess  float64
	Once        bool
	DryRun      bool
	Store       *store.Store
	OnEvent     func(store.Event)
}

// Daemon periodically tests nodes and switches the selector group when the
// current node degrades.
type Daemon struct {
	api    API
	opts   Options
	tester *tester.Tester
	streak int
}

// New creates a daemon with defaults applied.
func New(api API, opts Options) *Daemon {
	if opts.Interval <= 0 {
		opts.Interval = 5 * time.Minute
	}
	if opts.Rounds <= 0 {
		opts.Rounds = 3
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 8
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.TestURL == "" {
		opts.TestURL = "https://www.gstatic.com/generate_204"
	}
	if opts.Margin <= 0 {
		opts.Margin = 0.2
	}
	if opts.Confirm <= 0 {
		opts.Confirm = 2
	}
	if opts.MinSuccess <= 0 {
		opts.MinSuccess = 0.8
	}
	t := tester.New(api, tester.Options{
		Rounds:      opts.Rounds,
		Concurrency: opts.Concurrency,
		URL:         opts.TestURL,
		Timeout:     opts.Timeout,
	})
	return &Daemon{api: api, opts: opts, tester: t}
}

// Run tests and switches until the context is canceled. In Once mode it runs
// a single cycle.
func (d *Daemon) Run(ctx context.Context) error {
	for {
		if err := d.RunOnce(ctx); err != nil {
			d.emit(store.Event{Kind: "error", Reason: err.Error()})
			if d.opts.Once {
				return err
			}
		}
		if d.opts.Once {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d.opts.Interval):
		}
	}
}

// RunOnce executes one test/switch cycle.
func (d *Daemon) RunOnce(ctx context.Context) error {
	proxies, err := d.api.Proxies(ctx)
	if err != nil {
		return fmt.Errorf("list proxies: %w", err)
	}
	group, err := d.resolveGroup(proxies)
	if err != nil {
		return err
	}
	info := proxies[group]
	current := info.Now

	nodes := candidates(info, proxies)
	if len(nodes) == 0 {
		return fmt.Errorf("group %q has no testable nodes", group)
	}
	results := d.tester.Test(ctx, nodes)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	ranked := score.Rank(results)

	var currentResult score.Result
	foundCurrent := false
	for _, r := range results {
		if r.Name == current {
			currentResult = r
			foundCurrent = true
			break
		}
	}
	if !foundCurrent {
		currentResult = score.Result{Name: current}
	}

	best, bestOK := firstQualified(ranked, d.opts.MinSuccess)
	to, reason, streak := d.decide(current, currentResult, best, bestOK)
	d.streak = streak

	ev := store.Event{
		Kind:      "cycle",
		Group:     group,
		From:      current,
		To:        to,
		Reason:    reason,
		Best:      best.Name,
		BestScore: best.Score,
		CurScore:  currentResult.Score,
		OK:        true,
	}
	if to == "" {
		d.emit(ev)
		return nil
	}
	if d.opts.DryRun {
		ev.Reason = "[dry-run] " + reason
		d.emit(ev)
		return nil
	}

	if err := d.api.Switch(ctx, group, to); err != nil {
		ev.OK = false
		ev.Reason = reason + "; switch failed: " + err.Error()
		d.emit(ev)
		return fmt.Errorf("switch %s -> %s: %w", group, to, err)
	}
	// Verify the switch. Some controllers accept the call but keep the old
	// selection, so never trust the response alone.
	time.Sleep(200 * time.Millisecond)
	now, err := d.api.Current(ctx, group)
	if err != nil || now != to {
		ev.Kind = "rollback"
		ev.OK = false
		ev.Reason = fmt.Sprintf("%s; verify failed (now=%q err=%v), rolling back", reason, now, err)
		if current != "" {
			_ = d.api.Switch(ctx, group, current)
		}
		d.emit(ev)
		return fmt.Errorf("verify switch to %q failed: now=%q err=%v", to, now, err)
	}
	ev.Kind = "switch"
	d.emit(ev)
	d.streak = 0
	return nil
}

func (d *Daemon) decide(current string, cur, best score.Result, bestOK bool) (to, reason string, streak int) {
	if !bestOK {
		return "", "no node passes min-success", 0
	}
	if current == "" {
		return best.Name, "no current node", 0
	}
	if cur.Success == 0 || cur.SuccessRate() < d.opts.MinSuccess {
		return best.Name, fmt.Sprintf("current below min-success (%d/%d)", cur.Success, cur.Total), 0
	}
	if best.Name == current {
		return "", "current is already the best qualified node", 0
	}
	if best.Score > cur.Score*(1+d.opts.Margin) {
		n := d.streak + 1
		if n >= d.opts.Confirm {
			return best.Name, fmt.Sprintf("candidate score %.3f > current %.3f by >%.0f%% for %d cycles",
				best.Score, cur.Score, d.opts.Margin*100, n), 0
		}
		return "", fmt.Sprintf("candidate better, waiting for confirmation (%d/%d)", n, d.opts.Confirm), n
	}
	return "", "no candidate exceeds the switch margin", 0
}

func (d *Daemon) resolveGroup(proxies map[string]clash.ProxyInfo) (string, error) {
	if d.opts.Group != "" {
		if _, ok := proxies[d.opts.Group]; !ok {
			return "", fmt.Errorf("group %q not found", d.opts.Group)
		}
		return d.opts.Group, nil
	}
	preferred := []string{"🚀节点选择", "节点选择", "Proxy", "PROXY", "一分机场", "赔钱机场", "GLOBAL"}
	for _, name := range preferred {
		if info, ok := proxies[name]; ok && isSelector(info) {
			return name, nil
		}
	}
	names := make([]string, 0, len(proxies))
	for name, info := range proxies {
		if isSelector(info) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) > 0 {
		return names[0], nil
	}
	return "", fmt.Errorf("no selector group found; pass --group")
}

func isSelector(info clash.ProxyInfo) bool {
	return strings.EqualFold(info.Type, "Selector")
}

func isTestable(info clash.ProxyInfo) bool {
	switch strings.ToLower(info.Type) {
	case "selector", "urltest", "fallback", "loadbalance", "direct", "reject", "rejectdrop", "compatible", "pass", "dns":
		return false
	}
	return true
}

func candidates(group clash.ProxyInfo, proxies map[string]clash.ProxyInfo) []config.Proxy {
	seen := map[string]bool{}
	out := make([]config.Proxy, 0, len(group.All))
	for _, name := range group.All {
		if seen[name] {
			continue
		}
		seen[name] = true
		info, ok := proxies[name]
		if !ok || !isTestable(info) {
			continue
		}
		out = append(out, config.Proxy{Name: name, Type: info.Type})
	}
	// Some controllers omit `all` for selector groups; fall back to every
	// testable proxy.
	if len(out) == 0 {
		for name, info := range proxies {
			if isTestable(info) {
				out = append(out, config.Proxy{Name: name, Type: info.Type})
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	}
	return out
}

func firstQualified(ranked []score.Result, minSuccess float64) (score.Result, bool) {
	for _, r := range ranked {
		if r.Success > 0 && r.SuccessRate() >= minSuccess {
			return r, true
		}
	}
	return score.Result{}, false
}

func (d *Daemon) emit(ev store.Event) {
	if d.opts.Store != nil {
		_ = d.opts.Store.Append(ev)
	}
	if d.opts.OnEvent != nil {
		d.opts.OnEvent(ev)
	}
}
