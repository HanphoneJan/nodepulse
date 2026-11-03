// Package health reports whether the current environment can actually run a
// test/auto-selection. It is read-only and intentionally returns actionable
// checks instead of a setup wizard.
package health

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/discover"
	"github.com/HanphoneJan/nodepulse/internal/selector"
	"github.com/HanphoneJan/nodepulse/internal/settings"
	"github.com/HanphoneJan/nodepulse/internal/subscription"
)

const (
	StatusOK    = "ok"
	StatusWarn  = "warn"
	StatusError = "error"
)

// Check is one environment check.
type Item struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Hint   string `json:"hint,omitempty"`
}

// Report is the health snapshot shown by the settings page.
type Report struct {
	OK        bool      `json:"ok"`
	Checks    []Item    `json:"checks"`
	CheckedAt time.Time `json:"checked_at"`
}

// Check inspects the controller, subscription, mihomo and managed group.
func Run(ctx context.Context, s settings.Settings) Report {
	report := Report{OK: true, CheckedAt: time.Now()}
	add := func(check Item) {
		if check.Status == StatusError {
			report.OK = false
		}
		report.Checks = append(report.Checks, check)
	}

	var proxies map[string]clash.ProxyInfo
	if s.API == "" {
		add(Item{ID: "controller", Label: "控制口", Status: StatusWarn, Detail: "尚未配置", Hint: "连接 Clash 控制口，或选择本地订阅走 sidecar"})
	} else {
		callCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		list, err := clash.New(s.API, s.Secret).Proxies(callCtx)
		cancel()
		if err != nil {
			add(Item{ID: "controller", Label: "控制口", Status: StatusError, Detail: err.Error(), Hint: "确认客户端已启动，并检查控制口地址/secret"})
		} else {
			proxies = list
			add(Item{ID: "controller", Label: "控制口", Status: StatusOK, Detail: s.API, Hint: "连接正常"})
			group, err := selector.ResolveGroup(proxies, s.Group)
			if err != nil {
				add(Item{ID: "group", Label: "Selector 组", Status: StatusWarn, Detail: err.Error(), Hint: "在设置里选择一个可管理的 Selector 组"})
			} else {
				add(Item{ID: "group", Label: "Selector 组", Status: StatusOK, Detail: group, Hint: "自动选择只会操作这个组"})
			}
		}
	}

	if s.ConfigPath == "" {
		add(Item{ID: "subscription", Label: "订阅配置", Status: StatusWarn, Detail: "尚未选择", Hint: "选择本地 YAML，或导入订阅 URL"})
	} else if cfg, err := subscription.Load(s.ConfigPath); err != nil {
		add(Item{ID: "subscription", Label: "订阅配置", Status: StatusError, Detail: err.Error(), Hint: "重新选择订阅文件或重新导入"})
	} else if len(cfg.RealProxies()) == 0 {
		add(Item{ID: "subscription", Label: "订阅配置", Status: StatusError, Detail: "没有可测试节点", Hint: "检查订阅是否过期或格式是否受支持"})
	} else {
		add(Item{ID: "subscription", Label: "订阅配置", Status: StatusOK, Detail: fmt.Sprintf("%s · %d 个节点", s.ConfigPath, len(cfg.RealProxies())), Hint: "导出稳定配置会使用这个文件"})
	}

	if s.MihomoPath != "" {
		path := s.MihomoPath
		if !filepath.IsAbs(path) {
			if found, err := exec.LookPath(path); err == nil {
				path = found
			}
		}
		if _, err := os.Stat(path); err != nil {
			add(Item{ID: "mihomo", Label: "mihomo", Status: StatusError, Detail: err.Error(), Hint: "重新选择 mihomo 可执行文件"})
		} else {
			add(Item{ID: "mihomo", Label: "mihomo", Status: StatusOK, Detail: path, Hint: "本地订阅测速可用"})
		}
	} else if path, err := discover.MihomoBinary(); err == nil {
		add(Item{ID: "mihomo", Label: "mihomo", Status: StatusOK, Detail: path, Hint: "本地订阅测速可用"})
	} else if s.API != "" {
		add(Item{ID: "mihomo", Label: "mihomo", Status: StatusWarn, Detail: "未找到", Hint: "已连接控制口，不需要 sidecar"})
	} else {
		add(Item{ID: "mihomo", Label: "mihomo", Status: StatusError, Detail: "未找到", Hint: "安装 mihomo，或连接一个 Clash 控制口"})
	}

	if s.URL == "" {
		add(Item{ID: "test-url", Label: "测速 URL", Status: StatusWarn, Detail: "使用默认值", Hint: "默认 https://www.gstatic.com/generate_204"})
	} else if parsed, err := url.Parse(s.URL); err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		add(Item{ID: "test-url", Label: "测速 URL", Status: StatusError, Detail: "不是有效的 http/https URL", Hint: "填写完整的 http/https URL"})
	} else {
		add(Item{ID: "test-url", Label: "测速 URL", Status: StatusOK, Detail: s.URL})
	}
	return report
}
