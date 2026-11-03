package lock

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAcquireExcludesSecondWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.lock")
	first, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(path); err == nil {
		t.Fatal("second acquire should fail")
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	second, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = second.Release()
}

func TestStaleLockIsReclaimed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.lock")
	if err := os.WriteFile(path, []byte("2147483646\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	l, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Release()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != strconv.Itoa(os.Getpid())+"\n" {
		t.Fatalf("lock pid = %q", data)
	}
}
