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

func TestStablePreservesRulesGroupsAndExtra(t *testing.T) {
	cfg := &config.Config{
		MixedPort: 7897,
		Mode:      "rule",
		Proxies: []config.Proxy{
			{Name: "A", Type: "ss", Server: "127.0.0.1", Port: 1},
			{Name: "B", Type: "ss", Server: "127.0.0.1", Port: 2},
		},
		ProxyGroups: []config.Group{
			{Name: "PROXY", Type: "select", Proxies: []string{"A", "B", "Auto"}},
			{Name: "Auto", Type: "url-test", Proxies: []string{"A", "B"}},
			{Name: "Streaming", Type: "select", Proxies: []string{"B"}},
		},
		Rules: []string{"DOMAIN-SUFFIX,example.com,PROXY", "MATCH,PROXY"},
		Extra: map[string]any{
			"dns": map[string]any{"enable": true, "nameserver": []any{"1.1.1.1"}},
			"tun": map[string]any{"enable": true},
		},
	}
	results := []score.Result{
		score.FromLatencies("A", "ss", repeat(100, 10), 10),
		score.FromLatencies("B", "ss", []int{900, 900}, 10),
	}
	sel, err := Stable(cfg, results, Options{MinSuccess: 0.8, TestURL: "https://example.com", Preserve: true})
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Preserved {
		t.Fatal("expected Preserved")
	}
	if sel.Config.MixedPort != 7897 {
		t.Fatalf("mixed port = %d", sel.Config.MixedPort)
	}
	if len(sel.Config.Rules) != 2 || sel.Config.Rules[0] != cfg.Rules[0] {
		t.Fatalf("rules = %v", sel.Config.Rules)
	}
	if sel.Config.Extra["dns"] == nil || sel.Config.Extra["tun"] == nil {
		t.Fatalf("extra fields were dropped: %+v", sel.Config.Extra)
	}
	groups := map[string]config.Group{}
	for _, group := range sel.Config.ProxyGroups {
		groups[group.Name] = group
	}
	if _, ok := groups["NodePulse"]; !ok {
		t.Fatal("NodePulse group missing")
	}
	if got := groups["PROXY"].Proxies; len(got) != 2 || got[0] != "A" || got[1] != "Auto" {
		t.Fatalf("PROXY members = %v", got)
	}
	if got := groups["Auto"].Proxies; len(got) != 1 || got[0] != "A" {
		t.Fatalf("Auto members = %v", got)
	}
	if got := groups["Streaming"].Proxies; len(got) != 1 || got[0] != "DIRECT" {
		t.Fatalf("empty group fallback = %v", got)
	}
	if sel.Config.Secret != "" || sel.Config.ExternalController != "" {
		t.Fatalf("controller credentials leaked into export")
	}
}

func TestStablePreserveAddsRuleWhenMissing(t *testing.T) {
	cfg := &config.Config{Proxies: []config.Proxy{{Name: "A", Type: "ss", Server: "127.0.0.1", Port: 1}}}
	results := []score.Result{score.FromLatencies("A", "ss", []int{100}, 1)}
	sel, err := Stable(cfg, results, Options{MinSuccess: 0.8, Preserve: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(sel.Config.Rules) != 1 || sel.Config.Rules[0] != "MATCH,NodePulse" {
		t.Fatalf("rules = %v", sel.Config.Rules)
	}
}

func TestPreviewPreserveReportsImpact(t *testing.T) {
	cfg := &config.Config{
		Proxies: []config.Proxy{
			{Name: "A", Type: "ss", Server: "127.0.0.1", Port: 1},
			{Name: "B", Type: "ss", Server: "127.0.0.1", Port: 2},
		},
		ProxyGroups: []config.Group{
			{Name: "PROXY", Type: "select", Proxies: []string{"A", "B"}},
			{Name: "Streaming", Type: "select", Proxies: []string{"B"}},
		},
		Rules: []string{"MATCH,PROXY"},
		Extra: map[string]any{
			"dns":            map[string]any{"enable": true},
			"tun":            map[string]any{"enable": true},
			"rule-providers": map[string]any{"ads": map[string]any{}},
		},
	}
	results := []score.Result{
		score.FromLatencies("A", "ss", repeat(100, 10), 10),
		score.FromLatencies("B", "ss", []int{900, 900}, 10),
	}
	impact, err := Preview(cfg, results, Options{MinSuccess: 0.8, Preserve: true})
	if err != nil {
		t.Fatal(err)
	}
	if !impact.Preserve || impact.Included != 1 || len(impact.Excluded) != 1 || impact.Excluded[0] != "B" {
		t.Fatalf("impact = %+v", impact)
	}
	if !impact.DNS || !impact.TUN || impact.RuleProviders != 1 {
		t.Fatalf("extra flags = %+v", impact)
	}
	if len(impact.DirectFallback) != 1 || impact.DirectFallback[0] != "Streaming" {
		t.Fatalf("fallbacks = %v", impact.DirectFallback)
	}
	if len(impact.Warnings) == 0 {
		t.Fatal("expected warnings")
	}
}

func TestPreviewMinimalWarnsAboutDroppedRules(t *testing.T) {
	cfg := &config.Config{
		Proxies:     []config.Proxy{{Name: "A", Type: "ss", Server: "127.0.0.1", Port: 1}},
		ProxyGroups: []config.Group{{Name: "PROXY", Proxies: []string{"A"}}},
		Rules:       []string{"MATCH,PROXY"},
		Extra:       map[string]any{"dns": map[string]any{"enable": true}},
	}
	results := []score.Result{score.FromLatencies("A", "ss", repeat(100, 10), 10)}
	impact, err := Preview(cfg, results, Options{MinSuccess: 0.8})
	if err != nil {
		t.Fatal(err)
	}
	if impact.Preserve {
		t.Fatal("minimal preview should not be preserved")
	}
	if len(impact.Warnings) < 2 {
		t.Fatalf("warnings = %v", impact.Warnings)
	}
}
