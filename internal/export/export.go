package export

import (
	"fmt"
	"sort"

	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/score"
)

// Options controls which nodes pass the stability filter.
type Options struct {
	MinSuccess  float64 // 0..1, default 0.8
	MaxP90Ms    int     // 0 disables the check
	MaxJitterMs int     // 0 disables the check
	TestURL     string
	MixedPort   int
	// Preserve keeps the original rules, groups, DNS, TUN and all unknown
	// fields, filters group members to the stable nodes and adds a
	// "NodePulse" url-test group. It is the default for daily-use exports.
	Preserve bool
}

// Selection is the result of filtering a subscription.
type Selection struct {
	Config    *config.Config
	Included  []string
	Excluded  []string
	Preserved bool
}

// Stable builds a minimal Clash config containing only nodes that pass the
// thresholds, plus a url-test group named "Auto".
func Stable(cfg *config.Config, results []score.Result, opts Options) (*Selection, error) {
	if opts.MinSuccess <= 0 {
		opts.MinSuccess = 0.8
	}
	if opts.TestURL == "" {
		opts.TestURL = "https://www.gstatic.com/generate_204"
	}
	if opts.MixedPort <= 0 {
		opts.MixedPort = 7890
	}

	ranked := score.Rank(results)
	proxies := make([]config.Proxy, 0, len(ranked))
	names := make([]string, 0, len(ranked))
	excluded := make([]string, 0)

	for _, r := range ranked {
		p, ok := cfg.FindProxy(r.Name)
		if !ok {
			excluded = append(excluded, r.Name)
			continue
		}
		if r.Success == 0 || r.SuccessRate() < opts.MinSuccess {
			excluded = append(excluded, r.Name)
			continue
		}
		if opts.MaxP90Ms > 0 && r.P90Ms > opts.MaxP90Ms {
			excluded = append(excluded, r.Name)
			continue
		}
		if opts.MaxJitterMs > 0 && r.JitterMs > opts.MaxJitterMs {
			excluded = append(excluded, r.Name)
			continue
		}
		proxies = append(proxies, p)
		names = append(names, p.Name)
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("no nodes passed the thresholds (min-success=%.2f max-p90=%dms max-jitter=%dms)",
			opts.MinSuccess, opts.MaxP90Ms, opts.MaxJitterMs)
	}

	sort.Strings(excluded)

	if opts.Preserve {
		out, err := preserve(cfg, proxies, names, opts)
		if err != nil {
			return nil, err
		}
		return &Selection{Config: out, Included: names, Excluded: excluded, Preserved: true}, nil
	}

	out := &config.Config{
		MixedPort: opts.MixedPort,
		Mode:      "rule",
		LogLevel:  "info",
		Proxies:   proxies,
		ProxyGroups: []config.Group{{
			Name:      "Auto",
			Type:      "url-test",
			URL:       opts.TestURL,
			Interval:  300,
			Tolerance: 50,
			Proxies:   names,
		}},
		Rules: []string{"MATCH,Auto"},
	}
	return &Selection{Config: out, Included: names, Excluded: excluded}, nil
}

// preserve filters the original config instead of replacing it. Original
// groups are kept (including nested group references), their member lists are
// reduced to nodes that passed, and a dedicated NodePulse url-test group is
// added. Rules, DNS, TUN, rule-providers and other unknown fields survive the
// round trip.
func preserve(cfg *config.Config, proxies []config.Proxy, names []string, opts Options) (*config.Config, error) {
	out, err := cfg.DeepClone()
	if err != nil {
		return nil, err
	}
	included := make(map[string]bool, len(names))
	for _, name := range names {
		included[name] = true
	}
	groupNames := make(map[string]bool, len(out.ProxyGroups))
	for _, group := range out.ProxyGroups {
		groupNames[group.Name] = true
	}
	knownProxies := make(map[string]bool, len(out.Proxies))
	for _, proxy := range out.Proxies {
		knownProxies[proxy.Name] = true
	}

	groups := make([]config.Group, 0, len(out.ProxyGroups)+1)
	for _, group := range out.ProxyGroups {
		if group.Name == "NodePulse" {
			continue
		}
		kept := make([]string, 0, len(group.Proxies))
		for _, name := range group.Proxies {
			// Keep stable nodes, nested groups and provider references.
			// Only known proxies that failed the filter are removed.
			if included[name] || groupNames[name] || !knownProxies[name] {
				kept = append(kept, name)
			}
		}
		if len(kept) == 0 {
			kept = []string{"DIRECT"}
		}
		group.Proxies = kept
		groups = append(groups, group)
	}
	groups = append(groups, config.Group{
		Name:      "NodePulse",
		Type:      "url-test",
		URL:       opts.TestURL,
		Interval:  300,
		Tolerance: 50,
		Proxies:   names,
	})

	out.Proxies = proxies
	out.ProxyGroups = groups
	out.Secret = ""
	out.ExternalController = ""
	if out.MixedPort <= 0 {
		out.MixedPort = opts.MixedPort
	}
	if len(out.Rules) == 0 {
		out.Rules = []string{"MATCH,NodePulse"}
	}
	return out, nil
}
