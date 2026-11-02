package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runMCP(t *testing.T, input string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := Run(context.Background(), strings.NewReader(input), &out, "test-version"); err != nil {
		t.Fatal(err)
	}
	var responses []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("decode %q: %v", line, err)
		}
		responses = append(responses, m)
	}
	return responses
}

func TestInitializeAndToolsList(t *testing.T) {
	resp := runMCP(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
`)
	if len(resp) != 2 {
		t.Fatalf("responses = %d, want 2 (notification must not reply)", len(resp))
	}
	init := resp[0]["result"].(map[string]any)
	if init["protocolVersion"] != protocolVersion {
		t.Fatalf("protocolVersion = %v", init["protocolVersion"])
	}
	tools := resp[1]["result"].(map[string]any)["tools"].([]any)
	if len(tools) == 0 {
		t.Fatal("no tools listed")
	}
}

func TestCallNodesList(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte(`
proxies:
  - {name: "Node A", type: ss, server: 127.0.0.1, port: 1}
  - {name: "剩余流量", type: ss, server: 127.0.0.1, port: 2}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nodes.list","arguments":{"config_path":` + quote(cfg) + `}}}` + "\n"
	resp := runMCP(t, input)
	result := resp[0]["result"].(map[string]any)
	content := result["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(content, "Node A") {
		t.Fatalf("content = %q", content)
	}
	if result["isError"].(bool) {
		t.Fatalf("unexpected error: %v", content)
	}
}

func TestUnknownMethod(t *testing.T) {
	resp := runMCP(t, `{"jsonrpc":"2.0","id":1,"method":"nope"}`+"\n")
	errObj := resp[0]["error"].(map[string]any)
	if errObj["code"].(float64) != -32601 {
		t.Fatalf("error = %v", errObj)
	}
}

func quote(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}

func TestResources(t *testing.T) {
	resp := runMCP(t, `{"jsonrpc":"2.0","id":1,"method":"resources/list"}`+"\n")
	resources := resp[0]["result"].(map[string]any)["resources"].([]any)
	if len(resources) != 2 {
		t.Fatalf("resources = %d, want 2", len(resources))
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "results.json")
	if err := os.WriteFile(path, []byte(`{"results":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NODEPULSE_RESULTS", path)
	resp = runMCP(t, `{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"nodepulse://results"}}`+"\n")
	contents := resp[0]["result"].(map[string]any)["contents"].([]any)
	text := contents[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "results") {
		t.Fatalf("text = %q", text)
	}
}
