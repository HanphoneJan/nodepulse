package v2raya

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to the v2rayA REST API.
type Client struct {
	BaseURL string
	Token   string
	HC      *http.Client
}

// New creates a v2rayA client.
func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HC:      &http.Client{},
	}
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Login authenticates and stores the JWT token.
func (c *Client) Login(ctx context.Context, username, password string) error {
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var data struct {
		Token string `json:"token"`
	}
	if err := c.do(callCtx, http.MethodPost, "/api/login", map[string]string{
		"username": username,
		"password": password,
	}, &data); err != nil {
		return err
	}
	if data.Token == "" {
		return fmt.Errorf("v2rayA login returned an empty token")
	}
	c.Token = data.Token
	return nil
}

// Server is a direct server or a subscription server.
type Server struct {
	ID          int    `json:"id"`
	Type        string `json:"_type"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Net         string `json:"net"`
	PingLatency string `json:"pingLatency"`
}

// Subscription is one v2rayA subscription with its servers.
type Subscription struct {
	ID         int      `json:"id"`
	Type       string   `json:"_type"`
	Remarks    string   `json:"remarks"`
	Host       string   `json:"host"`
	Address    string   `json:"address"`
	Status     string   `json:"status"`
	AutoSelect bool     `json:"autoSelect"`
	Servers    []Server `json:"servers"`
}

// Ref identifies a node in v2rayA's tree.
type Ref struct {
	Type     string `json:"_type"`
	ID       int    `json:"id"`
	Sub      int    `json:"sub,omitempty"`
	Outbound string `json:"outbound,omitempty"`
	Selected bool   `json:"selected,omitempty"`
}

// Touch is the node tree returned by GET /api/touch.
type Touch struct {
	Running bool `json:"running"`
	Touch   struct {
		Servers          []Server       `json:"servers"`
		Subscriptions    []Subscription `json:"subscriptions"`
		ConnectedServers []Ref          `json:"connectedServer"`
	} `json:"touch"`
}

// Touch fetches the current node tree.
func (c *Client) Touch(ctx context.Context) (*Touch, error) {
	callCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var data Touch
	if err := c.do(callCtx, http.MethodGet, "/api/touch", nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// Node is a flattened entry for display / selection.
type Node struct {
	Name    string
	Address string
	Net     string
	Latency string
	Ref     Ref
	Group   string // "direct" or subscription remarks
}

// Nodes flattens the touch tree.
func (t *Touch) Nodes() []Node {
	var out []Node
	for _, s := range t.Touch.Servers {
		out = append(out, Node{
			Name: s.Name, Address: s.Address, Net: s.Net, Latency: s.PingLatency,
			Ref:   Ref{Type: "server", ID: s.ID},
			Group: "direct",
		})
	}
	for _, sub := range t.Touch.Subscriptions {
		for _, s := range sub.Servers {
			out = append(out, Node{
				Name: s.Name, Address: s.Address, Net: s.Net, Latency: s.PingLatency,
				Ref:   Ref{Type: "subscriptionServer", ID: s.ID, Sub: sub.ID},
				Group: sub.Remarks,
			})
		}
	}
	return out
}

// FindNode finds a node by exact name, then by case-insensitive substring.
func (t *Touch) FindNode(name string) (Node, bool) {
	nodes := t.Nodes()
	for _, n := range nodes {
		if n.Name == name {
			return n, true
		}
	}
	lower := strings.ToLower(name)
	for _, n := range nodes {
		if strings.Contains(strings.ToLower(n.Name), lower) {
			return n, true
		}
	}
	return Node{}, false
}

// Connect selects a node (v2rayA restarts its core).
func (c *Client) Connect(ctx context.Context, ref Ref) error {
	callCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	body := map[string]any{"_type": ref.Type, "id": ref.ID}
	if ref.Sub > 0 {
		body["sub"] = ref.Sub
	}
	return c.do(callCtx, http.MethodPost, "/api/connection", body, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HC.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("v2rayA %s %s: HTTP %d: %s", method, path, resp.StatusCode, truncate(string(data), 200))
	}
	if resp.StatusCode >= 400 || (env.Code != 0 && env.Code != 200) {
		return fmt.Errorf("v2rayA %s %s: code=%d message=%s", method, path, env.Code, env.Message)
	}
	if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("v2rayA %s %s: decode data: %w", method, path, err)
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// URL builds a v2rayA base URL from host and port.
func URL(host string, port int) string {
	return (&url.URL{Scheme: "http", Host: fmt.Sprintf("%s:%d", host, port)}).String()
}
