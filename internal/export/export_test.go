package export

import (
	"testing"

	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/score"
)

func TestStableFiltersAndBuildsGroup(t *testing.T) {
	cfg := &config.Config{Proxies: []config.Proxy{
		{Name: "A", Type: "ss", Server: "127.0.0.1", Port: 1},
		{Name: "B", Type: "ss", Server: "127.0.0.1", Port: 2},
	}}
	results := []score.Result{
		score.FromLatencies("A", "ss", repeat(100, 10), 10),
		score.FromLatencies("B", "ss", []int{100, 100}, 10),
	}
	sel, err := Stable(cfg, results, Options{MinSuccess: 0.8, TestURL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sel.Included) != 1 || sel.Included[0] != "A" {
		t.Fatalf("included = %v, want [A]", sel.Included)
	}
	if len(sel.Excluded) != 1 || sel.Excluded[0] != "B" {
		t.Fatalf("excluded = %v, want [B]", sel.Excluded)
	}
	if len(sel.Config.Proxies) != 1 || sel.Config.Proxies[0].Name != "A" {
		t.Fatalf("config proxies = %+v", sel.Config.Proxies)
	}
	if len(sel.Config.ProxyGroups) != 1 || sel.Config.ProxyGroups[0].Name != "Auto" {
		t.Fatalf("groups = %+v", sel.Config.ProxyGroups)
	}
	if sel.Config.ProxyGroups[0].URL != "https://example.com" {
		t.Fatalf("group url = %q", sel.Config.ProxyGroups[0].URL)
	}
	if len(sel.Config.Rules) != 1 || sel.Config.Rules[0] != "MATCH,Auto" {
		t.Fatalf("rules = %v", sel.Config.Rules)
	}
}

func TestStableErrorsWhenNothingPasses(t *testing.T) {
	cfg := &config.Config{Proxies: []config.Proxy{
		{Name: "B", Type: "ss", Server: "127.0.0.1", Port: 2},
	}}
	results := []score.Result{score.FromLatencies("B", "ss", []int{100}, 10)}
	if _, err := Stable(cfg, results, Options{MinSuccess: 0.8}); err == nil {
		t.Fatal("expected an error when no node passes")
	}
}

func repeat(v, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = v
	}
	return out
}
