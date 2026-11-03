// Package probe measures a bounded HTTP Range download through a proxy. It is
// used by download mode to remove nodes that cannot reach a target and to
// compare sustained throughput instead of single-request latency.
package probe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Result is one bounded download probe.
type Result struct {
	Node        string    `json:"node"`
	TargetHost  string    `json:"target_host"`
	TargetPath  string    `json:"target_path"`
	OK          bool      `json:"ok"`
	Status      int       `json:"status,omitempty"`
	Err         string    `json:"error,omitempty"`
	Bytes       int64     `json:"bytes"`
	TTFBMs      int64     `json:"ttfb_ms"`
	ElapsedMs   int64     `json:"elapsed_ms"`
	P25KiBps    float64   `json:"p25_kibps"`
	MedianKiBps float64   `json:"median_kibps"`
	MeanKiBps   float64   `json:"mean_kibps"`
	Stalls      int       `json:"stalls"`
	Cached      bool      `json:"cached,omitempty"`
	MeasuredAt  time.Time `json:"measured_at,omitempty"`
}

// Target describes a validated probe target.
type Target struct {
	URL  string
	Host string
	Path string
}

// ParseTarget validates a user-supplied direct file URL.
func ParseTarget(raw string) (Target, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return Target{}, fmt.Errorf("目标 URL 无效")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return Target{}, fmt.Errorf("目标 URL 必须是 http 或 https")
	}
	if u.User != nil {
		return Target{}, fmt.Errorf("目标 URL 不能包含用户名或密码")
	}
	return Target{URL: raw, Host: u.Hostname(), Path: u.Path}, nil
}

// Measure performs one Range request through proxyAddr (host:port).
func Measure(ctx context.Context, proxyAddr string, target Target, maxBytes int64, maxDuration time.Duration) (Result, error) {
	res := Result{
		TargetHost: target.Host,
		TargetPath: target.Path,
		MeasuredAt: time.Now(),
	}
	if maxBytes <= 0 {
		maxBytes = 3 << 20
	}
	if maxDuration <= 0 {
		maxDuration = 5 * time.Second
	}
	transport := &http.Transport{}
	if proxyAddr != "" {
		proxyURL, err := url.Parse("http://" + proxyAddr)
		if err != nil {
			return res, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	client := &http.Client{Transport: transport}
	callCtx, cancel := context.WithTimeout(ctx, maxDuration+5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(callCtx, http.MethodGet, target.URL, nil)
	if err != nil {
		return res, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", maxBytes-1))
	req.Header.Set("User-Agent", "NodePulse/probe")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		res.Err = safeError(err)
		return res, fmt.Errorf("目标站点不可达: %s", res.Err)
	}
	defer resp.Body.Close()
	res.Status = resp.StatusCode
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		res.Err = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return res, fmt.Errorf("目标站点返回 HTTP %d", resp.StatusCode)
	}

	buf := make([]byte, 64<<10)
	var total int64
	var first time.Time
	lastChunk := start
	samples := make([]float64, 0, 256)
	for total < maxBytes {
		now := time.Now()
		if now.Sub(start) >= maxDuration {
			break
		}
		n, readErr := resp.Body.Read(buf)
		now = time.Now()
		if n > 0 {
			if first.IsZero() {
				first = now
				res.TTFBMs = now.Sub(start).Milliseconds()
			}
			gap := now.Sub(lastChunk)
			if gap > 2*time.Second {
				res.Stalls++
			}
			lastChunk = now
			total += int64(n)
			if gap > 50*time.Microsecond {
				samples = append(samples, float64(n)/gap.Seconds()/1024)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			if total == 0 {
				res.Err = safeError(readErr)
				return res, fmt.Errorf("读取目标失败: %s", res.Err)
			}
			break
		}
	}
	elapsed := time.Since(start)
	res.Bytes = total
	res.ElapsedMs = elapsed.Milliseconds()
	if total == 0 {
		res.Err = "没有收到数据"
		return res, fmt.Errorf("目标站点没有返回数据")
	}
	if elapsed > 0 {
		res.MeanKiBps = float64(total) / elapsed.Seconds() / 1024
	}
	if len(samples) == 0 {
		samples = append(samples, res.MeanKiBps)
	}
	sorted := append([]float64(nil), samples...)
	sort.Float64s(sorted)
	res.P25KiBps = percentile(sorted, 0.25)
	res.MedianKiBps = percentile(sorted, 0.5)
	res.OK = true
	return res, nil
}

// safeError removes the request URL, which may contain a short-lived signed
// token, from errors stored in probe results and caches.
func safeError(err error) string {
	if err == nil {
		return ""
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	msg := err.Error()
	if i := strings.IndexByte(msg, '?'); i >= 0 {
		msg = msg[:i]
	}
	return msg
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Floor(float64(len(sorted)-1) * p))
	return sorted[index]
}
