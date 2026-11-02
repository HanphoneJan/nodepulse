# 客户端适配矩阵

## 能力总览

| 客户端 | 测速 | 实时切换 | 接入方式 | 备注 |
|---|---|---|---|---|
| FlClash | ✅ | ⚠️ 需开启控制口 | Clash REST API | 默认 `external-controller` 为空 |
| Clash Verge Rev | ✅ | ✅ | Clash REST API | 默认控制口通常已开 |
| Mihomo Party | ✅ | ✅ | Clash REST API | 在设置里确认端口和 secret |
| sing-box GUI | ✅ | ✅ | `experimental.clash_api` | 需要启用 clash_api |
| v2rayA | ✅（规划） | ✅（规划） | v2rayA REST API | 需要登录 token |
| v2rayN | ✅（导出） | ⚠️ 受限 | 读取配置 + 导出 | 无稳定本地控制 API |
| 任意订阅 URL | ✅ | — | 内置 mihomo sidecar | 不需要客户端 |

图例：✅ 支持；⚠️ 需要额外配置或有条件；— 不适用。

## 通用兜底：sidecar 模式

任何客户端、甚至没有客户端，都可以用：

```bash
nodepulse test -f subscription.yaml --mihomo /path/to/mihomo
```

NodePulse 会：

1. 用订阅里的 `proxies` 生成临时配置；
2. 在 `127.0.0.1` 上随机分配 mixed-port 和 external-controller；
3. 关闭 DNS/TUN，避免和现有客户端抢端口；
4. 启动 mihomo，逐节点多轮测试；
5. 结束后杀掉进程并清理临时目录。

## FlClash

FlClash 底层是 mihomo，但默认没有开 `external-controller`。

- 打开 FlClash 的 Clash / 网络设置；
- 找到“外部控制 / External Controller”，填写 `127.0.0.1:9090`；
- 如果设置了 secret，运行时用 `--secret` 传入；
- 不确定入口时，直接用 sidecar 模式，不需要改 FlClash。

## Clash Verge Rev

- 设置 → Clash → 外部控制；
- 记下地址（常见 `127.0.0.1:9097`）和 secret；
- 用 `--api http://127.0.0.1:9097 --secret ...`。

## Mihomo Party

- 设置里确认 external-controller 地址和 secret；
- 用 `--api` / `--secret` 接入。

## sing-box GUI

sing-box 本身没有 Clash 控制口，需要在配置里启用：

```json
{
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:9090",
      "secret": ""
    }
  }
}
```

启用后与 Clash 系用法一致。

## v2rayA

- v2rayA 自带 REST API（默认 `127.0.0.1:2017`）；
- 需要登录拿 token；
- Phase 5 实现适配：读取节点、连接/切换；
- 目前可先用 sidecar 模式测速，再手动在 v2rayA 里选。

## v2rayN

v2rayN 没有稳定的本地控制 API，强实时切换只能改数据库或重启核心，风险高。v1 策略：

1. 读取 v2rayN 的订阅 URL 或导出的 Clash 配置；
2. 用 sidecar 测速；
3. 导出只包含稳定节点的配置；
4. 在 v2rayN 里导入该配置。

后续可选“应用并重启”模式：写入配置后调用 v2rayN 的重启，不修改其内部数据库。

## 安全提示

- 控制口只监听 `127.0.0.1`；
- 不要在没有防火墙的局域网上把控制口开到 `0.0.0.0`；
- secret 不要写进仓库或日志。
