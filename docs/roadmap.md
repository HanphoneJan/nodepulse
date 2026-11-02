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

## Phase 3 · GUI + 托盘 + 打包

- [ ] Wails 应用：仪表盘、节点表、订阅、设置、历史
- [ ] 托盘菜单：立即测试 / 自动模式开关 / 打开面板 / 退出
- [ ] 本地 Web UI（127.0.0.1 随机端口）
- [ ] `.deb`（nfpm）
- [ ] AppImage
- [ ] GitHub Actions 发布 linux/amd64、arm64
- [ ] 图标、桌面入口、systemd user unit

验收：下载 `.deb` 安装后，托盘可用，能导入订阅并看到稳定性排名。

## Phase 4 · MCP + dsh 插件

- [ ] MCP stdio server
- [ ] 工具：`list_profiles`、`list_nodes`、`test_nodes`、`rank_nodes`、`get_current`、`switch_node`、`export_config`、`daemon_status`
- [ ] MCP 资源：最近一次结果、节点历史
- [ ] dsh 插件薄封装（复用 CLI/MCP）
- [ ] 文档与示例

验收：在支持 MCP 的 agent 客户端里可以“测一下所有节点，切到最稳的”。

## Phase 5 · v2rayA / v2rayN 适配

- [ ] v2rayA REST：读取节点、连接/切换
- [ ] v2rayN：读取订阅/配置、导出过滤配置
- [ ] v2rayN 可选“应用并重启”模式
- [ ] base64 v2ray 订阅解析
- [ ] sing-box JSON 解析（可选）

验收：v2rayA 可实时切换；v2rayN 能导入过滤后的配置并保留节点参数。

## 后续候选

- Windows / macOS 构建
- 定时任务导出到 GitHub Gist / WebDAV
- 节点速度测试（下载吞吐）
- 多订阅合并与去重
- 延迟/成功率历史图表
- 与 OpenChamber / 其他 agent 的深度集成
