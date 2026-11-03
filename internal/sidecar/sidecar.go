package sidecar

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
)

// ProbeGroup is the selector group created by StartProbe.
const ProbeGroup = "NodePulseProbe"

// Sidecar is a temporary mihomo process used purely as a local test lab.
type Sidecar struct {
	BaseURL   string
	Secret    string
	MixedPort int

	cmd  *exec.Cmd
	dir  string
	logs *lockedBuffer
}

// Start launches mihomo with a minimal config derived from cfg. It binds only
// to loopback and uses random free ports so it never conflicts with a client
// already running on the machine.
func Start(ctx context.Context, cfg *config.Config, mihomoPath string) (*Sidecar, error) {
	return start(ctx, cfg.RealProxies(), mihomoPath, "")
}

// StartProbe starts a sidecar with a selector group so download-mode probes can
// switch between candidate nodes through one mixed-port.
func StartProbe(ctx context.Context, proxies []config.Proxy, mihomoPath string) (*Sidecar, error) {
	return start(ctx, proxies, mihomoPath, ProbeGroup)
}

func start(ctx context.Context, proxies []config.Proxy, mihomoPath, groupName string) (*Sidecar, error) {
	if len(proxies) == 0 {
		return nil, fmt.Errorf("no proxies for sidecar")
	}
	mixedPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("allocate mixed-port: %w", err)
	}
	controlPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("allocate controller port: %w", err)
	}
	secret, err := randomSecret()
	if err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}

	dir, err := os.MkdirTemp("", "nodepulse-sidecar-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	minimal := map[string]any{
		"mixed-port":          mixedPort,
		"allow-lan":           false,
		"mode":                "rule",
		"log-level":           "warning",
		"external-controller": fmt.Sprintf("127.0.0.1:%d", controlPort),
		"secret":              secret,
		"dns":                 map[string]any{"enable": false},
		"proxies":             proxies,
		"rules":               []string{"MATCH,DIRECT"},
	}
	if groupName != "" {
		names := make([]string, 0, len(proxies))
		for _, proxy := range proxies {
			names = append(names, proxy.Name)
		}
		minimal["proxy-groups"] = []map[string]any{{
			"name":    groupName,
			"type":    "select",
			"proxies": names,
		}}
		minimal["rules"] = []string{"MATCH," + groupName}
	}
	data, err := yaml.Marshal(minimal)
	if err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("marshal sidecar config: %w", err)
	}
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, data, 0o600); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("write sidecar config: %w", err)
	}

	logs := &lockedBuffer{}
	cmd := exec.Command(mihomoPath, "-f", cfgPath, "-d", dir)
	cmd.Stdout = logs
	cmd.Stderr = logs
	if err := cmd.Start(); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("start mihomo: %w", err)
	}

	s := &Sidecar{
		BaseURL:   fmt.Sprintf("http://127.0.0.1:%d", controlPort),
		Secret:    secret,
		MixedPort: mixedPort,
		cmd:       cmd,
		dir:       dir,
		logs:      logs,
	}

	if err := s.waitReady(ctx); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func (s *Sidecar) waitReady(ctx context.Context) error {
	client := clash.New(s.BaseURL, s.Secret)
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if _, err := client.Version(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("mihomo did not become ready: %v: %s", lastErr, s.logs.Last(2000))
}

// Close stops mihomo and removes the temporary directory.
func (s *Sidecar) Close() error {
	if s == nil {
		return nil
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- s.cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = s.cmd.Process.Kill()
			<-done
		}
	}
	if s.dir != "" {
		_ = os.RemoveAll(s.dir)
	}
	return nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func randomSecret() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) Last(n int) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	data := b.buf.Bytes()
	if len(data) > n {
		data = data[len(data)-n:]
	}
	return string(data)
}

// Select switches the probe selector group to a candidate node.
func (s *Sidecar) Select(ctx context.Context, node string) error {
	return clash.New(s.BaseURL, s.Secret).Switch(ctx, ProbeGroup, node)
}
