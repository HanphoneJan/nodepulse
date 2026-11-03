# 设计

## 测试协议

每个节点执行 `R` 轮独立请求（默认 5，推荐 10）：

```text
GET /proxies/{name}/delay?url=<test-url>&timeout=<ms>
```

- 请求经过目标节点，返回该次请求的毫秒延迟；
- 超时、DNS、TLS、节点拒绝都记为失败，不重试；
- 节点之间并发，节点内部串行，避免同一节点自竞争；
- 默认测试 URL：`https://www.gstatic.com/generate_204`。

## 指标与评分

| 指标 | 定义 |
|---|---|
| `success` / `total` | 成功轮数 / 总轮数 |
| `median_ms` | P50 |
| `p90_ms` | P90，反映尾部延迟 |
| `jitter_ms` | `P90 - P50` |
| `score` | 综合评分 |

```text
successRate    = success / total
latencyScore   = clamp(1 - (p90Ms - 100) / 900, 0, 1)
stabilityScore = clamp(1 - jitterMs / 500, 0, 1)
score          = 0.5*successRate + 0.3*latencyScore + 0.2*stabilityScore
```

成功率低于 80% 的节点再乘以成功率，避免“偶尔能通”的节点排前面。

## 自动选择

目标不是“每次选最低延迟”，而是当前节点明显变差时才切换，并且切得可解释、可回滚。

### 组与候选

- 只管理 `Selector` 组，永不自动管理 `GLOBAL`；
- 优先使用设置里显式配置的组；未配置时按常见名称排序；
- 只取组内直接、可测节点；嵌套组、DIRECT、REJECT 跳过；
- 组内没有直接节点时直接报错，不回退到全量代理。

### 决策

```text
candidate.successRate >= minSuccess (默认 0.8)
candidate.score > current.score * (1 + margin)  (默认 0.2)
连续 confirm 轮成立 (默认 2)
=> switch
```

- 当前节点不达标：立即切到最佳达标节点；
- 当前已是最佳：保持；
- 所有节点不达标：用 `DIRECT` 做本地基线；DIRECT 也失败则提示网络/测试 URL 问题；
- 结果带置信度：轮数 <3 低、3–4 中、≥5 高。

### 切换事务

1. 记录切换前指标、候选、原因码、置信度；
2. 调用切换；
3. 等待控制器生效，校验当前节点；
4. 对新节点做一次真实延迟复测；
5. 任一步失败都尝试回滚，记录 `rollback_attempted` / `rollback_ok`；
6. 事件写入历史。

## 多订阅联合测速

客户端控制口只暴露当前激活配置；未激活订阅需要读取本地 profile：

- FlClash：`profiles/*.yaml` + 当前 `config.yaml`；
- Clash Verge：profiles 目录；
- Mihomo/Clash：当前配置。

流程：

1. 读取各 profile 的 `proxies`；
2. 同名节点加来源后缀，避免冲突；
3. 用同一个临时 mihomo sidecar 测速；
4. 结果带来源标签，给跨订阅总排名；
5. 只推荐，不自动切换未激活 profile。

如果控制口已包含全部节点，优先用控制口；本地 profile 作为补充。

## 下载模式

默认自动选择服务于浏览：低延迟、成功率、抖动。大文件下载是另一个目标。

1. 先按常规评分得到候选池；
2. 取前 5 个达标候选；全部不可达时最多补测下一批 5 个；
3. 通过临时 sidecar 的 selector + mixed-port，对目标文件做 Range 下载（默认 3 MiB / 5 秒）；
4. 记录 TTFB、p25/中位吞吐、卡顿；
5. 目标 TLS/EOF/超时/HTTP 4xx/5xx 的节点淘汰；
6. 同一节点/直连 + 目标 host/path：成功缓存 20 分钟，失败缓存 5 分钟；
7. 同时探测直连；直连 p25 比最佳节点高 20% 以上时推荐直连，不切节点；
8. 否则只在下载开始前切换一次，切换沿用验证和回滚；下载过程中不切换。

下载评分：

```text
throughput = p25 KiB/s（下界）
downloadScore = 0.60*log(1+throughput) 归一化
              + 0.15*(p25/median)
              + 0.15*常规节点评分
              + 0.10*首字节评分
```

边界：目标必须是直接文件 URL；需要本地订阅配置和 mihomo 才能构建 sidecar；签名链接会过期。

## 导出

默认导出保留原配置：

- 保留 rules、proxy-groups、DNS、TUN、rule-providers 等字段；
- `proxies` 只保留通过阈值的节点；
- 组成员过滤为稳定节点或仍存在的子组；空组回退 `DIRECT`；
- 追加 `NodePulse`（url-test）组；
- 清理 `external-controller` 和 `secret`；
- `--minimal` 才生成只含节点和 `MATCH,NodePulse` 的最小配置。

导出前通过 `export.Preview` 展示影响：保留/排除数、失去成员的分组、`DIRECT` 回退、规则/DNS/TUN/rule-providers 是否保留。

## 持久化与安全

- `results.json`：结果跨重启恢复；配置路径不一致时不展示旧结果。
- `history.jsonl`：cycle/switch/rollback/error 结构化事件。
- `probe-cache.json`：下载探测缓存，不含签名 URL。
- 设置、结果、备份、锁文件权限 0600；备份目录 0700。
- 设置和导出覆盖前自动备份，保留最近 20 份。
- 订阅下载只允许 http/https，限制重定向/超时/10 MiB，拒绝链路本地/云元数据地址。
- daemon 使用跨进程锁；第二个实例明确报错，不静默重复运行。

## 边界情况

- 信息伪节点：名称包含“剩余流量/套餐到期/流量重置/官网”等，默认跳过；
- 同名节点：排名保留，导出按名称校验；
- DNS 失败：计为失败，不尝试修复；
- 节点改名：结果按 name 关联，导出前校验存在；
- 订阅更新/到期提醒：当前不做。
