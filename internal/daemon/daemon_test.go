package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/selector"
)

type fakeAPI struct {
	proxies  map[string]clash.ProxyInfo
	delays   map[string]int
	switched []string
}

func (f *fakeAPI) Proxies(context.Context) (map[string]clash.ProxyInfo, error) {
	return f.proxies, nil
}

func (f *fakeAPI) Delay(_ context.Context, name, _ string, _ time.Duration) (int, error) {
	if ms, ok := f.delays[name]; ok {
		return ms, nil
	}
	return 0, errors.New("unreachable")
}

func (f *fakeAPI) Current(_ context.Context, group string) (string, error) {
	return f.proxies[group].Now, nil
}

func (f *fakeAPI) Switch(_ context.Context, group, node string) error {
	f.switched = append(f.switched, node)
	info := f.proxies[group]
	info.Now = node
	f.proxies[group] = info
	return nil
}

func newFake(current string) *fakeAPI {
	return &fakeAPI{
		proxies: map[string]clash.ProxyInfo{
			"Proxy": {Type: "Selector", Now: current, All: []string{"A", "B"}},
			"A":     {Type: "ss"},
			"B":     {Type: "ss"},
		},
		delays: map[string]int{"A": 100, "B": 800},
	}
}

func TestRunOnceSwitchesToBetterCandidate(t *testing.T) {
	api := newFake("B")
	d := New(api, Options{
		Group:       "Proxy",
		Rounds:      2,
		Concurrency: 2,
		Confirm:     1,
		Margin:      0.2,
		MinSuccess:  0.8,
		Once:        true,
	})
	if err := d.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(api.switched) != 1 || api.switched[0] != "A" {
		t.Fatalf("switched = %v, want [A]", api.switched)
	}
	if api.proxies["Proxy"].Now != "A" {
		t.Fatalf("current = %q, want A", api.proxies["Proxy"].Now)
	}
}

func TestRunOnceSwitchesWhenCurrentFails(t *testing.T) {
	api := newFake("B")
	delete(api.delays, "B")
	d := New(api, Options{Group: "Proxy", Rounds: 2, Concurrency: 2, Confirm: 2, MinSuccess: 0.8})
	if err := d.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(api.switched) != 1 || api.switched[0] != "A" {
		t.Fatalf("switched = %v, want [A]", api.switched)
	}
}

func TestRunOnceDryRunDoesNotSwitch(t *testing.T) {
	api := newFake("B")
	d := New(api, Options{Group: "Proxy", Rounds: 2, Concurrency: 2, Confirm: 1, DryRun: true})
	if err := d.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(api.switched) != 0 {
		t.Fatalf("switched = %v, want none", api.switched)
	}
}

func TestDecideNeedsConfirmations(t *testing.T) {
	cur := score.Result{Name: "B", Success: 10, Total: 10, Score: 0.5}
	best := score.Result{Name: "A", Success: 10, Total: 10, Score: 0.9}
	ranked := []score.Result{best, cur}

	first := selector.Decide("B", cur, ranked, selector.Options{Margin: 0.2, Confirm: 2, MinSuccess: 0.8})
	if first.Switch || first.Streak != 1 {
		t.Fatalf("first cycle: switch=%v streak=%d, want no switch and streak 1", first.Switch, first.Streak)
	}
	second := selector.Decide("B", cur, ranked, selector.Options{Margin: 0.2, Confirm: 2, MinSuccess: 0.8, Streak: first.Streak})
	if !second.Switch || second.Best.Name != "A" || second.Streak != 0 {
		t.Fatalf("second cycle: %+v, want switch A and streak 0", second)
	}
}

func TestDecideKeepsCurrentWithinMargin(t *testing.T) {
	cur := score.Result{Name: "B", Success: 10, Total: 10, Score: 0.90}
	best := score.Result{Name: "A", Success: 10, Total: 10, Score: 0.95}
	decision := selector.Decide("B", cur, []score.Result{best, cur}, selector.Options{Margin: 0.2, Confirm: 1})
	if decision.Switch {
		t.Fatalf("decision = %+v, want no switch", decision)
	}
}
