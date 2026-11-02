package progress

import (
	"context"
	"errors"
	"testing"
)

func TestTrackerLifecycle(t *testing.T) {
	tr := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Begin("test", cancel)
	tr.Update(2, 10, "A")
	snap := tr.Snapshot()
	if !snap.Running || snap.Done != 2 || snap.Total != 10 || snap.Current != "A" {
		t.Fatalf("snapshot = %+v", snap)
	}
	if !tr.Cancel() {
		t.Fatal("cancel should be available")
	}
	if ctx.Err() == nil {
		t.Fatal("context was not canceled")
	}
	tr.Finish(errors.New("context canceled"))
	snap = tr.Snapshot()
	if snap.Running || snap.Error != "context canceled" {
		t.Fatalf("finished snapshot = %+v", snap)
	}
	if tr.Cancel() {
		t.Fatal("cancel after finish should be false")
	}
}
