package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/HanphoneJan/nodepulse/internal/clash"
	"github.com/HanphoneJan/nodepulse/internal/config"
	"github.com/HanphoneJan/nodepulse/internal/daemon"
	"github.com/HanphoneJan/nodepulse/internal/export"
	"github.com/HanphoneJan/nodepulse/internal/score"
	"github.com/HanphoneJan/nodepulse/internal/sidecar"
	"github.com/HanphoneJan/nodepulse/internal/store"
	"github.com/HanphoneJan/nodepulse/internal/subscription"
	"github.com/HanphoneJan/nodepulse/internal/tester"
)

const (
	defaultTestURL     = "https://www.gstatic.com/generate_204"
	defaultResultsFile = "results.json"
)

// Run dispatches a CLI invocation and returns the process exit code.
func Run(args []string, version string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "version", "-v", "--version":
		fmt.Printf("nodepulse %s\n", version)
		return 0
	case "help", "-h", "--help":
		usage()
		return 0
	case "nodes":
		return cmdNodes(args[1:])
	case "test":
		return cmdTest(args[1:])
	case "rank":
		return cmdRank(args[1:])
	case "export":
		return cmdExport(args[1:])
	case "daemon":
		return cmdDaemon(args[1:])
	case "status":
		return cmdStatus(args[1:])
	case "history":
		return cmdHistory(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知命令 %q\n\n", args[0])
		usage()
		return 2
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `NodePulse - 代理节点稳定性工具

用法:
  nodepulse nodes  -f <config.yaml>
  nodepulse test   -f <config.yaml> [--api URL | --mihomo PATH] [options]
  nodepulse rank   -i <results.json>
  nodepulse export -f <config.yaml> -i <results.json> -o <stable.yaml>
  nodepulse daemon --api http://127.0.0.1:9090 [options]
  nodepulse status --api http://127.0.0.1:9090
  nodepulse history [-n 20]

命令:
  nodes    列出节点
  test     多轮测速并输出 JSON
  rank     按稳定性排名
  export   过滤并导出 Clash 配置
  daemon   定时测速并自动切换节点
  status   查看客户端当前节点
  history  查看切换历史
  version  版本

运行 "nodepulse <command> -h" 查看该命令的参数。
`)
}

func cmdNodes(args []string) int {
	fs := flag.NewFlagSet("nodes", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("f", "", "订阅配置文件（Clash / mihomo YAML）")
	skipInfo := fs.Bool("skip-info", true, "跳过“剩余流量/套餐到期”等信息节点")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "缺少 -f <config.yaml>")
		return 2
	}
	cfg, err := subscription.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取失败:", err)
		return 1
	}

	nodes := cfg.RealProxies()
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tSERVER\tFLAGS")
	shown := 0
	for _, p := range nodes {
		info := config.IsInfoNode(p.Name)
		if info && *skipInfo {
			continue
		}
		flags := ""
		if info {
			flags = "info"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.Type, serverOf(p), flags)
		shown++
	}
	w.Flush()
	fmt.Fprintf(os.Stderr, "\n共 %d 个真实节点，显示 %d 个\n", len(nodes), shown)
	return 0
}

func cmdTest(args []string) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("f", "", "订阅配置文件（Clash / mihomo YAML）")
	api := fs.String("api", "", "已有 Clash 控制口，如 http://127.0.0.1:9090")
	secret := fs.String("secret", "", "Clash 控制口 secret")
	mihomoPath := fs.String("mihomo", "", "mihomo 可执行文件路径（sidecar 模式）")
	rounds := fs.Int("rounds", 5, "每个节点的测试轮数")
	testURL := fs.String("url", defaultTestURL, "测试 URL")
	timeout := fs.Duration("timeout", 5*time.Second, "单轮超时")
	concurrency := fs.Int("concurrency", 8, "并发节点数")
	out := fs.String("out", defaultResultsFile, "结果 JSON 输出路径")
	skipInfo := fs.Bool("skip-info", true, "跳过信息伪节点")
	noRaw := fs.Bool("no-raw", false, "结果中不保留原始延迟样本")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "缺少 -f <config.yaml>")
		return 2
	}
	cfg, err := subscription.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取失败:", err)
		return 1
	}

	nodes := cfg.RealProxies()
	if *skipInfo {
		filtered := nodes[:0]
		for _, n := range nodes {
			if !config.IsInfoNode(n.Name) {
				filtered = append(filtered, n)
			}
		}
		nodes = filtered
	}
	if len(nodes) == 0 {
		fmt.Fprintln(os.Stderr, "没有可测试的节点")
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var delayer tester.Delayer
	source := ""
	if *api != "" {
		delayer = clash.New(*api, *secret)
		source = *api
	} else {
		path, err := resolveMihomo(*mihomoPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		sc, err := sidecar.Start(ctx, cfg, path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "启动 sidecar 失败:", err)
			fmt.Fprintln(os.Stderr, "提示: 也可以先用客户端开启 external-controller，再用 --api 接入")
			return 1
		}
		defer sc.Close()
		delayer = clash.New(sc.BaseURL, sc.Secret)
		source = "sidecar (" + path + ")"
	}

	fmt.Fprintf(os.Stderr, "测试 %d 个节点 × %d 轮，通过 %s\n\n", len(nodes), *rounds, source)
	t := tester.New(delayer, tester.Options{
		Rounds:      *rounds,
		Concurrency: *concurrency,
		URL:         *testURL,
		Timeout:     *timeout,
	})
	t.OnNode(func(done, total int, r score.Result) {
		mark := "✗"
		if r.Success > 0 {
			mark = "✓"
		}
		fmt.Fprintf(os.Stderr, "[%2d/%2d] %s %-36s %2d/%2d  p50=%4dms p90=%4dms jitter=%3dms score=%.3f\n",
			done, total, mark, r.Name, r.Success, r.Total, r.MedianMs, r.P90Ms, r.JitterMs, r.Score)
	})

	results := t.Test(ctx, nodes)
	if ctx.Err() != nil {
		fmt.Fprintln(os.Stderr, "\n已中断")
		return 130
	}

	ranked := score.Rank(results)
	fmt.Println()
	printTable(ranked)

	if *noRaw {
		for i := range results {
			results[i].Latencies = nil
		}
	}
	rf := ResultsFile{
		GeneratedAt: time.Now().Format(time.RFC3339),
		URL:         *testURL,
		Rounds:      *rounds,
		Results:     results,
	}
	data, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "序列化结果失败:", err)
		return 1
	}
	if err := os.WriteFile(*out, data, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "写入结果失败:", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "\n结果已写入 %s\n", *out)
	return 0
}

func cmdRank(args []string) int {
	fs := flag.NewFlagSet("rank", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	in := fs.String("i", defaultResultsFile, "结果 JSON 路径")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rf, err := readResults(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取结果失败:", err)
		return 1
	}
	fmt.Printf("生成时间: %s   测试 URL: %s   轮数: %d\n\n", rf.GeneratedAt, rf.URL, rf.Rounds)
	printTable(score.Rank(rf.Results))
	return 0
}

func cmdExport(args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("f", "", "原始订阅配置")
	in := fs.String("i", defaultResultsFile, "结果 JSON 路径")
	out := fs.String("o", "stable.yaml", "导出配置路径")
	minSuccess := fs.Float64("min-success", 0.8, "最低成功率（0..1）")
	maxP90 := fs.Int("max-p90", 0, "P90 上限（ms，0 表示不限制）")
	maxJitter := fs.Int("max-jitter", 0, "抖动上限（ms，0 表示不限制）")
	testURL := fs.String("url", defaultTestURL, "导出配置里的测试 URL")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "缺少 -f <config.yaml>")
		return 2
	}
	cfg, err := subscription.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取失败:", err)
		return 1
	}
	rf, err := readResults(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取结果失败:", err)
		return 1
	}

	sel, err := export.Stable(cfg, rf.Results, export.Options{
		MinSuccess:  *minSuccess,
		MaxP90Ms:    *maxP90,
		MaxJitterMs: *maxJitter,
		TestURL:     *testURL,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "导出失败:", err)
		return 1
	}
	if err := sel.Config.Write(*out); err != nil {
		fmt.Fprintln(os.Stderr, "写入失败:", err)
		return 1
	}
	fmt.Printf("已写入 %s：保留 %d 个节点，排除 %d 个\n", *out, len(sel.Included), len(sel.Excluded))
	if len(sel.Excluded) > 0 {
		fmt.Fprintln(os.Stderr, "排除:", strings.Join(sel.Excluded, "、"))
	}
	return 0
}

// ResultsFile is the on-disk schema shared by test/rank/export.
type ResultsFile struct {
	GeneratedAt string         `json:"generated_at"`
	URL         string         `json:"url"`
	Rounds      int            `json:"rounds"`
	Results     []score.Result `json:"results"`
}

func readResults(path string) (*ResultsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rf ResultsFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return nil, err
	}
	if len(rf.Results) == 0 {
		return nil, errors.New("结果文件里没有节点")
	}
	return &rf, nil
}

func printTable(results []score.Result) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "RANK\tNODE\tTYPE\tOK\tP50\tP90\tJITTER\tSCORE")
	for i, r := range results {
		fmt.Fprintf(w, "%d\t%s\t%s\t%d/%d\t%dms\t%dms\t%dms\t%.3f\n",
			i+1, r.Name, r.Type, r.Success, r.Total, r.MedianMs, r.P90Ms, r.JitterMs, r.Score)
	}
	w.Flush()
}

func resolveMihomo(explicit string) (string, error) {
	candidates := []string{
		explicit,
		os.Getenv("NODEPULSE_MIHOMO"),
		"mihomo",
		"/usr/local/bin/mihomo",
		"/usr/bin/mihomo",
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p, nil
		}
	}
	return "", errors.New("未找到 mihomo：用 --mihomo 指定，或设置 NODEPULSE_MIHOMO；也可以改用 --api 连接已有客户端")
}

func serverOf(p config.Proxy) string {
	if p.Port > 0 {
		return fmt.Sprintf("%s:%d", p.Server, p.Port)
	}
	return p.Server
}

func cmdDaemon(args []string) int {
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	api := fs.String("api", "", "Clash 控制口，如 http://127.0.0.1:9090")
	secret := fs.String("secret", "", "Clash 控制口 secret")
	group := fs.String("group", "", "要切换的 Selector 组（默认自动选择）")
	interval := fs.Duration("interval", 5*time.Minute, "测试间隔")
	rounds := fs.Int("rounds", 3, "每个节点测试轮数")
	concurrency := fs.Int("concurrency", 8, "并发节点数")
	testURL := fs.String("url", defaultTestURL, "测试 URL")
	timeout := fs.Duration("timeout", 5*time.Second, "单轮超时")
	margin := fs.Float64("margin", 0.2, "切换阈值：新节点需优于当前 20%")
	confirm := fs.Int("confirm", 2, "连续多少轮占优才切换")
	minSuccess := fs.Float64("min-success", 0.8, "当前节点低于此成功率时立即切换")
	once := fs.Bool("once", false, "只运行一轮")
	dryRun := fs.Bool("dry-run", false, "只输出决策，不实际切换")
	history := fs.String("history", "", "历史文件路径（默认 ~/.local/share/nodepulse/history.jsonl）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *api == "" {
		fmt.Fprintln(os.Stderr, "daemon 需要 --api <http://127.0.0.1:9090>（自动切换必须走客户端控制口）")
		return 2
	}

	client := clash.New(*api, *secret)
	historyPath := *history
	if historyPath == "" {
		historyPath = store.DefaultPath()
	}
	st, err := store.New(historyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "打开历史失败:", err)
		return 1
	}

	onEvent := func(e store.Event) {
		ts := e.Time.Format("15:04:05")
		switch e.Kind {
		case "switch":
			fmt.Fprintf(os.Stderr, "%s SWITCH %s -> %s  (%s)\n", ts, e.From, e.To, e.Reason)
		case "rollback":
			fmt.Fprintf(os.Stderr, "%s ROLLBACK %s -> %s  (%s)\n", ts, e.From, e.To, e.Reason)
		case "error":
			fmt.Fprintf(os.Stderr, "%s ERROR %s\n", ts, e.Reason)
		default:
			fmt.Fprintf(os.Stderr, "%s cycle %s: current=%s best=%s (%.2f vs %.2f)  %s\n",
				ts, e.Group, e.From, e.Best, e.BestScore, e.CurScore, e.Reason)
		}
	}

	d := daemon.New(client, daemon.Options{
		Group:       *group,
		Interval:    *interval,
		Rounds:      *rounds,
		Concurrency: *concurrency,
		TestURL:     *testURL,
		Timeout:     *timeout,
		Margin:      *margin,
		Confirm:     *confirm,
		MinSuccess:  *minSuccess,
		Once:        *once,
		DryRun:      *dryRun,
		Store:       st,
		OnEvent:     onEvent,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(os.Stderr, "守护进程启动：group=%s interval=%s rounds=%d margin=%.0f%% confirm=%d\n",
		orAuto(*group), *interval, *rounds, *margin*100, *confirm)
	if err := d.Run(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return 0
		}
		fmt.Fprintln(os.Stderr, "守护进程退出:", err)
		return 1
	}
	return 0
}

func orAuto(s string) string {
	if s == "" {
		return "auto"
	}
	return s
}

func cmdStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	api := fs.String("api", "", "Clash 控制口，如 http://127.0.0.1:9090")
	secret := fs.String("secret", "", "Clash 控制口 secret")
	group := fs.String("group", "", "只看某个组")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *api == "" {
		fmt.Fprintln(os.Stderr, "缺少 --api <http://127.0.0.1:9090>")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client := clash.New(*api, *secret)
	proxies, err := client.Proxies(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "查询失败:", err)
		return 1
	}

	keys := make([]string, 0, len(proxies))
	for name := range proxies {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "GROUP\tTYPE\tNOW\tMEMBERS")
	for _, name := range keys {
		info := proxies[name]
		if !isGroupType(info.Type) {
			continue
		}
		if *group != "" && name != *group {
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", name, info.Type, info.Now, len(info.All))
	}
	w.Flush()

	if *group != "" {
		info, ok := proxies[*group]
		if !ok {
			fmt.Fprintf(os.Stderr, "组 %q 不存在\n", *group)
			return 1
		}
		fmt.Printf("\n组 %s 的成员（当前: %s）:\n", *group, info.Now)
		for _, name := range info.All {
			mark := " "
			if name == info.Now {
				mark = "*"
			}
			fmt.Printf(" %s %s\n", mark, name)
		}
	}
	return 0
}

func isGroupType(t string) bool {
	switch strings.ToLower(t) {
	case "selector", "urltest", "fallback", "loadbalance":
		return true
	}
	return false
}

func cmdHistory(args []string) int {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	n := fs.Int("n", 20, "显示最近多少条")
	history := fs.String("history", "", "历史文件路径")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	path := *history
	if path == "" {
		path = store.DefaultPath()
	}
	st, err := store.New(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "打开历史失败:", err)
		return 1
	}
	events, err := st.Recent(*n)
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取历史失败:", err)
		return 1
	}
	if len(events) == 0 {
		fmt.Println("暂无历史记录")
		return 0
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "TIME\tKIND\tFROM\tTO\tREASON")
	for _, e := range events {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.Time.Format("01-02 15:04:05"), e.Kind, e.From, e.To, e.Reason)
	}
	w.Flush()
	return 0
}
