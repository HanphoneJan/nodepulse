# 客户端适配

NodePulse 通过 Clash REST API、v2rayA REST API 或本地 sidecar 工作，不接管客户端流量。

## 能力

| 客户端 | 测速 | 实时切换 | 接入方式 | 说明 |
|---|---|---|---|---|
| FlClash | 支持 | 需开启控制口 | Clash REST | 默认可能未开启 `external-controller` |
| Clash Verge Rev | 支持 | 支持 | Clash REST | 控制口通常默认开启 |
| Mihomo Party | 支持 | 支持 | Clash REST | 使用其 external-controller 与 secret |
| sing-box GUI | 支持 | 支持 | `experimental.clash_api` | 需要在配置中启用 clash_api |
| v2rayA | 支持 | CLI 支持 | v2rayA REST | `nodepulse v2raya`；GUI/Web 未接入 |
| v2rayN | 支持（导出） | 不支持 | 导出 Clash / base64 | 无稳定本地控制 API |
| 任意订阅 | 支持 | — | mihomo sidecar | 不需要客户端 |

NodePulse 自动探测常见控制口：`9090`、`9097`、`9091`、`9099`、`9092`、`9093`。如果自动探测不对，在设置或 CLI 中显式填写端口和 secret。

## Clash REST

适用于 FlClash、Clash Verge Rev、Mihomo Party、sing-box `clash_api`。

```bash
nodepulse test -f sub.yaml --api http://127.0.0.1:9090 --secret <secret>
nodepulse daemon --api http://127.0.0.1:9090 --group "节点选择"
```

GUI/Web 中选择控制口即可；自动选择只管理 Selector 组，不管理 `GLOBAL`。

### FlClash

FlClash 底层是 mihomo，但默认可能没有开启外部控制口。

1. 打开 FlClash 的 Clash/网络设置；
2. 启用 External Controller，例如 `127.0.0.1:9090`；
3. 如设置了 secret，在 NodePulse 中填写；
4. 不想改客户端时，直接用本地订阅 + sidecar 测速（不能实时切换）。

### Clash Verge Rev

在设置中启用外部控制，常见端口 `127.0.0.1:9097`。填入地址和 secret 后即可一键优化和自动选择。

### Mihomo Party

确认 external-controller 地址和 secret，使用 Clash REST 接入。

### sing-box GUI

在配置中启用：

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

## sidecar

任何订阅都可以不依赖客户端测速：

```bash
nodepulse test -f subscription.yaml --mihomo /path/to/mihomo --rounds 5
```

NodePulse 会：

1. 用订阅里的 `proxies` 生成临时配置；
2. 分配随机 mixed-port 和 external-controller；
3. 关闭 DNS/TUN，避免和现有客户端抢端口；
4. 启动 mihomo，逐节点测速；
5. 结束后停止进程并清理临时目录。

sidecar 只能测速/推荐，不能切换客户端。

## v2rayA

CLI 适配，支持 login/status/switch：

```bash
nodepulse v2raya login  --url http://127.0.0.1:2017 --username <u> --password <p>
nodepulse v2raya status --url http://127.0.0.1:2017 --token <jwt>
nodepulse v2raya switch --url http://127.0.0.1:2017 --token <jwt> --node "香港"
```

GUI/Web 目前没有 v2rayA 环境类型；需要时使用 CLI 或 sidecar。

## v2rayN

v2rayN 没有稳定的本地控制 API。当前只支持导出：

```bash
nodepulse export -f sub.yaml -i results.json -o stable.yaml
nodepulse export -f sub.yaml -i results.json -o stable.txt --format v2rayn
```

在 v2rayN 中导入导出的 Clash 配置或 base64 订阅；NodePulse 不能实时切换 v2rayN 节点。
