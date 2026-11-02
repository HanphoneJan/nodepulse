// Package progress tracks a single long-running test/optimization job so the
// GUI and the local web UI can show the same progress and cancel it.
package progress

import (
	"context"
	"sync"
	"time"
)

// Snapshot is safe to encode as JSON.
type Snapshot struct {
	Running   bool      `json:"running"`
	Kind      string    `json:"kind,omitempty"`
	Done      int       `json:"done"`
	Total     int       `json:"total"`
	Current   string    `json:"current,omitempty"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// Tracker is shared by one app/server instance.
type Tracker struct {
	mu     sync.Mutex
	snap   Snapshot
	cancel context.CancelFunc
}

func New() *Tracker {
	return &Tracker{}
}

// Begin starts a job. A previous cancel function is replaced.
func (t *Tracker) Begin(kind string, cancel context.CancelFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	t.cancel = cancel
	t.snap = Snapshot{
		Running:   true,
		Kind:      kind,
		StartedAt: now,
		UpdatedAt: now,
	}
}

// Update reports one completed node.
func (t *Tracker) Update(done, total int, current string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.snap.Running {
		return
	}
	t.snap.Done = done
	t.snap.Total = total
	t.snap.Current = current
	t.snap.UpdatedAt = time.Now()
}

// Finish ends the job. err is empty on success, or the cancellation/failure
// message shown by the UI.
func (t *Tracker) Finish(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.snap.Running && t.snap.Kind == "" {
		return
	}
	t.snap.Running = false
	t.snap.Current = ""
	t.snap.UpdatedAt = time.Now()
	if err != nil {
		t.snap.Error = err.Error()
	} else {
		t.snap.Error = ""
	}
	t.cancel = nil
}

// Cancel requests cancellation of the running job.
func (t *Tracker) Cancel() bool {
	t.mu.Lock()
	cancel := t.cancel
	t.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (t *Tracker) Snapshot() Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.snap
}
