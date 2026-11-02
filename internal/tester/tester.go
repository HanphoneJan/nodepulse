package tester

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/score"
)

// Delayer is the only capability the tester needs. It is implemented by the
// Clash REST client and, in tests, by fakes.
type Delayer interface {
	Delay(ctx context.Context, name, testURL string, timeout time.Duration) (int, error)
}

// Options controls a test run.
type Options struct {
	Rounds      int
	Concurrency int
	URL         string
	Timeout     time.Duration
}

// Tester runs multi-round tests over a set of nodes.
type Tester struct {
	delayer Delayer
	opts    Options

	mu     sync.Mutex
	onNode func(done, total int, result score.Result)

	total int
	done  atomic.Int64
}

// New creates a tester with sensible defaults applied.
func New(d Delayer, opts Options) *Tester {
	if opts.Rounds <= 0 {
		opts.Rounds = 5
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 8
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.URL == "" {
		opts.URL = "https://www.gstatic.com/generate_204"
	}
	return &Tester{delayer: d, opts: opts}
}

// OnNode registers a callback invoked after each node finishes. Calls are
// serialized so callbacks do not need their own locking.
func (t *Tester) OnNode(fn func(done, total int, result score.Result)) {
	t.mu.Lock()
	t.onNode = fn
	t.mu.Unlock()
}

// Test tests every node and returns results in input order.
func (t *Tester) Test(ctx context.Context, nodes []config.Proxy) []score.Result {
	if len(nodes) == 0 {
		return nil
	}

	t.total = len(nodes)
	t.done.Store(0)

	type job struct {
		index int
		node  config.Proxy
	}
	jobs := make(chan job)
	results := make([]score.Result, len(nodes))

	workers := t.opts.Concurrency
	if workers > len(nodes) {
		workers = len(nodes)
	}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				results[j.index] = t.testNode(ctx, j.node)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for i, node := range nodes {
			select {
			case <-ctx.Done():
				return
			case jobs <- job{index: i, node: node}:
			}
		}
	}()

	wg.Wait()
	return results
}

func (t *Tester) testNode(ctx context.Context, node config.Proxy) score.Result {
	latencies := make([]int, 0, t.opts.Rounds)
	for i := 0; i < t.opts.Rounds; i++ {
		if ctx.Err() != nil {
			break
		}
		ms, err := t.delayer.Delay(ctx, node.Name, t.opts.URL, t.opts.Timeout)
		if err != nil {
			continue
		}
		latencies = append(latencies, ms)
	}
	result := score.FromLatencies(node.Name, node.Type, latencies, t.opts.Rounds)
	t.emit(int(t.done.Add(1)), t.total, result)
	return result
}

func (t *Tester) emit(done, total int, r score.Result) {
	t.mu.Lock()
	cb := t.onNode
	t.mu.Unlock()
	if cb != nil {
		cb(done, total, r)
	}
}
