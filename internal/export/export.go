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
}

// Selection is the result of filtering a subscription.
type Selection struct {
	Config   *config.Config
	Included []string
	Excluded []string
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
