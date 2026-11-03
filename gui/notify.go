package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/HanphoneJan/nodepulse/internal/settings"
	"github.com/HanphoneJan/nodepulse/internal/store"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const errorNotifyInterval = 30 * time.Minute

// notifyEvent sends a desktop notification only for events the user needs to
// act on, and only while the window is out of sight.
func (a *App) notifyEvent(event store.Event) {
	switch event.Kind {
	case "switch", "rollback", "error":
	default:
		return
	}
	s, err := settings.Load("")
	if err != nil || !s.NotifyOnSwitch {
		return
	}

	a.mu.Lock()
	hidden := a.windowHidden
	if event.Kind == "error" {
		if time.Since(a.lastErrorNotify) < errorNotifyInterval {
			a.mu.Unlock()
			return
		}
		a.lastErrorNotify = time.Now()
	}
	a.mu.Unlock()

	if a.ctx != nil && !hidden && !runtime.WindowIsMinimised(a.ctx) {
		return
	}

	title, body := notificationText(event)
	if body == "" {
		return
	}
	_ = sendNotification(title, body)
}

func notificationText(event store.Event) (string, string) {
	switch event.Kind {
	case "switch":
		body := fmt.Sprintf("%s → %s", valueOr(event.From, "未知节点"), valueOr(event.To, "新节点"))
		if event.Reason != "" {
			body += "\n" + truncate(event.Reason, 160)
		}
		return "NodePulse 已切换节点", body
	case "rollback":
		body := fmt.Sprintf("已回滚到 %s", valueOr(event.From, "原节点"))
		if event.Reason != "" {
			body += "\n" + truncate(event.Reason, 160)
		}
		return "NodePulse 已回滚", body
	case "error":
		return "NodePulse 自动选择失败", truncate(valueOr(event.Reason, "控制口或测速失败"), 180)
	default:
		return "", ""
	}
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func truncate(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit-1]) + "…"
}
