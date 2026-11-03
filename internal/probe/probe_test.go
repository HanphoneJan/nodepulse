package probe

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMeasureRange(t *testing.T) {
	payload := make([]byte, 2<<20)
	for i := range payload {
		payload[i] = 'x'
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		start := 0
		if _, err := fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-", &start); err == nil && start > 0 {
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(payload[start:])
			return
		}
		_, _ = w.Write(payload)
	}))
	defer srv.Close()
	target, err := ParseTarget(srv.URL + "/file.apk?token=secret")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Measure(context.Background(), "", target, 1<<20, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.Bytes != 1<<20 || res.TTFBMs < 0 || res.P25KiBps <= 0 {
		t.Fatalf("result = %+v", res)
	}
}

func TestMeasureRejectsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()
	target, _ := ParseTarget(srv.URL + "/file.apk")
	res, err := Measure(context.Background(), "", target, 1<<20, time.Second)
	if err == nil || res.Status != http.StatusForbidden || res.OK {
		t.Fatalf("result=%+v err=%v", res, err)
	}
}

func TestParseTarget(t *testing.T) {
	if _, err := ParseTarget("file:///etc/passwd"); err == nil {
		t.Fatal("file URL should be rejected")
	}
	if _, err := ParseTarget("https://user:pass@example.com/a"); err == nil {
		t.Fatal("credentials should be rejected")
	}
	target, err := ParseTarget("https://example.com/path/file.apk?token=1")
	if err != nil || target.Host != "example.com" || target.Path != "/path/file.apk" {
		t.Fatalf("target=%+v err=%v", target, err)
	}
}
