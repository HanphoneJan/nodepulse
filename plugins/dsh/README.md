# @hanphone/dsh-nodepulse

NodePulse 的 DeepSeek Harness（dsh）插件：把节点稳定性测试、排名、切换和导出暴露成一个模型工具。

## 前置条件

- 已安装 `nodepulse`（`make build` 后把 `bin/nodepulse` 放到 `PATH`，或设置插件配置 `binary`）；
- FlClash / Clash Verge 等开启了 Clash 控制口，或提供 `mihomo_path` 走 sidecar。

## 安装

```bash
# 发布后
dsh plugin --profile web add @hanphone/dsh-nodepulse

# 或本地打包
npm pack
dsh plugin --profile web add ./hanphone-dsh-nodepulse-0.1.0.tgz
```

## 工具调用示例

- `action: "list", config_path: "~/sub.yaml"`
- `action: "test", config_path: "~/sub.yaml", api: "http://127.0.0.1:9090", rounds: 5`
- `action: "rank", results_path: "results.json"`
- `action: "switch", api: "http://127.0.0.1:9090", group: "🚀节点选择", node: "🇭🇰香港家宽专线"`
- `action: "export", config_path: "~/sub.yaml", results_path: "results.json", out: "stable.yaml", min_success: 0.8`
- `action: "status", api: "http://127.0.0.1:9090"`
- `action: "history", n: 20`

## 配置

```yaml
- id: nodepulse
  name: '@hanphone/dsh-nodepulse'
  config:
    binary: /usr/bin/nodepulse   # 可选
    api: http://127.0.0.1:9090   # 可选默认值
    config_path: ~/sub.yaml      # 可选默认值
```

插件只负责调用 CLI；测速、评分、切换都在 NodePulse 核心里，和 GUI/MCP 共用同一套逻辑。
