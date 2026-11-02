package clash

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to a Clash-compatible REST controller (mihomo, Clash Verge,
// Mihomo Party, sing-box clash_api).
type Client struct {
	baseURL string
	secret  string
	hc      *http.Client
}

// New creates a client. baseURL is like http://127.0.0.1:9090.
func New(baseURL, secret string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		secret:  secret,
		hc:      &http.Client{},
	}
}

// Version returns the controller version string.
func (c *Client) Version(ctx context.Context) (string, error) {
	var out struct {
		Version string `json:"version"`
	}
	if err := c.get(ctx, "/version", &out); err != nil {
		return "", err
	}
	return out.Version, nil
}

// ProxyInfo is the subset of /proxies we care about.
type ProxyInfo struct {
	Type string   `json:"type"`
	Now  string   `json:"now"`
	All  []string `json:"all"`
}

// Proxies lists every proxy and group known to the controller.
func (c *Client) Proxies(ctx context.Context) (map[string]ProxyInfo, error) {
	var out struct {
		Proxies map[string]ProxyInfo `json:"proxies"`
	}
	if err := c.get(ctx, "/proxies", &out); err != nil {
		return nil, err
	}
	return out.Proxies, nil
}

// Delay measures one request through the named proxy.
func (c *Client) Delay(ctx context.Context, name, testURL string, timeout time.Duration) (int, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout+3*time.Second)
	defer cancel()

	endpoint := fmt.Sprintf("%s/proxies/%s/delay?timeout=%d&url=%s",
		c.baseURL,
		url.PathEscape(name),
		timeout.Milliseconds(),
		url.QueryEscape(testURL),
	)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	c.authorize(req)

	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("delay %q: HTTP %d: %s", name, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out struct {
		Delay   int    `json:"delay"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, fmt.Errorf("delay %q: decode response: %w", name, err)
	}
	if out.Message != "" {
		return 0, fmt.Errorf("delay %q: %s", name, out.Message)
	}
	if out.Delay <= 0 {
		return 0, fmt.Errorf("delay %q: empty delay", name)
	}
	return out.Delay, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("GET %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("GET %s: decode: %w", path, err)
	}
	return nil
}

func (c *Client) authorize(req *http.Request) {
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
}
