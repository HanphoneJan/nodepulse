package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/export"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/sidecar"
	"github.com/HanphoneJan/nodepulse/internal/store"
	"github.com/HanphoneJan/nodepulse/internal/subscription"
	"github.com/HanphoneJan/nodepulse/internal/tester"
)

type toolDef struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     func(context.Context, json.RawMessage) (string, error)
}

func toolList() []map[string]any {
	defs := tools()
	out := make([]map[string]any, 0, len(defs))
	for _, d := range defs {
		out = append(out, map[string]any{
			"name":        d.Name,
			"description": d.Description,
			"inputSchema": d.InputSchema,
		})
	}
	return out
}

func tools() []toolDef {
	return []toolDef{
		{
			Name:        "nodes.list",
			Description: "List real proxy nodes from a Clash/mihomo YAML subscription.",
			InputSchema: objectSchema(map[string]any{
				"config_path": strProp("Path to the Clash/mihomo YAML subscription"),
			}, "config_path"),
			Handler: handleNodesList,
		},
		{
			Name:        "nodes.test",
			Description: "Run multi-round stability tests and return a ranked table. Uses an existing Clash controller (api) or a temporary mihomo sidecar (mihomo_path).",
			InputSchema: objectSchema(map[string]any{
				"config_path": strProp("Subscription YAML path; required"),
				"api":         strProp("Existing Clash controller URL, e.g. http://127.0.0.1:9090"),
				"secret":      strProp("Clash controller secret"),
				"mihomo_path": strProp("mihomo executable for sidecar mode"),
				"rounds":      intProp("Rounds per node (default 5)"),
				"concurrency": intProp("Concurrent nodes (default 8)"),
				"url":         strProp("Test URL (default gstatic generate_204)"),
				"timeout_ms":  intProp("Per-round timeout in milliseconds (default 5000)"),
			}, "config_path"),
			Handler: handleNodesTest,
		},
		{
			Name:        "nodes.rank",
			Description: "Rank a previously saved results.json by stability.",
			InputSchema: objectSchema(map[string]any{
				"results_path": strProp("Path to results.json"),
			}, "results_path"),
			Handler: handleNodesRank,
		},
		{
			Name:        "nodes.switch",
			Description: "Select a node in a Clash Selector group via the REST controller.",
			InputSchema: objectSchema(map[string]any{
				"api":    strProp("Clash controller URL"),
				"secret": strProp("Clash controller secret"),
				"group":  strProp("Selector group name"),
				"node":   strProp("Node name"),
			}, "api", "group", "node"),
			Handler: handleNodesSwitch,
		},
		{
			Name:        "export.stable",
			Description: "Write a filtered Clash config containing only nodes that pass the stability thresholds.",
			InputSchema: objectSchema(map[string]any{
				"config_path":   strProp("Original subscription YAML"),
				"results_path":  strProp("results.json to filter by"),
				"out":           strProp("Output YAML path (default stable.yaml)"),
				"min_success":   numProp("Minimum success rate 0..1 (default 0.8)"),
				"max_p90_ms":    intProp("Maximum P90 in ms (0 = no limit)"),
				"max_jitter_ms": intProp("Maximum jitter in ms (0 = no limit)"),
			}, "config_path", "results_path"),
			Handler: handleExportStable,
		},
		{
			Name:        "status.groups",
			Description: "List groups and the currently selected node from a Clash controller.",
			InputSchema: objectSchema(map[string]any{
				"api":    strProp("Clash controller URL"),
				"secret": strProp("Clash controller secret"),
			}, "api"),
			Handler: handleStatusGroups,
		},
		{
			Name:        "history.recent",
			Description: "Show recent NodePulse daemon events (switches, rollbacks, errors).",
			InputSchema: objectSchema(map[string]any{
				"path": strProp("History JSONL path; default ~/.local/share/nodepulse/history.jsonl"),
				"n":    intProp("Number of events (default 20)"),
			}),
			Handler: handleHistoryRecent,
		},
	}
}

func callTool(ctx context.Context, name string, raw json.RawMessage) (string, error) {
	for _, d := range tools() {
		if d.Name == name {
			return d.Handler(ctx, raw)
		}
	}
	return "", fmt.Errorf("unknown tool: %s", name)
}

type listArgs struct {
	ConfigPath string `json:"config_path"`
}

func handleNodesList(_ context.Context, raw json.RawMessage) (string, error) {
	var args listArgs
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if args.ConfigPath == "" {
		return "", fmt.Errorf("config_path is required")
	}
	cfg, err := subscription.Load(args.ConfigPath)
	if err != nil {
		return "", err
	}
	nodes := cfg.RealProxies()
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tSERVER\tINFO")
	for _, n := range nodes {
		fmt.Fprintf(w, "%s\t%s\t%s\t%t\n", n.Name, n.Type, serverOf(n), config.IsInfoNode(n.Name))
	}
	_ = w.Flush()
	fmt.Fprintf(&b, "\n%d nodes\n", len(nodes))
	return b.String(), nil
}

type testArgs struct {
	ConfigPath  string `json:"config_path"`
	API         string `json:"api"`
	Secret      string `json:"secret"`
	MihomoPath  string `json:"mihomo_path"`
	Rounds      int    `json:"rounds"`
	Concurrency int    `json:"concurrency"`
	URL         string `json:"url"`
	TimeoutMs   int    `json:"timeout_ms"`
}

func handleNodesTest(ctx context.Context, raw json.RawMessage) (string, error) {
	var args testArgs
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if args.ConfigPath == "" {
		return "", fmt.Errorf("config_path is required")
	}
	cfg, err := subscription.Load(args.ConfigPath)
	if err != nil {
		return "", err
	}
	nodes := make([]config.Proxy, 0, len(cfg.Proxies))
	for _, n := range cfg.RealProxies() {
		if !config.IsInfoNode(n.Name) {
			nodes = append(nodes, n)
		}
	}
	if len(nodes) == 0 {
		return "", fmt.Errorf("no testable nodes")
	}

	var delayer tester.Delayer
	var cleanup func()
	if args.API != "" {
		delayer = clash.New(args.API, args.Secret)
	} else {
		if args.MihomoPath == "" {
			return "", fmt.Errorf("api or mihomo_path is required")
		}
		sc, err := sidecar.Start(ctx, cfg, args.MihomoPath)
		if err != nil {
			return "", err
		}
		delayer = clash.New(sc.BaseURL, sc.Secret)
		cleanup = func() { _ = sc.Close() }
	}
	if cleanup != nil {
		defer cleanup()
	}

	runCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	t := tester.New(delayer, tester.Options{
		Rounds:      args.Rounds,
		Concurrency: args.Concurrency,
		URL:         args.URL,
		Timeout:     time.Duration(args.TimeoutMs) * time.Millisecond,
	})
	results := score.Rank(t.Test(runCtx, nodes))
	return resultTable(results), nil
}

type rankArgs struct {
	ResultsPath string `json:"results_path"`
}

func handleNodesRank(_ context.Context, raw json.RawMessage) (string, error) {
	var args rankArgs
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	rf, err := readResults(args.ResultsPath)
	if err != nil {
		return "", err
	}
	return resultTable(score.Rank(rf.Results)), nil
}

type switchArgs struct {
	API    string `json:"api"`
	Secret string `json:"secret"`
	Group  string `json:"group"`
	Node   string `json:"node"`
}

func handleNodesSwitch(ctx context.Context, raw json.RawMessage) (string, error) {
	var args switchArgs
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if args.API == "" || args.Group == "" || args.Node == "" {
		return "", fmt.Errorf("api, group and node are required")
	}
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := clash.New(args.API, args.Secret).Switch(callCtx, args.Group, args.Node); err != nil {
		return "", err
	}
	return fmt.Sprintf("switched %s -> %s", args.Group, args.Node), nil
}

type exportArgs struct {
	ConfigPath  string  `json:"config_path"`
	ResultsPath string  `json:"results_path"`
	Out         string  `json:"out"`
	MinSuccess  float64 `json:"min_success"`
	MaxP90Ms    int     `json:"max_p90_ms"`
	MaxJitterMs int     `json:"max_jitter_ms"`
}

func handleExportStable(_ context.Context, raw json.RawMessage) (string, error) {
	var args exportArgs
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if args.ConfigPath == "" || args.ResultsPath == "" {
		return "", fmt.Errorf("config_path and results_path are required")
	}
	cfg, err := subscription.Load(args.ConfigPath)
	if err != nil {
		return "", err
	}
	rf, err := readResults(args.ResultsPath)
	if err != nil {
		return "", err
	}
	out := args.Out
	if out == "" {
		out = "stable.yaml"
	}
	sel, err := export.Stable(cfg, rf.Results, export.Options{
		MinSuccess:  args.MinSuccess,
		MaxP90Ms:    args.MaxP90Ms,
		MaxJitterMs: args.MaxJitterMs,
	})
	if err != nil {
		return "", err
	}
	if err := sel.Config.Write(out); err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote %s: kept %d, excluded %d", out, len(sel.Included), len(sel.Excluded)), nil
}

type statusArgs struct {
	API    string `json:"api"`
	Secret string `json:"secret"`
}

func handleStatusGroups(ctx context.Context, raw json.RawMessage) (string, error) {
	var args statusArgs
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if args.API == "" {
		return "", fmt.Errorf("api is required")
	}
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	proxies, err := clash.New(args.API, args.Secret).Proxies(callCtx)
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(proxies))
	for name, info := range proxies {
		if isGroup(info.Type) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "GROUP\tTYPE\tNOW\tMEMBERS")
	for _, name := range names {
		info := proxies[name]
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", name, info.Type, info.Now, len(info.All))
	}
	_ = w.Flush()
	return b.String(), nil
}

type historyArgs struct {
	Path string `json:"path"`
	N    int    `json:"n"`
}

func handleHistoryRecent(_ context.Context, raw json.RawMessage) (string, error) {
	var args historyArgs
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	path := args.Path
	if path == "" {
		path = store.DefaultPath()
	}
	st, err := store.New(path)
	if err != nil {
		return "", err
	}
	events, err := st.Recent(args.N)
	if err != nil {
		return "", err
	}
	if len(events) == 0 {
		return "no history", nil
	}
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "TIME\tKIND\tFROM\tTO\tREASON")
	for _, e := range events {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.Time.Format("01-02 15:04:05"), e.Kind, e.From, e.To, e.Reason)
	}
	_ = w.Flush()
	return b.String(), nil
}

type resultsFile struct {
	Results []score.Result `json:"results"`
}

func readResults(path string) (*resultsFile, error) {
	if path == "" {
		return nil, fmt.Errorf("results_path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rf resultsFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return nil, err
	}
	if len(rf.Results) == 0 {
		return nil, fmt.Errorf("results file has no nodes")
	}
	return &rf, nil
}

func resultTable(results []score.Result) string {
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "RANK\tNODE\tOK\tP50\tP90\tJITTER\tSCORE")
	for i, r := range results {
		fmt.Fprintf(w, "%d\t%s\t%d/%d\t%dms\t%dms\t%dms\t%.3f\n",
			i+1, r.Name, r.Success, r.Total, r.MedianMs, r.P90Ms, r.JitterMs, r.Score)
	}
	_ = w.Flush()
	return b.String()
}

func serverOf(p config.Proxy) string {
	if p.Port > 0 {
		return fmt.Sprintf("%s:%d", p.Server, p.Port)
	}
	return p.Server
}

func isGroup(t string) bool {
	switch strings.ToLower(t) {
	case "selector", "urltest", "fallback", "loadbalance":
		return true
	}
	return false
}

func decode(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	return nil
}

func objectSchema(props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func strProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intProp(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func numProp(desc string) map[string]any {
	return map[string]any{"type": "number", "description": desc}
}
