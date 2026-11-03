package export

import (
	"fmt"

	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/score"
)

// Impact describes what an export will change before anything is written.
type Impact struct {
	Preserve       bool          `json:"preserve"`
	TotalNodes     int           `json:"total_nodes"`
	Included       int           `json:"included"`
	Excluded       []string      `json:"excluded,omitempty"`
	Groups         int           `json:"groups"`
	ChangedGroups  []GroupImpact `json:"changed_groups,omitempty"`
	DirectFallback []string      `json:"direct_fallback,omitempty"`
	Rules          int           `json:"rules"`
	RuleProviders  int           `json:"rule_providers"`
	DNS            bool          `json:"dns"`
	TUN            bool          `json:"tun"`
	Warnings       []string      `json:"warnings,omitempty"`
}

// GroupImpact shows how one proxy group changes.
type GroupImpact struct {
	Name     string   `json:"name"`
	Removed  []string `json:"removed,omitempty"`
	Fallback bool     `json:"fallback"`
}

// Preview builds the same selection Stable would write and summarizes the
// difference for the UI. It never touches the filesystem.
func Preview(cfg *config.Config, results []score.Result, opts Options) (*Impact, error) {
	sel, err := Stable(cfg, results, opts)
	if err != nil {
		return nil, err
	}
	out := sel.Config
	impact := &Impact{
		Preserve:      sel.Preserved,
		TotalNodes:    len(results),
		Included:      len(sel.Included),
		Excluded:      append([]string(nil), sel.Excluded...),
		Groups:        len(out.ProxyGroups),
		Rules:         len(out.Rules),
		RuleProviders: countRuleProviders(out),
		DNS:           hasKey(out, "dns"),
		TUN:           hasKey(out, "tun"),
	}

	if !sel.Preserved {
		if len(cfg.ProxyGroups) > 0 || len(cfg.Rules) > 0 {
			impact.Warnings = append(impact.Warnings, fmt.Sprintf("最小配置会替换原有 %d 个分组和 %d 条规则", len(cfg.ProxyGroups), len(cfg.Rules)))
		}
		if hasKey(cfg, "dns") || hasKey(cfg, "tun") || countRuleProviders(cfg) > 0 {
			impact.Warnings = append(impact.Warnings, "最小配置会丢弃 DNS、TUN 或 rule-providers")
		}
		return impact, nil
	}

	after := make(map[string]config.Group, len(out.ProxyGroups))
	for _, group := range out.ProxyGroups {
		after[group.Name] = group
	}
	for _, before := range cfg.ProxyGroups {
		groupAfter, ok := after[before.Name]
		if !ok {
			continue
		}
		kept := make(map[string]bool, len(groupAfter.Proxies))
		for _, name := range groupAfter.Proxies {
			kept[name] = true
		}
		var removed []string
		for _, name := range before.Proxies {
			if !kept[name] {
				removed = append(removed, name)
			}
		}
		fallback := len(groupAfter.Proxies) == 1 && groupAfter.Proxies[0] == "DIRECT" && len(before.Proxies) > 0
		if len(removed) == 0 && !fallback {
			continue
		}
		impact.ChangedGroups = append(impact.ChangedGroups, GroupImpact{Name: before.Name, Removed: removed, Fallback: fallback})
		if fallback {
			impact.DirectFallback = append(impact.DirectFallback, before.Name)
		}
	}
	if len(impact.DirectFallback) > 0 {
		impact.Warnings = append(impact.Warnings, fmt.Sprintf("%d 个分组过滤后为空，将回退到 DIRECT", len(impact.DirectFallback)))
	}
	if len(impact.Excluded) > 0 {
		impact.Warnings = append(impact.Warnings, fmt.Sprintf("%d 个节点未通过阈值，不会被写入", len(impact.Excluded)))
	}
	return impact, nil
}

func hasKey(cfg *config.Config, key string) bool {
	if cfg == nil || cfg.Extra == nil {
		return false
	}
	_, ok := cfg.Extra[key]
	return ok
}

func countRuleProviders(cfg *config.Config) int {
	if cfg == nil || cfg.Extra == nil {
		return 0
	}
	value, ok := cfg.Extra["rule-providers"]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case map[string]any:
		return len(typed)
	case map[any]any:
		return len(typed)
	default:
		return 0
	}
}
