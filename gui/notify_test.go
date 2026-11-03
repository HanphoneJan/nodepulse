package main

import (
	"strings"
	"testing"

	"github.com/HanphoneJan/nodepulse/internal/store"
)

func TestNotificationText(t *testing.T) {
	title, body := notificationText(store.Event{Kind: "switch", From: "A", To: "B", Reason: "better"})
	if title == "" || !strings.Contains(body, "A → B") || !strings.Contains(body, "better") {
		t.Fatalf("title=%q body=%q", title, body)
	}
	title, body = notificationText(store.Event{Kind: "rollback", From: "A", Reason: "verify failed"})
	if !strings.Contains(title, "回滚") || !strings.Contains(body, "A") {
		t.Fatalf("title=%q body=%q", title, body)
	}
	if title, body := notificationText(store.Event{Kind: "cycle"}); title != "" || body != "" {
		t.Fatalf("cycle should not notify: %q %q", title, body)
	}
	if body := truncate(strings.Repeat("x", 300), 20); len([]rune(body)) != 20 {
		t.Fatalf("truncated body length = %d", len([]rune(body)))
	}
}
