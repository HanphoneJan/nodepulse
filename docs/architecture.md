# 架构

## 目标

1. **客户端无关**：测速结果不依赖某个客户端，可给 FlClash、Clash Verge、Mihomo Party、sing-box 或 v2rayA 使用。
2. **可复用**：CLI、Flutter 桌面端、MCP 共用同一套核心。
3. **可解释**：自动选择记录原因、候选、评分和回滚结果。
4. **安全边界清晰**：本地服务不暴露公网；订阅下载限制目标；daemon 单写者。

## 非目标

- 不实现代理协议、TUN、DNS 转发或流量接管。
- 不替代 Clash/mihomo 客户端。
- 不分发节点或订阅。
- 不做站点专用下载链接抓取。
- 不提供 Android/iOS 客户端；移动端需要独立项目。

## 运行方式

```text
CLI ───────────────┐
Flutter 桌面端 ────┤  本机 token API
MCP server ────────┼──> core: test / score / selector / daemon / export
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
   Clash REST API   mihomo sidecar   subscription YAML
   (现有客户端)      (本地临时测速)    (文件/base64)
```

## 数据流

1. 读取订阅 YAML、base64 分享链接，或连接已有控制口。
2. 解析节点和 Selector 组；过滤信息伪节点。
3. 多轮 `delay` 请求，计算成功率、P50/P90、抖动和评分。
4. 自动选择按阈值、滞后和确认决定是否切换。
5. 切换后校验当前节点并做真实延迟复测，失败回滚。
6. 结果写入 `results.json`；历史写入 JSONL。
7. 导出时过滤节点、保留原配置结构，并展示影响预览。

## 包结构

```text
cmd/nodepulse/        CLI 入口
desktop/              Flutter 桌面端（Dart）
  lib/ui/flclash/     从 FlClash 移植的设计系统（GPL-3.0）

internal/
  clash/              Clash REST 客户端
  config/             Clash/mihomo YAML 模型，保留未知字段
  subscription/       订阅读取、base64 解析
  share/              v2ray 分享链接解析
  discover/           控制口、订阅文件、mihomo、本地 profiles
  selector/           Selector 组解析、候选、决策、DIRECT 基线
  tester/             多轮并发测速
  score/              指标与评分
  daemon/             定时自动选择、验证、回滚、历史
  optimize/           一键优化、多订阅联合测速、下载模式
  sidecar/            临时 mihomo 进程
  export/             配置过滤与导出影响预览
  netfetch/           订阅下载边界（重定向/大小/链路本地地址）
  probe/              目标文件 Range 下载探测
  probecache/         目标探测缓存
  health/             控制口/订阅/mihomo/Selector 组检查
  lock/               daemon 跨进程锁
  progress/           测速进度与取消
  results/            结果文件持久化
  backup/             设置/导出文件覆盖前备份
  store/              历史事件 JSONL
  settings/           settings.json、profiles、自动选择参数
  adapters/v2raya/    v2rayA REST 客户端
  mcp/                MCP stdio server
  cli/                CLI 命令实现
  localapi/           只给 Flutter 用的本机 HTTP API（token 校验）

plugins/dsh/          dsh 插件薄封装
packaging/            deb/nfpm、AppImage、Windows/macOS 构建脚本
.github/workflows/    Linux release、Windows/macOS desktop 构建
```

## 自动选择事务

`internal/selector` 负责安全规则和决策，`internal/daemon` 负责执行：

1. 解析可管理的 Selector 组；拒绝 `GLOBAL`。
2. 只取组内直接、可测节点；嵌套组和 DIRECT/REJECT 跳过。
3. 计算当前节点是否低于最低成功率。
4. 候选评分需超过当前 `margin`，并连续 `confirm` 轮成立。
5. 切换前记录 before 指标；切换后校验控制器当前节点。
6. 对新节点做真实 `delay` 验证；失败则回滚。
7. 写入结构化历史，Flutter 桌面端展示最近决策。

全部节点不达标时，`DIRECT` 基线用于区分本地网络问题与节点问题。

## 下载模式

下载模式与浏览模式分离：

1. 常规测速得到候选池。
2. 取前 5 个达标候选，全部不可达时最多补测下一批 5 个。
3. 用临时 mihomo sidecar 的 selector + mixed-port，对目标文件做 Range 下载。
4. 目标 TLS/EOF/超时/HTTP 4xx/5xx 的节点淘汰。
5. 记录 TTFB、p25/中位吞吐、卡顿；成功/失败分别缓存 20/5 分钟。
6. 同时探测直连；直连 p25 高于最佳节点 20% 时推荐直连，不切节点。
7. 否则只在下载开始前切换一次，沿用验证和回滚；下载过程中不切换。

## 本地服务安全

- 本机 API 只监听 `127.0.0.1`，Flutter 启动时生成随机 token，所有 `/api/*` 校验 Bearer token。
- API 设置 `Cache-Control: no-store`，拒绝跨站写请求。
- 订阅下载只允许 http/https，限制重定向、超时和 10 MiB 体积，拒绝链路本地/云元数据地址。
- 设置文件、结果、备份、锁使用 0600/0700。
- 签名下载 URL、secret 不写入日志或历史。

## 打包

- Linux：`make build desktop deb`，`make appimage`。
- Windows：`packaging/build-windows.ps1`，Flutter + NSIS。
- macOS：`packaging/build-macos.sh`，Flutter universal app + dmg。
- 详细步骤见 [packaging.md](packaging.md)。
