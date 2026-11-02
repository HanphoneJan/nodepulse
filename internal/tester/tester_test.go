package tester

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/score"
)

type fakeDelayer struct {
	mu    sync.Mutex
	calls map[string]int
}

func (f *fakeDelayer) Delay(ctx context.Context, name, testURL string, timeout time.Duration) (int, error) {
	f.mu.Lock()
	f.calls[name]++
	n := f.calls[name]
	f.mu.Unlock()
	if name == "bad" {
		return 0, fmt.Errorf("boom")
	}
	return 100 + n, nil
}

func TestTestCountsSuccessAndCallsCallback(t *testing.T) {
	fake := &fakeDelayer{calls: map[string]int{}}
	tr := New(fake, Options{Rounds: 3, Concurrency: 2, URL: "https://example.com", Timeout: time.Second})

	var mu sync.Mutex
	var seen []string
	tr.OnNode(func(done, total int, r score.Result) {
		mu.Lock()
		seen = append(seen, r.Name)
		mu.Unlock()
	})

	nodes := []config.Proxy{
		{Name: "good", Type: "ss"},
		{Name: "bad", Type: "ss"},
		{Name: "also-good", Type: "ss"},
	}
	results := tr.Test(context.Background(), nodes)
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	byName := map[string]score.Result{}
	for _, r := range results {
		byName[r.Name] = r
	}
	if byName["good"].Success != 3 || byName["good"].Total != 3 {
		t.Fatalf("good = %+v", byName["good"])
	}
	if byName["bad"].Success != 0 || byName["bad"].Total != 3 {
		t.Fatalf("bad = %+v", byName["bad"])
	}
	if byName["also-good"].Success != 3 {
		t.Fatalf("also-good = %+v", byName["also-good"])
	}
	if len(seen) != 3 {
		t.Fatalf("callback calls = %d, want 3", len(seen))
	}
}

func TestTestCancelledContext(t *testing.T) {
	fake := &fakeDelayer{calls: map[string]int{}}
	tr := New(fake, Options{Rounds: 10, Concurrency: 1, URL: "https://example.com", Timeout: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results := tr.Test(ctx, []config.Proxy{{Name: "good", Type: "ss"}})
	if len(results) != 1 || results[0].Success != 0 {
		t.Fatalf("unexpected results: %+v", results)
	}
}
