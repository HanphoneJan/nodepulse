package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendAndRecent(t *testing.T) {
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "history.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := s.Append(Event{Time: time.Unix(int64(i), 0), Kind: "cycle", To: "node"}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := s.Recent(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	if events[1].Time.Unix() != 2 {
		t.Fatalf("last event time = %d", events[1].Time.Unix())
	}
	info, err := os.Stat(filepath.Join(dir, "history.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("history mode = %o, want 600", info.Mode().Perm())
	}
}
