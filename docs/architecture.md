# 架构

## 目标

1. **客户端无关**：测速结果不依赖某个客户端；同一套数据可以给 FlClash、Clash Verge、v2rayN 用。
2. **可分发**：单二进制、`.deb`、AppImage，后续支持 Windows/macOS。
3. **可复用**：CLI、GUI、MCP 共用同一个核心引擎。
4. **可自动化**：守护进程按稳定性自动切换节点，带防抖。

## 非目标

- 不实现新的代理协议；
- 不替代客户端（NodePulse 不负责 TUN、DNS、流量转发）；
- 不分发节点或订阅；
- v1 不做 v2rayN 的强实时切换（见适配矩阵）。

## 总览

```
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│     CLI      │   │  Wails GUI   │   │  MCP server  │
└──────┬───────┘   └──────┬───────┘   └──────┬───────┘
       └──────────────┬───┴──────────────────┘
                 ┌────▼─────────────┐
                 │   core engine    │
                 │  test / score /  │
                 │  scheduler       │
                 └───┬──────────┬───┘
        ┌────────────┘          └────────────┐
   ┌────▼─────────┐                  ┌───────▼────────┐
   │  adapters    │                  │  subscription  │
   │ Clash REST   │                  │  YAML parser   │
   │ v2rayA       │                  └───────┬────────┘
   │ export       │                          │
   └──────────────┘                  ┌───────▼────────┐
                                     │ mihomo sidecar │
                                     │ local test lab │
                                     └────────────────┘
```

## 组件

### `internal/config`

Clash / mihomo 配置的数据模型。使用 `yaml:",inline"` 保留未知字段，保证导出时不丢协议参数。

### `internal/subscription`

订阅读取与校验：

- Clash / mihomo YAML；
- 节点去重、信息伪节点识别；
- 后续扩展：base64 v2ray 订阅、sing-box JSON。

### `internal/clash`

Clash REST API 客户端：

- `GET /version`
- `GET /proxies`
- `GET /proxies/{name}/delay?url=...&timeout=...`
- `PUT /proxies/{group}`（Phase 2 自动切换）

兼容 mihomo、Clash Verge、Mihomo Party、sing-box 的 `clash_api`。

### `internal/sidecar`

用订阅配置启动一个临时 mihomo：

- 随机本地端口，`external-controller` 只绑 `127.0.0.1`；
- 关闭 DNS/TUN，避免和用户客户端冲突；
- 只保留 `proxies` 和 `MATCH,DIRECT` 规则，不需要 Geo 数据；
- 测速结束后杀掉进程、清理临时目录。

这样即使用户没有开任何客户端的控制口，也能测速。

### `internal/tester`

测试引擎：

- 节点级并发，节点内多轮串行；
- 每轮独立超时；
- 失败不重试（失败本身是稳定性信号）；
- 输出 `score.Result`。

### `internal/score`

指标计算与评分：

- 成功率、P50、P90、Max、Jitter；
- 综合评分；
- 排序与阈值过滤。

### `internal/export`

生成可日用配置：

- 默认保留原 rules / proxy-groups / DNS / TUN / rule-providers；
- 过滤组内未通过阈值的节点，追加 `NodePulse` url-test 组；
- `--minimal` 才输出只含节点和 `MATCH,NodePulse` 的最小配置。

### `internal/cli`

命令行入口，负责 `nodes / test / rank / export / daemon / status / history`。

### `internal/daemon`

自动选择守护进程（核心）：

- `internal/selector` 统一解析 Selector 组、候选节点与切换决策；
- 永不自动管理 `GLOBAL`；只测试组内直接节点，不回退到全量代理；
- 最低成功率 + 滞后 margin + 连续 confirm；
- 切换后校验当前节点并做真实延迟验证，失败回滚并记录回滚结果；
- 全部不达标时用 `DIRECT` 基线区分本地网络/测试 URL 与节点问题；
- 结构化事件写入历史（原因码、置信度、候选数、前后评分、耗时、回滚结果）。

### `internal/store`

历史记录（JSONL）：

- 每次 cycle / switch / rollback / error；
- `nodepulse history` 读取最近 N 条；
- 接口独立，后续可替换为 SQLite。

### `internal/export`

配置导出：

- `Stable` 生成最终配置；
- `Preview` 在写入前返回影响面：保留/排除节点、受影响分组、`DIRECT` 回退、规则/DNS/TUN/rule-providers 是否保留；
- GUI / Web 先展示预览，再确认写入。

### `internal/results`

GUI 与 Web 共用的结果文件：

- schema 与 `results.json` 一致；
- 原子写入；
- 结果绑定配置路径，切换环境时不展示旧结果。

### `internal/backup`

覆盖前快照：

- 设置文件和导出配置写入前自动备份；
- 保留最近 N 份；
- 目录固定在 `~/.local/share/nodepulse/backups/`，权限 0600。

### `internal/autostart`

桌面版开机自启：

- Linux 写 XDG autostart；
- macOS 写 LaunchAgent；
- Windows 写 HKCU Run；
- 默认关闭，开启后带 `--background` 参数启动。

### `gui/tray.go` 与 `gui/notify_*.go`

桌面驻留：

- 托盘提供打开面板、开机启动、退出；
- 关闭窗口隐藏到托盘，daemon 继续运行；
- 切换/回滚/失败时发送系统通知，窗口在前台时不重复打扰。

### `internal/health`

首启/环境检查：

- 只读检查控制口、订阅、mihomo 和 Selector 组；
- 返回结构化状态与建议；
- GUI / Web 共用。

### `internal/lock`

daemon 单写者：

- 基于锁文件和 PID 存活检查；
- GUI / Web / CLI 共用同一锁路径；
- 崩溃遗留锁可清理。

### `internal/netfetch`

订阅下载边界：

- 仅 http/https，拒绝内嵌凭据；
- 限制重定向、超时和大小；
- 拒绝链路本地/元数据地址，保留局域网/自建服务。

## 数据流

1. 读取订阅 YAML；
2. 选择测速后端：
   - `--api`：连接已有客户端；
   - 否则：启动 mihomo sidecar；
3. 对每个节点执行 N 轮 `delay` 请求；
4. 计算指标和评分；
5. 输出 `results.json`；
6. `export` 读取结果，过滤并生成 `stable.yaml`。

## 适配器模型

```go
type Delayer interface {
    Delay(ctx context.Context, name, url string, timeout time.Duration) (int, error)
}

type Switcher interface {
    Switch(ctx context.Context, group, node string) error
}
```

- 测速只需要 `Delayer`；
- 自动切换需要 `Switcher`；
- 不支持实时切换的客户端走 `export`。

## 安全

- 控制口只绑 `127.0.0.1`；
- sidecar 使用随机 secret；
- 日志默认不打印订阅 URL、UUID、密码；
- 导出的配置权限建议 `0600`；
- 不上报任何数据。

## 分发

- 单二进制：`cmd/nodepulse`；
- `.deb`：nfpm；
- AppImage：appimagetool；
- systemd user unit：`nodepulse daemon`；
- GitHub Actions + GoReleaser。

## 目录

```
cmd/nodepulse/          # 入口
internal/config/        # 配置模型
internal/subscription/  # 订阅解析
internal/clash/         # Clash REST 客户端
internal/tester/        # 测试引擎
internal/score/         # 指标与评分
internal/export/        # 配置导出
internal/share/         # base64 / 分享链接解析与编码
internal/sidecar/       # mihomo 生命周期
internal/daemon/        # 自动切换守护
internal/store/         # 历史记录（JSONL）
internal/mcp/           # MCP stdio server
internal/webui/         # 本地 Web UI
internal/adapters/      # v2rayA 等客户端适配
internal/cli/           # 命令行
gui/                    # Wails 桌面应用（独立 Go module）
plugins/dsh/            # DeepSeek Harness 插件
packaging/              # 打包（nfpm / AppImage / systemd）
ui/                     # Wails 前端（Phase 3）
mcp/                    # MCP server（Phase 4）
packaging/              # 打包（Phase 3）
```
