package daemon

import (
	"context"
	"fmt"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/selector"
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
	Group        string
	Interval     time.Duration
	Rounds       int
	Concurrency  int
	TestURL      string
	Timeout      time.Duration
	Margin       float64
	Confirm      int
	MinSuccess   float64
	VerifyDelay  time.Duration
	VerifyRounds int
	Baseline     bool
	Once         bool
	DryRun       bool
	Store        *store.Store
	OnEvent      func(store.Event)
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
		opts.Margin = selector.DefaultMargin
	}
	if opts.Confirm <= 0 {
		opts.Confirm = selector.DefaultConfirm
	}
	if opts.MinSuccess <= 0 {
		opts.MinSuccess = selector.DefaultMinSuccess
	}
	if opts.VerifyDelay <= 0 {
		opts.VerifyDelay = 300 * time.Millisecond
	}
	if opts.VerifyRounds <= 0 {
		opts.VerifyRounds = 1
	}
	// Baseline diagnostics are always on; they only run when no node passes.
	opts.Baseline = true
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
			if ctx.Err() != nil {
				return ctx.Err()
			}
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

// RunOnce executes one test/switch cycle and emits a structured transaction.
func (d *Daemon) RunOnce(ctx context.Context) error {
	started := time.Now()
	proxies, err := d.api.Proxies(ctx)
	if err != nil {
		d.emit(store.Event{Kind: "error", ReasonCode: "controller-error", Reason: fmt.Sprintf("读取控制口失败: %v", err), OK: false})
		return fmt.Errorf("list proxies: %w", err)
	}
	group, err := selector.ResolveGroup(proxies, d.opts.Group)
	if err != nil {
		d.emit(store.Event{Kind: "error", ReasonCode: "group-error", Reason: err.Error(), OK: false})
		return err
	}
	info := proxies[group]
	current := info.Now

	nodes, meta, err := selector.Candidates(proxies, group)
	if err != nil {
		d.emit(store.Event{Kind: "error", Group: group, ReasonCode: "no-candidates", Reason: err.Error(), Candidates: 0, NestedSkipped: meta.NestedSkipped, OK: false})
		return err
	}

	results := d.tester.Test(ctx, nodes)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	ranked := score.Rank(results)
	currentResult, _ := selector.ResultFor(results, current)

	decision := selector.Decide(current, currentResult, ranked, selector.Options{
		MinSuccess: d.opts.MinSuccess,
		Margin:     d.opts.Margin,
		Confirm:    d.opts.Confirm,
		Streak:     d.streak,
	})
	d.streak = decision.Streak

	to := ""
	if decision.Switch {
		to = decision.Best.Name
	}
	ev := store.Event{
		Time:          time.Now(),
		Kind:          "cycle",
		Group:         group,
		From:          current,
		To:            to,
		Reason:        decision.Reason,
		ReasonCode:    decision.ReasonCode,
		Best:          decision.Best.Name,
		BestScore:     decision.Best.Score,
		CurScore:      currentResult.Score,
		Confidence:    selector.Confidence(d.opts.Rounds),
		Candidates:    len(nodes),
		NestedSkipped: meta.NestedSkipped,
		OK:            true,
		DurationMs:    time.Since(started).Milliseconds(),
	}

	if !decision.BestOK {
		base := selector.BaselineResult{}
		if d.opts.Baseline {
			base = selector.Baseline(ctx, d.api, proxies, d.opts.TestURL, d.opts.Timeout)
		}
		switch {
		case base.Available && !base.OK:
			ev.OK = false
			ev.ReasonCode = "baseline-failed"
			ev.Reason = fmt.Sprintf("本地 DIRECT 基线失败，可能是网络或测试 URL 问题: %v", base.Err)
		case len(ranked) > 0 && ranked[0].LastError != "":
			ev.OK = false
			ev.ReasonCode = "all-nodes-failed"
			ev.Reason = "没有节点达标；最近错误: " + ranked[0].LastError
		}
		d.emit(ev)
		return nil
	}

	if !decision.Switch {
		d.emit(ev)
		return nil
	}
	if d.opts.DryRun {
		ev.To = ""
		ev.ReasonCode = "dry-run"
		ev.Reason = "[dry-run] " + decision.Reason
		d.emit(ev)
		return nil
	}

	if err := d.api.Switch(ctx, group, decision.Best.Name); err != nil {
		ev.Kind = "error"
		ev.OK = false
		ev.ReasonCode = "switch-failed"
		ev.Reason = fmt.Sprintf("%s；切换失败: %v", decision.Reason, err)
		ev.DurationMs = time.Since(started).Milliseconds()
		d.emit(ev)
		return fmt.Errorf("switch %s -> %s: %w", group, decision.Best.Name, err)
	}

	if err := d.wait(ctx, d.opts.VerifyDelay); err != nil {
		return err
	}
	now, currentErr := d.api.Current(ctx, group)
	verifyErr := currentErr
	if verifyErr == nil && now != decision.Best.Name {
		verifyErr = fmt.Errorf("控制器仍显示 %q", now)
	}
	if verifyErr == nil {
		verifyErr = d.verifyNode(ctx, decision.Best.Name)
	}
	if verifyErr != nil {
		ev.Kind = "rollback"
		ev.OK = false
		ev.ReasonCode = "verify-failed"
		ev.RollbackAttempted = current != ""
		if current != "" {
			rollbackErr := d.api.Switch(ctx, group, current)
			ev.RollbackOK = rollbackErr == nil
			if rollbackErr != nil {
				ev.ReasonCode = "rollback-failed"
				ev.Reason = fmt.Sprintf("%s；切换验证失败: %v；回滚也失败: %v", decision.Reason, verifyErr, rollbackErr)
			} else {
				ev.Reason = fmt.Sprintf("%s；切换验证失败: %v；已回滚到 %s", decision.Reason, verifyErr, current)
			}
		} else {
			ev.Reason = fmt.Sprintf("%s；切换验证失败: %v；没有可回滚的节点", decision.Reason, verifyErr)
		}
		ev.DurationMs = time.Since(started).Milliseconds()
		d.streak = 0
		d.emit(ev)
		return fmt.Errorf("verify switch to %q failed: %v", decision.Best.Name, verifyErr)
	}

	ev.Kind = "switch"
	ev.To = decision.Best.Name
	ev.From = current
	ev.Verified = true
	ev.Reason = decision.Reason
	ev.DurationMs = time.Since(started).Milliseconds()
	d.streak = 0
	d.emit(ev)
	return nil
}

func (d *Daemon) verifyNode(ctx context.Context, name string) error {
	for i := 0; i < d.opts.VerifyRounds; i++ {
		if i > 0 {
			if err := d.wait(ctx, d.opts.VerifyDelay); err != nil {
				return err
			}
		}
		ms, err := d.api.Delay(ctx, name, d.opts.TestURL, d.opts.Timeout)
		if err != nil {
			return fmt.Errorf("验证 %s 失败: %w", name, err)
		}
		if ms <= 0 {
			return fmt.Errorf("验证 %s 返回无效延迟", name)
		}
	}
	return nil
}

func (d *Daemon) wait(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (d *Daemon) emit(ev store.Event) {
	if d.opts.Store != nil {
		_ = d.opts.Store.Append(ev)
	}
	if d.opts.OnEvent != nil {
		d.opts.OnEvent(ev)
	}
}
