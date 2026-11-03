# 路线图

## 已完成

### 核心与 CLI

- [x] Clash / mihomo YAML 解析，保留未知节点字段
- [x] 信息伪节点识别
- [x] Clash REST 客户端（version / proxies / delay / switch）
- [x] mihomo sidecar：随机端口、临时配置、自动清理
- [x] 多轮测试引擎（成功率、P50/P90、抖动、评分）
- [x] 结果 JSON、CLI 排名、过滤导出
- [x] base64 分享链接订阅解析
- [x] v2rayA REST：login / touch / connect（CLI）
- [x] v2rayN：导出 Clash 配置或 base64 订阅
- [x] MCP server：节点、测速、排名、切换、导出、状态、历史

### 自动选择

- [x] `internal/selector`：Selector 组安全解析、候选、阈值、滞后、确认
- [x] 永不自动管理 `GLOBAL`；只测管理组直接节点，不回退全量代理
- [x] 切换后验证、真实延迟复测、失败回滚
- [x] 全部不达标时用 `DIRECT` 基线区分网络/测试 URL 问题
- [x] 结构化历史：原因码、置信度、候选数、评分、回滚、耗时
- [x] 首启/设置页环境检查
- [x] GUI/Web/CLI 共用的 daemon 单写者锁
- [x] 手动测速进度与取消
- [x] 自动化参数持久化、重载恢复、环境切换

### 桌面与 Web

- [x] Wails GUI + 本地 Web UI，共用 Preact/Material 3 资源
- [x] 环境 profiles 快速切换
- [x] 导出影响预览、覆盖前备份、设置导出/恢复
- [x] GUI 结果跨重启保留
- [x] 托盘驻留、可选开机自启、切换/回滚通知
- [x] Linux `.deb` / AppImage；Windows zip；macOS app（CI 脚本）
- [x] 应用图标与 Wails 图标嵌入
- [x] WebKitGTK Wayland 滚动性能兼容开关
- [x] 多订阅联合测速：读取本地 profile，跨机场排名并标注来源
- [x] 下载模式：目标站点探测、不可达淘汰、p25 吞吐、缓存、一次切换
- [x] 下载模式直连基线：直连明显更快时推荐直连，不切节点

## 仍需完成

### 发布级

- [ ] Windows/macOS CI 首次实跑并修正平台差异
- [ ] Windows 代码签名 / macOS Developer ID + notarization
- [ ] 可选：MSI / dmg 安装器
- [ ] 修复 `release.yml` 中 AppImage 失败被 `|| true` 静默吞掉的问题
- [ ] 发布前跑一次完整 race test、跨平台打包和 GUI smoke

### 客户端适配

- [ ] v2rayA 接入 GUI/Web 环境与一键优化（目前仅 CLI）
- [ ] v2rayN “应用并重启”模式
- [ ] sing-box JSON 解析（可选）
- [ ] FlClash / Clash Verge 真机集成回归（已有用户环境，但未自动化）

### 产品边界

- [ ] 跨机场自动应用：目前只能联合测速/推荐，不能在两个独立组之间安全自动切换
- [ ] 若要做，需要显式父子组和双组事务；默认不做
- [ ] 订阅 URL 自动更新/到期提醒：按用户要求暂不做
- [ ] 站点专用下载链接抓取：不做；通用性不足，维护成本高
- [ ] Android/iOS 客户端：需要独立移动项目，不是桌面打包步骤

## Android / APK 结论

当前仓库不能直接打包成可用的 Android APK：

- GUI 是 Wails + WebKit/WebView 桌面应用，没有 Android 运行时；
- 核心依赖本机 Clash/mihomo 控制口、订阅文件、sidecar 和 daemon；
- 当前 Web 服务默认只绑定 `127.0.0.1`，没有远程认证/TLS，不能安全地给手机访问。

如果移动端是明确目标，应单独立项：原生 Android UI + 前台服务 + 手机本地 Clash 控制口，或带认证/TLS 的桌面远程模式。不要用 WebView 套壳假装成产品。
