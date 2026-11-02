# NodePulse

**Beyond latency: measure proxy node stability, rank nodes, and keep the good ones.**

NodePulse 是一个与客户端解耦的代理节点稳定性工具。它不只看一次延迟，而是对每个节点做多轮真实请求，统计**成功率、P50/P90 延迟、抖动**，再给出稳定性评分和过滤后的配置。

> 当前状态：**Phase 1（CLI 核心）已实现**。GUI、守护进程、MCP 和 v2rayA/v2rayN 适配在路线图中。

## 为什么

FlClash、Clash Verge、v2rayN 等客户端的“延迟”通常只是一次 URL 测试：

- 延迟 80ms 的节点可能 5 次里挂 3 次；
- 延迟 300ms 的节点可能 10/10 稳定；
- 节点列表里很多名字其实是同一个服务器，换名字不换线路。

NodePulse 解决的是“**哪个节点真的稳**”，而不是“哪个节点这一次最快”。

## 功能

- 解析 Clash / mihomo 订阅配置（YAML）；
- 解析 base64 分享链接订阅（vmess / vless / trojan / ss / hysteria2）；
- 内置 mihomo sidecar，用订阅在本地随机端口跑测试，不依赖具体客户端；
- 也支持直接连接已有的 Clash REST 控制口（FlClash / Clash Verge / Mihomo Party / sing-box）；
- 多轮真实代理请求，统计成功率、P50、P90、最大值、抖动；
- 稳定性评分与排序；
- 导出只包含稳定节点的 Clash 配置，或 v2rayN 可导入的 base64 订阅；
- 守护进程：定时测试 + 滞后阈值 + 自动切换 + 失败回滚；
- Wails 桌面 GUI（Material Design 3，明暗主题、导航栏、一键优化）；
- MCP server（通用 agent 入口）与 dsh 插件；
- v2rayA 适配（状态 / 切换）。

## 快速开始

### 依赖

- Go 1.24+（构建时）
- 一个 `mihomo` 可执行文件（sidecar 模式需要）；如果你的客户端已经开了控制口，可以不用 sidecar

### 构建

```bash
make build
# 或
go build -o bin/nodepulse ./cmd/nodepulse

# GUI（需要 GTK3 + WebKit2GTK 4.1 开发库，带系统托盘）
make gui
./bin/nodepulse-gui
# 或者一步构建并启动
make run

# 或者浏览器里的本地 Web UI
make web
# 等价于 ./bin/nodepulse serve --addr 127.0.0.1:8099
```

### GUI 一键使用

```bash
make run    # 构建并启动桌面版（带托盘）
```

打开后按 4 步走：

1. **连接客户端**：自动检测 FlClash / Clash Verge / Mihomo Party 的本地控制口，点一下即可；
2. **选择订阅**：自动列出本机订阅文件，或直接粘贴订阅 URL 导入；
3. **一键优化**：多轮测速后自动切换到最稳节点；
4. **后台自动优化**：按间隔持续测速，节点变差时自动切换并回滚。

高级设置里保留了控制口、secret、Selector 组、mihomo 路径、并发/超时等手动项。

本地 Web UI：

```bash
make web    # http://127.0.0.1:8099
```

### CLI 使用

### 1. 准备订阅配置

NodePulse 直接读取 Clash / mihomo 格式的 YAML（FlClash、Clash Verge 导出的配置都可以）：

```bash
nodepulse nodes -f ~/my-subscription.yaml
```

### 2. 测速

**方式 A：使用内置 mihomo sidecar（推荐，零客户端配置）**

```bash
nodepulse test \
  -f ~/my-subscription.yaml \
  --mihomo /usr/local/bin/mihomo \
  --rounds 10 \
  --out results.json
```

**方式 B：复用客户端已有的控制口**

```bash
nodepulse test \
  -f ~/my-subscription.yaml \
  --api http://127.0.0.1:9090 \
  --secret your-secret \
  --rounds 10 \
  --out results.json
```

### 3. 查看排名

```bash
nodepulse rank -i results.json
```

输出包含成功率、P50/P90、抖动和综合评分，按稳定性排序。

### 4. 导出稳定节点配置

```bash
nodepulse export \
  -f ~/my-subscription.yaml \
  -i results.json \
  -o stable.yaml \
  --min-success 0.8 \
  --max-p90 800 \
  --max-jitter 250
```

把 `stable.yaml` 导入客户端即可，里面只保留通过阈值的节点，并自动生成一个 `Auto`（url-test）组。

导出 v2rayN / 通用 base64 订阅：

```bash
nodepulse export -f sub.yaml -i results.json -o stable.txt --format v2rayn
```

## 命令

| 命令 | 说明 |
|---|---|
| `nodepulse nodes -f <config>` | 列出可用节点（默认跳过“剩余流量/套餐到期”等信息节点） |
| `nodepulse test -f <config>` | 多轮测速并输出 JSON |
| `nodepulse rank -i <results.json>` | 按稳定性排名 |
| `nodepulse export -f <config> -i <results.json>` | 导出过滤后的 Clash 配置 |
| `nodepulse daemon --api <url>` | 定时测速并自动切换节点 |
| `nodepulse status --api <url>` | 查看控制口里的组和当前节点 |
| `nodepulse switch --api <url> --group <组> --node <节点>` | 切换 Selector 组 |
| `nodepulse v2raya login\|status\|switch` | v2rayA 适配 |
| `nodepulse mcp` | 启动 MCP stdio server |
| `nodepulse serve` | 启动本地 Web UI（默认 127.0.0.1:8099） |
| `nodepulse history` | 查看切换历史 |
| `nodepulse version` | 版本信息 |

`test` 的常用参数：

| 参数 | 默认 | 说明 |
|---|---|---|
| `--rounds` | `5` | 每个节点测试轮数 |
| `--url` | `https://www.gstatic.com/generate_204` | 测试 URL |
| `--timeout` | `5s` | 单轮超时 |
| `--concurrency` | `8` | 并发节点数 |
| `--skip-info` | `true` | 跳过信息伪节点 |

## 自动切换（Phase 2）

前提：客户端开启了 Clash 控制口（见适配矩阵）。

```bash
nodepulse daemon \
  --api http://127.0.0.1:9090 \
  --group "🚀节点选择" \
  --interval 5m \
  --rounds 3 \
  --margin 0.2 \
  --confirm 2
```

策略：

- 当前节点成功率低于 `--min-success`（默认 0.8）→ 立即切换；
- 候选节点评分需比当前高 `--margin`（默认 20%）；
- 连续 `--confirm`（默认 2）轮占优才切换，避免抖动；
- 切换后验证，失败自动回滚；
- `--dry-run` 只看决策；`--once` 只跑一轮。

历史写入 `~/.local/share/nodepulse/history.jsonl`，用 `nodepulse history` 查看。

## 评分

```
success = 成功轮数 / 总轮数
latencyScore = clamp(1 - (P90 - 100ms) / 900ms, 0, 1)
stabilityScore = clamp(1 - jitter / 500ms, 0, 1)
score = 0.5 * success + 0.3 * latencyScore + 0.2 * stabilityScore
```

成功率低于 80% 的节点会被降权。详细定义见 [docs/design.md](docs/design.md)。

## 客户端适配

| 客户端 | 测速 | 实时切换 | 方式 |
|---|---|---|---|
| FlClash | 支持 | 需开启控制口 | Clash REST API |
| Clash Verge Rev | 支持 | 支持 | Clash REST API |
| Mihomo Party | 支持 | 支持 | Clash REST API |
| sing-box GUI | 支持 | 支持 | `clash_api` |
| v2rayA | 支持 | 支持 | v2rayA REST API |
| v2rayN | 支持（导出） | 受限 | Clash 配置 / base64 订阅导入 |
| 任意订阅 URL | 支持 | — | 内置 mihomo sidecar |

详见 [docs/adapter-matrix.md](docs/adapter-matrix.md)。

## 文档

- [架构](docs/architecture.md)
- [设计：测试协议与评分](docs/design.md)
- [UI 设计：Material Design 3](docs/ui.md)
- [路线图](docs/roadmap.md)
- [客户端适配矩阵](docs/adapter-matrix.md)
- [开发与发布](docs/development.md)

## 路线图

- [x] Phase 1：核心引擎 + CLI
- [x] Phase 2：Clash 适配 + 守护/自动切换
- [x] Phase 3：Wails GUI + 打包配置（托盘/本地 Web UI 后续）
- [x] Phase 4：MCP server + dsh 插件
- [x] Phase 5：base64 订阅 + v2rayA 适配 + v2rayN 导出

## License

Apache-2.0
