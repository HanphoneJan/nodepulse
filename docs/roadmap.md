# 路线图

## Phase 1 · 核心引擎 + CLI（已实现）

目标：不依赖 GUI，跑通“订阅 → 测速 → 排名 → 导出”。

- [x] Clash / mihomo YAML 解析，保留未知节点字段
- [x] 信息伪节点识别（剩余流量 / 套餐到期等）
- [x] Clash REST 客户端（version / proxies / delay）
- [x] mihomo sidecar：随机端口、临时配置、自动清理
- [x] 多轮测试引擎（并发 + 成功率 + P50/P90/抖动）
- [x] 稳定性评分与排序
- [x] JSON 结果输出
- [x] 过滤导出 Clash 配置
- [x] 单元测试
- [x] README / 架构 / 设计文档

验收：

```bash
nodepulse nodes -f testdata/subscription.yaml
nodepulse test  -f testdata/subscription.yaml --mihomo ./mihomo --rounds 3
nodepulse rank  -i results.json
nodepulse export -f testdata/subscription.yaml -i results.json -o stable.yaml
```

## Phase 2 · Clash 适配 + 守护/自动切换（已实现）

- [x] 自动发现 Clash 控制口里的 Selector 组
- [x] `Switcher` 接口与 `PUT /proxies/{group}` 实现
- [x] `nodepulse daemon`：定时测试 + 滞后阈值 + 连续确认
- [x] 切换验证与失败回滚
- [x] 历史记录（JSONL；SQLite 作为后续可替换后端）
- [x] `status` / `history` 命令
- [ ] systemd user unit（随 Phase 3 打包一起做）
- [ ] 与 FlClash / Clash Verge 的实机集成测试（需要用户环境）

验收：当当前节点连续失败或明显变差时，守护进程在 2 个周期内切换到更稳节点，并在失败时回滚。

## Phase 3 · GUI + 打包（已实现，托盘/本地 Web UI 后续）

- [x] Wails 应用：连接/状态、订阅测速、结果表、切换、导出
- [ ] 托盘菜单（后续）
- [x] 本地 Web UI（`nodepulse serve`，与 GUI 共用同一套核心）
- [x] `.deb`（nfpm 配置）
- [x] AppImage 构建脚本
- [x] GitHub Actions 发布工作流
- [x] 图标、桌面入口、systemd user unit

验收：`make gui` 构建桌面应用；`make deb` / `make appimage` 产出安装包。

## Phase 4 · MCP + dsh 插件（已实现）

- [x] MCP stdio server
- [x] 工具：`nodes.list`、`nodes.test`、`nodes.rank`、`nodes.switch`、`export.stable`、`status.groups`、`history.recent`
- [ ] MCP 资源（最近结果、节点历史）
- [x] dsh 插件薄封装（复用 CLI）
- [x] 文档与示例

验收：支持 MCP 的 agent 客户端可以“测一下所有节点，切到最稳的”。

## Phase 5 · base64 / v2rayA / v2rayN（已实现）

- [x] base64 分享链接订阅解析（vmess / vless / trojan / ss / hysteria2）
- [x] v2rayA REST：login / touch / connect（状态与切换）
- [x] v2rayN：导出 Clash 配置或 base64 订阅
- [ ] v2rayN “应用并重启”模式
- [ ] sing-box JSON 解析（可选）

验收：v2rayA 可切换；导出的 base64 订阅可被 v2rayN 导入。

## 后续候选

- Windows / macOS 构建
- 定时任务导出到 GitHub Gist / WebDAV
- 节点速度测试（下载吞吐）
- 多订阅合并与去重
- 延迟/成功率历史图表
- 与 OpenChamber / 其他 agent 的深度集成
