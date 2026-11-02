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

生成只包含稳定节点的配置：

- 保留原始节点参数；
- 生成 `url-test` 自动组；
- 输出 `rules: [MATCH,Auto]`。

### `internal/cli`

命令行入口，负责 `nodes / test / rank / export / daemon / status / history`。

### `internal/daemon`

自动切换守护进程：

- 解析 Selector 组与候选节点；
- 定时调用测试引擎；
- 滞后阈值 + 连续确认；
- 切换后验证，失败回滚；
- 事件写入历史。

### `internal/store`

历史记录（JSONL）：

- 每次 cycle / switch / rollback / error；
- `nodepulse history` 读取最近 N 条；
- 接口独立，后续可替换为 SQLite。

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
internal/sidecar/       # mihomo 生命周期
internal/daemon/        # 自动切换守护
internal/store/         # 历史记录（JSONL）
internal/cli/           # 命令行
ui/                     # Wails 前端（Phase 3）
mcp/                    # MCP server（Phase 4）
packaging/              # 打包（Phase 3）
```
