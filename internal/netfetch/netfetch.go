// Package netfetch downloads subscription URLs with explicit limits. NodePulse
// allows LAN and self-hosted endpoints, but never link-local/metadata targets.
package netfetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// MaxBytes is the largest subscription body NodePulse accepts.
	MaxBytes = 10 << 20
	// MaxRedirects keeps redirect chains short and auditable.
	MaxRedirects = 5
)

// Download fetches a subscription URL after validating every hop.
func Download(ctx context.Context, rawURL string) ([]byte, error) {
	parsed, err := ValidateURL(rawURL)
	if err != nil {
		return nil, err
	}
	// Resolve named hosts up front as well; the DialContext check is skipped
	// when the request goes through an HTTP proxy.
	if net.ParseIP(parsed.Hostname()) == nil {
		if ips, err := net.DefaultResolver.LookupIPAddr(ctx, parsed.Hostname()); err == nil {
			for _, candidate := range ips {
				if err := checkIP(candidate.IP); err != nil {
					return nil, err
				}
			}
		}
	}
	client := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				for _, candidate := range ips {
					if err := checkIP(candidate.IP); err != nil {
						return nil, err
					}
				}
				if len(ips) == 0 {
					return nil, fmt.Errorf("无法解析 %s", host)
				}
				dialer := &net.Dialer{Timeout: 10 * time.Second}
				return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= MaxRedirects {
				return fmt.Errorf("重定向次数过多（最多 %d 次）", MaxRedirects)
			}
			if _, err := ValidateURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "NodePulse/subscription")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("订阅内容超过 %d MiB 限制", MaxBytes>>20)
	}
	return data, nil
}

// ValidateURL checks scheme, credentials and literal/metadata hosts.
func ValidateURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return nil, errors.New("订阅 URL 无效")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("订阅 URL 必须是 http 或 https")
	}
	if parsed.User != nil {
		return nil, errors.New("订阅 URL 不能包含用户名或密码")
	}
	host := parsed.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if err := checkIP(ip); err != nil {
			return nil, err
		}
	} else if host == "" {
		return nil, errors.New("订阅 URL 缺少主机名")
	}
	return parsed, nil
}

func checkIP(ip net.IP) error {
	if ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("拒绝访问链路本地/元数据地址 %s", ip)
	}
	// Common cloud metadata endpoints outside link-local space.
	if ip.Equal(net.ParseIP("100.100.100.200")) {
		return fmt.Errorf("拒绝访问云元数据地址 %s", ip)
	}
	return nil
}
