package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

const protocolVersion = "2024-11-05"

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Run serves the MCP stdio transport: one JSON-RPC object per line.
func Run(ctx context.Context, in io.Reader, out io.Writer, version string) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	writer := bufio.NewWriter(out)
	defer writer.Flush()

	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			writeResponse(writer, response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error: " + err.Error()}})
			continue
		}
		result, rpcErr := handle(ctx, req, version)
		if len(req.ID) == 0 {
			// Notification: no response.
			continue
		}
		resp := response{JSONRPC: "2.0", ID: req.ID}
		if rpcErr != nil {
			resp.Error = rpcErr
		} else {
			resp.Result = result
		}
		writeResponse(writer, resp)
	}
	return scanner.Err()
}

func writeResponse(w io.Writer, resp response) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = w.Write(append(data, '\n'))
}

func handle(ctx context.Context, req request, version string) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
			},
			"serverInfo": map[string]any{"name": "nodepulse", "version": version},
		}, nil
	case "notifications/initialized":
		return nil, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": toolList()}, nil
	case "resources/list":
		return map[string]any{"resources": resourceList()}, nil
	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params: " + err.Error()}
		}
		text, mime, err := readResource(params.URI)
		if err != nil {
			return nil, &rpcError{Code: -32602, Message: err.Error()}
		}
		return map[string]any{
			"contents": []map[string]string{{
				"uri":      params.URI,
				"mimeType": mime,
				"text":     text,
			}},
		}, nil
	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params: " + err.Error()}
		}
		text, err := callTool(ctx, params.Name, params.Arguments)
		if err != nil {
			return map[string]any{
				"content": []map[string]string{{"type": "text", "text": err.Error()}},
				"isError": true,
			}, nil
		}
		return map[string]any{
			"content": []map[string]string{{"type": "text", "text": text}},
			"isError": false,
		}, nil
	default:
		return nil, &rpcError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}
}
