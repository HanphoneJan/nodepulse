# 设计：测试协议、指标与评分

## 1. 测试协议

每个节点执行 `R` 轮（默认 5，推荐 10）独立的代理请求：

```
GET http://<controller>/proxies/{name}/delay?url=<test-url>&timeout=<ms>
```

- 请求经过目标节点，返回该次请求的毫秒延迟；
- 失败（超时、DNS、TLS、节点拒绝）记为失败；
- 不做失败重试：失败率本身就是稳定性信号；
- 节点之间并发，节点内部串行，避免同一节点自竞争。

默认测试 URL：`https://www.gstatic.com/generate_204`。可换成 `https://www.cloudflare.com/cdn-cgi/trace` 或自定义。

## 2. 指标

| 指标 | 定义 |
|---|---|
| `success` | 成功轮数 |
| `total` | 总轮数 |
| `min_ms` | 成功轮的最小值 |
| `median_ms` | P50 |
| `p90_ms` | P90，反映尾部延迟 |
| `max_ms` | 最大值（仅供参考，容易被单次尖刺影响） |
| `jitter_ms` | `P90 - P50`，比 `max-min` 更抗单次尖刺 |
| `score` | 综合评分，见下 |

## 3. 评分

```
successRate   = success / total
latencyScore  = clamp(1 - (p90 - 100) / 900, 0, 1)
stabilityScore= clamp(1 - jitter / 500, 0, 1)
score         = 0.5 * successRate + 0.3 * latencyScore + 0.2 * stabilityScore
```

解释：

- 100ms 的 P90 得满分，1000ms 得 0；
- 0ms 抖动得满分，500ms 抖动得 0；
- 成功率权重最高：能用比快更重要；
- 成功率 `< 0.8` 的节点额外乘 `successRate`，避免“偶尔能通”的节点排前面。

示例：

| 节点 | 成功率 | P90 | 抖动 | 分数 |
|---|---|---|---|---|
| A | 10/10 | 120ms | 20ms | 0.5 + 0.29 + 0.19 = 0.98 |
| B | 10/10 | 300ms | 120ms | 0.5 + 0.23 + 0.15 = 0.88 |
| C | 6/10 | 180ms | 60ms | (0.5*0.6 + 0.27 + 0.18) * 0.6 = 0.45 |

## 4. 自动切换（Phase 2）

目标：不是“每次选最低延迟”，而是“当前节点明显变差时才切”。

```
if candidate.score > current.score * 1.20
   and candidate.successRate >= 0.8
   and 连续 2 轮成立
then switch
```

- 滞后阈值 20%，避免抖动导致来回切；
- 连续 2 轮确认，避免单次测量噪声；
- 切换后验证：连续 2 轮失败则回滚到上一个稳定节点；
- 每次切换记录原因、前后指标、时间。

## 5. 导出格式

`export` 生成最小可用 Clash 配置：

```yaml
mixed-port: 7890
allow-lan: false
mode: rule
log-level: info
proxies:
  - { ...原始节点参数... }
proxy-groups:
  - name: Auto
    type: url-test
    url: https://www.gstatic.com/generate_204
    interval: 300
    tolerance: 50
    proxies: [节点A, 节点B, ...]
rules:
  - MATCH,Auto
```

节点参数原样保留（包括 Reality、WS、Hysteria2 等字段），避免导出后不可用。

## 6. `results.json` schema

```json
{
  "generated_at": "2026-11-03T00:00:00+08:00",
  "url": "https://www.gstatic.com/generate_204",
  "rounds": 10,
  "results": [
    {
      "name": "🇭🇰香港家宽专线",
      "type": "hysteria2",
      "success": 10,
      "total": 10,
      "min_ms": 52,
      "median_ms": 57,
      "p90_ms": 145,
      "max_ms": 149,
      "jitter_ms": 88,
      "score": 0.91
    }
  ]
}
```

## 7. 边界情况

- **信息伪节点**：名字包含“剩余流量 / 套餐到期 / 流量重置 / 官网”等，默认跳过；
- **重复节点**：同名或同 `server:port:type` 的节点在排名中标注，导出时去重；
- **DNS 失败**：直接计为失败，不尝试修复（客户端侧 DNS 问题另算）；
- **节点改名**：结果按 name 关联，导出前校验节点仍存在；
- **订阅更新**：结果文件带时间戳，导出时提示是否过期（Phase 2）。
