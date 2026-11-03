package probecache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/probe"
)

func TestCacheRoundTripAndTTL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "probe-cache.json")
	c, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	key := Key("A", "example.com", "/file.apk")
	c.Put(key, probe.Result{Node: "A", OK: true, P25KiBps: 900, MeasuredAt: time.Now()})
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := loaded.Get(key)
	if !ok || !got.Cached || got.P25KiBps != 900 {
		t.Fatalf("got=%+v ok=%v", got, ok)
	}
	loaded.Put(key, probe.Result{Node: "A", OK: true, MeasuredAt: time.Now().Add(-DefaultTTL - time.Minute)})
	if _, ok := loaded.Get(key); ok {
		t.Fatal("expired success should miss")
	}
	failKey := Key("B", "example.com", "/file.apk")
	loaded.Put(failKey, probe.Result{OK: false, Err: "tls", MeasuredAt: time.Now().Add(-FailureTTL - time.Minute)})
	if _, ok := loaded.Get(failKey); ok {
		t.Fatal("expired failure should miss")
	}
}
