package selector

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/score"
)

func proxies() map[string]clash.ProxyInfo {
	return map[string]clash.ProxyInfo{
		"GLOBAL": {Type: "Selector", Now: "Proxy", All: []string{"DIRECT", "Proxy"}},
		"Proxy":  {Type: "Selector", Now: "A", All: []string{"A", "B", "Auto", "DIRECT"}},
		"Auto":   {Type: "URLTest", All: []string{"A", "B"}},
		"Other":  {Type: "Selector", Now: "A", All: []string{"A"}},
		"A":      {Type: "ss"},
		"B":      {Type: "ss"},
		"DIRECT": {Type: "Direct"},
	}
}

func TestResolveGroupNeverPicksGlobal(t *testing.T) {
	got, err := ResolveGroup(proxies(), "")
	if err != nil || got != "Proxy" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := ResolveGroup(proxies(), "GLOBAL"); err == nil {
		t.Fatal("expected GLOBAL to be rejected")
	}
	if _, err := ResolveGroup(proxies(), "A"); err == nil {
		t.Fatal("expected non-selector to be rejected")
	}
}

func TestCandidatesDoNotFallBackOutsideGroup(t *testing.T) {
	nodes, meta, err := Candidates(proxies(), "Proxy")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 || nodes[0].Name != "A" || nodes[1].Name != "B" {
		t.Fatalf("nodes = %+v", nodes)
	}
	if meta.NestedSkipped != 1 || meta.Skipped != 1 {
		t.Fatalf("meta = %+v", meta)
	}

	empty := map[string]clash.ProxyInfo{
		"Proxy": {Type: "Selector", All: []string{"Auto"}},
		"Auto":  {Type: "URLTest", All: []string{"A"}},
		"A":     {Type: "ss"},
	}
	if _, _, err := Candidates(empty, "Proxy"); err == nil {
		t.Fatal("expected empty direct members to error")
	}
}

func TestDecide(t *testing.T) {
	best := score.Result{Name: "B", Success: 10, Total: 10, Score: .9}
	cur := score.Result{Name: "A", Success: 10, Total: 10, Score: .5}
	ranked := []score.Result{best, cur}

	d := Decide("A", cur, ranked, Options{MinSuccess: .8, Margin: .2, Confirm: 1})
	if !d.Switch || d.ReasonCode != "better-candidate" {
		t.Fatalf("expected switch: %+v", d)
	}

	d = Decide("A", cur, ranked, Options{MinSuccess: .8, Margin: .2, Confirm: 2, Streak: 0})
	if d.Switch || d.ReasonCode != "better-candidate-confirming" {
		t.Fatalf("expected first confirmation wait: %+v", d)
	}
	d = Decide("A", cur, ranked, Options{MinSuccess: .8, Margin: .2, Confirm: 2, Streak: 1})
	if !d.Switch || d.ReasonCode != "better-candidate" {
		t.Fatalf("expected switch after confirmation: %+v", d)
	}
	d = Decide("A", cur, ranked, Options{MinSuccess: .8, Margin: .2, Confirm: 3, Streak: 0})
	if d.Switch || d.ReasonCode != "better-candidate-confirming" {
		t.Fatalf("expected confirmation wait: %+v", d)
	}

	near := score.Result{Name: "A", Success: 10, Total: 10, Score: .88}
	d = Decide("A", near, ranked, Options{MinSuccess: .8, Margin: .2, Confirm: 2})
	if d.Switch || d.ReasonCode != "within-margin" {
		t.Fatalf("expected within margin: %+v", d)
	}

	bad := score.Result{Name: "A", Success: 2, Total: 10, Score: .1}
	d = Decide("A", bad, ranked, Options{MinSuccess: .8})
	if !d.Switch || d.ReasonCode != "current-unhealthy" {
		t.Fatalf("expected unhealthy switch: %+v", d)
	}

	d = Decide("B", best, ranked, Options{MinSuccess: .8})
	if d.Switch || d.ReasonCode != "already-best" {
		t.Fatalf("expected already best: %+v", d)
	}
}

type fakeDelayer struct {
	ms  int
	err error
}

func (f fakeDelayer) Delay(context.Context, string, string, time.Duration) (int, error) {
	return f.ms, f.err
}

func TestBaseline(t *testing.T) {
	if got := Baseline(context.Background(), fakeDelayer{ms: 10}, proxies(), "https://example.com", time.Second); !got.Available || !got.OK {
		t.Fatalf("baseline = %+v", got)
	}
	if got := Baseline(context.Background(), fakeDelayer{err: errors.New("down")}, proxies(), "https://example.com", time.Second); !got.Available || got.OK {
		t.Fatalf("failed baseline = %+v", got)
	}
	if got := Baseline(context.Background(), fakeDelayer{}, map[string]clash.ProxyInfo{}, "x", time.Second); got.Available {
		t.Fatalf("baseline without DIRECT should be unavailable: %+v", got)
	}
}

func TestConfidence(t *testing.T) {
	if Confidence(1) != "low" || Confidence(3) != "medium" || Confidence(5) != "high" {
		t.Fatal("unexpected confidence labels")
	}
}
