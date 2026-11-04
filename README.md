<div align="center">
  <img src="packaging/nodepulse.svg" width="96" alt="NodePulse">
  <h1>NodePulse</h1>
  <p><strong>Beyond latency: measure node stability, rank nodes, and keep the good ones.</strong></p>
  <p>
    <a href="https://github.com/HanphoneJan/nodepulse/releases"><img src="https://img.shields.io/github/v/release/HanphoneJan/nodepulse" alt="Release"></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/HanphoneJan/nodepulse" alt="License"></a>
    <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/HanphoneJan/nodepulse" alt="Go version"></a>
  </p>
</div>

NodePulse 是一个与客户端解耦的代理节点稳定性工具。它不只看一次延迟，而是对每个节点做多轮真实请求，统计成功率、P50/P90、抖动，并给出稳定性评分、自动选择和过滤后的配置。

NodePulse **不代理流量**：它测试和管理已有的 Clash/mihomo 控制口，或用临时 mihomo sidecar 做本地测速。

## 功能

**节点质量**

- Clash/mihomo YAML、base64 分享链接订阅解析
- 多轮测速：成功率、P50/P90、最大值、抖动、稳定性评分
- 信息伪节点过滤、同名节点去重、原配置字段保留
- 内置 mihomo sidecar，随机端口，不依赖具体客户端
- 导出前规则影响预览；默认保留 rules、分组、DNS/TUN、rule-providers

**自动选择**

- 只管理 Selector 组，永不自动管理 `GLOBAL`
- 只测组内直接节点，不回退到全量代理
- 成功率阈值 + 滞后余量 + 连续确认，避免抖动切换
- 切换后校验当前节点、真实延迟复测，失败回滚
- `DIRECT` 基线区分本地网络/测试 URL 问题
- 可解释原因码、置信度、候选数、前后评分、耗时、回滚结果

**桌面端**

- Flutter 桌面界面，配色与控件直接复用 FlClash 的设计系统（GPL-3.0）
- 环境 profiles：客户端 + 订阅 + Selector 组 + 测速参数
- 托盘驻留、可选开机自启
- 启动后自动检查 GitHub Release，也可在设置里手动检查并打开下载页
- 测速结果跨重启保留；设置自动备份，可导出/恢复
- 多订阅联合测速：读取本地 profile，跨机场排名并标注来源
- 下载模式：目标文件小范围吞吐探测、不可达淘汰、直连对比、只在下载前切换一次
- CLI、MCP server 与桌面端共用核心

## 平台

| 平台 | 产物 | 状态 |
|---|---|---|
| Linux | `.deb`、AppImage | 本地构建，推荐桌面端 |
| Windows | `.exe`（NSIS 安装包）、zip（便携版） | CI 构建；代码签名未配置 |
| macOS | `.dmg`、zip（universal `.app`） | CI 构建；签名/公证未配置 |
| Android / iOS | — | 不是桌面打包目标；需要独立移动客户端 |

桌面端通过本机 `127.0.0.1` 上的临时 token API 与 Go 后端通信；API 只监听本机，不要暴露到局域网或公网。

## 安装

从 [Releases](https://github.com/HanphoneJan/nodepulse/releases) 下载对应平台的包。Linux 也可以本地构建：

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
VERSION=0.3.1 make build desktop deb
```

产物在 `dist/`。Linux 桌面端需要 Flutter 和 GTK3；Windows/macOS 安装包由对应平台的 CI 构建。

## 快速开始

### 桌面端

```bash
make run
```

1. 连接 Clash/mihomo 控制口，或选择本地订阅；
2. 选择要管理的 Selector 组；
3. 一键优化，或开启后台自动选择；
4. 在设置里配置环境、导出、备份和托盘行为。

### CLI

```bash
# 测速
nodepulse test -f sub.yaml --mihomo /usr/bin/mihomo --rounds 5 --out results.json

# 或者复用客户端控制口
nodepulse test -f sub.yaml --api http://127.0.0.1:9090 --rounds 5

# 排名 / 导出
nodepulse rank -i results.json
nodepulse export -f sub.yaml -i results.json -o stable.yaml

# 后台自动选择
nodepulse daemon --api http://127.0.0.1:9090 --group "节点选择" --interval 5m
```

完整命令见 `nodepulse -h`。

## 评分与自动选择

```text
successRate    = success / total
latencyScore   = clamp(1 - (p90Ms - 100) / 900, 0, 1)
stabilityScore = clamp(1 - jitterMs / 500, 0, 1)
score          = 0.5*successRate + 0.3*latencyScore + 0.2*stabilityScore
```

自动选择不是“每次选最低延迟”，而是当前节点明显变差时才切换：

- 最低成功率默认 80%；
- 候选评分需高于当前 20% 才进入确认；
- 连续 2 轮确认后才切换；
- 切换后验证，失败回滚到原节点。

下载模式是独立场景：先按常规评分取前 5 个候选，再对目标文件做小范围 Range 下载探测；目标站点不可达的节点淘汰，按 p25 吞吐、稳定性和首字节排序。直连也会参与探测，直连明显更快时直接推荐直连。

## 支持的客户端

| 客户端 | 测速 | 实时切换 | 接入方式 |
|---|---|---|---|
| FlClash | 支持 | 需开启 external-controller | Clash REST |
| Clash Verge Rev | 支持 | 支持（需开启外部控制） | Clash REST |
| Mihomo Party | 支持 | 支持 | Clash REST |
| sing-box GUI | 支持 | 支持 | `experimental.clash_api` |
| v2rayA | 支持 | CLI 支持 | v2rayA REST（GUI/Web 未接入） |
| v2rayN | 支持（导出） | 不支持实时切换 | 导出 Clash / base64 |
| 任意订阅 | 支持 | — | 内置 mihomo sidecar |

NodePulse 会主动检查本机运行中的客户端进程和它的配置目录，而不只是扫描常见端口。Windows 上会读取 `%APPDATA%` 下 Clash Verge Rev / FlClash 等客户端的 `config.yaml`，从中取得控制口和 secret 并自动填入；如果客户端已运行但关闭了外部控制，界面会显示“已检测到客户端”并提示去客户端开启，不再显示成“未检测到”。

详见 [客户端适配矩阵](docs/adapter-matrix.md)。

## 文档

- [架构与包结构](docs/architecture.md)
- [测试协议、评分与自动选择](docs/design.md)
- [UI 设计与前端结构](docs/ui.md)
- [客户端适配](docs/adapter-matrix.md)
- [开发、测试与调试](docs/development.md)
- [跨平台打包](docs/packaging.md)
- [贡献指南](CONTRIBUTING.md)
- [安全策略](SECURITY.md)

## License

GPL-3.0。界面部分复用自 FlClash（GPL-3.0）；第三方组件许可见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
