# UI 设计：Material Design 3

NodePulse 的 GUI 与本地 Web UI 共用同一套 UI 资源（`internal/uiassets/dist`），风格全面采用 **Material Design 3**，参考 FlClash 的桌面端信息架构。

## 设计原则

1. **一键优先**：首屏只保留“当前节点 + 一键优化”，高级参数收进“设置”。
2. **状态可见**：连接状态、当前节点、评分、推荐节点始终在首屏。
3. **渐进披露**：自动检测优先，检测不到时再展示手动配置与帮助。
4. **明暗双主题**：默认跟随系统，支持系统/亮/暗三态。
5. **M3 语义色**：所有颜色通过角色变量引用，不在组件里写死。

## 设计语言（从 FlClash 学到的）

- **Scaffold 结构**：Top App Bar 固定在内容区上方，内容在独立区域滚动；标题永远不覆盖滚动内容；
- **一个视图一个主动作**：仪表盘是一键优化，设置是保存；
- **表面分层**：page / card / high / highest 四级 surface，靠色阶、描边和轻阴影表达层级，不用装饰性渐变；
- **节奏**：4/8/12/16/20/24 的间距体系，卡片 16/20px 圆角，页面 24px 内边距；
- **排版层级**：页面标题 24/600，当前节点 26/600，卡片标题 15/600，正文 14/400，辅助 12/400；
- **导航**：桌面用 Navigation Rail，移动端用底部 Dock；没有品牌含义的徽标不出现；
- **状态**：chip + 圆点 + 颜色，状态不只用颜色表达；
- **动效**：只用于 hover/press/内容切换，短而克制，不做无意义动画。

## 色彩

使用 M3 角色（dark / light 两套）：

| 角色 | 用途 |
|---|---|
| `--md-primary` / `on-primary` | 主按钮、选中导航、评分环 |
| `--md-primary-container` | 强调背景 |
| `--md-secondary-container` | tonal 按钮、激活项 |
| `--md-surface` | 页面背景 |
| `--md-surface-container` | 卡片 |
| `--md-surface-container-high/highest` | hover、输入框、嵌套面板 |
| `--md-on-surface` / `on-surface-variant` | 主/次文本 |
| `--md-outline` / `outline-variant` | 描边、分隔线 |
| `--md-error` / `--md-success` / `--md-warning` | 状态色（成功/警告为扩展） |

## 排版

`--font` 优先 Google Sans / Roboto / Noto Sans SC / system-ui。

| 级别 | 用法 |
|---|---|
| 24px / 600 | 页面标题 |
| 26px / 600 | 当前节点 |
| 15px / 600 | 卡片标题 |
| 14px / 400 | 正文 |
| 12px / 400 | 辅助文本 |
| 11px / 500 | 标签、节点表头 |

## 主题

- 默认 **跟随系统**（`prefers-color-scheme`），不写死明暗；
- 三态：跟随系统 → 亮色 → 暗色，设置页有分段控件，Rail/Dock 的主题按钮也会循环；
- 选择保存在 `localStorage`（`nodepulse.themeMode.v2`），系统模式时监听系统变化实时切换；
- 首屏在 `<head>` 内联脚本里先设 `data-theme`，避免闪烁。

## 主题色 Seed

- 设置页提供主题色选择器 + 7 个预设（蓝/紫/青/绿/橙/红/粉）；
- 使用 Google 官方 `@material/material-color-utilities` 从 seed 生成完整的 M3 明暗色板（primary/secondary/tertiary/surface/outline/error）；
- 默认 seed：`#0b57d0`；自定义 seed 保存在 `localStorage`（`nodepulse.seed.v1`）；
- 生成结果注入到 `#seed-theme` 的 `<style>`，同时更新 `theme-color`；
- 三方库与许可证见 `THIRD_PARTY_NOTICES.md`；
- Seed 是 **client-local** 设置：GUI 和 Web UI 各自保存自己的选择。

## 字体

使用系统字体栈，不加载远程字体：

```css
system-ui, -apple-system, "Segoe UI", "Noto Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif
```

## 顶部操作栏

采用**自定义 Material 3 Top App Bar**，而不是假装成系统原生标题栏：

- 保留系统原生窗口边框/窗口按钮（Wails 默认窗口），拖拽、缩放、贴边交给系统；
- 应用内部用自定义 Top App Bar 放标题、连接状态、版本、刷新等操作；
- Top App Bar 属于 Scaffold，固定在内容区顶部，**内容在下方独立滚动**，不会出现标题压住滚动内容的情况；
- 移动端 Web 使用同一个 Top App Bar，配合底部 Dock；
- 这与 FlClash 的做法一致：原生窗口 + 应用内自定义 AppBar。

## 导航栏

- 桌面：左侧 Navigation Rail，只放图标 + 标签，不放置没有品牌含义的装饰徽标；
- 移动：底部 Navigation Dock，保留同样 4 个目的地 + 主题切换；
- 当前项用 secondary-container 背景表达，不只靠颜色。

## 响应式

- `>900px`：左侧 Navigation Rail（带文字）；
- `700–900px`：Rail 收成图标，节点表隐藏类型/抖动列，保留 OK / P50 / P90 / 评分；
- `<700px`：隐藏 Rail，底部 Navigation Dock（仪表盘/节点/设置/历史/主题），订阅行改为上下布局。

## 形状与组件

- 卡片：16px 圆角，1px `outline-variant` 描边；
- 按钮：全圆角（pill），高度 40/48px；Filled / Tonal / Outlined / Text 四种；
- 输入框：12/12/0/0 圆角 + 底部描边（M3 filled text field）；
- 开关：52×32 轨道 + 24/28 thumb；
- 芯片：8px 圆角，Filter chip 选中用 secondary-container；
- 评分环：conic-gradient，主色表示 0–100；
- Snackbar：底部居中，错误态使用 error 色。

## 信息架构

左侧 Navigation Rail（M3）：

1. **仪表盘**：当前节点、一键优化、连接、订阅、自动优化、结果；
2. **节点**：完整排名、搜索、选中切换；
3. **设置**：控制口、secret、组、订阅、mihomo、URL、并发、超时、导出；
4. **历史**：自动优化事件。

窄屏（<900px）隐藏 Rail 文字标签，节点表隐藏类型与抖动列，保留 OK / P50 / P90 / 评分。

## 交互

- **一键优化**：测速 → 评分 → 自动切换 → Snackbar + 评分环；
- **强度**：快速 3 轮 / 标准 5 轮 / 精确 10 轮；
- **自动优化**：M3 Switch，开启后后台 daemon 持续工作；
- **订阅导入**：URL 下载 → 解析校验 → 保存到 `~/.config/nodepulse/subscriptions/`；
- **错误**：Snackbar 显示，不阻塞界面；
- **主题**：Rail 底部按钮切换明暗。

## 实现

- `internal/uiassets/dist`：HTML / CSS / JS / web-shim；
- `gui/`：Wails 直接调用 Go 绑定；
- `internal/webui`：同一套静态资源 + `web-shim.js` 把 `window.go.main.App` 映射到 HTTP API；
- 主题变量集中在 `style.css` 的 `:root` 与 `[data-theme="light"]`。

## 从 FlClash 借鉴的点

浅克隆 `chen08209/FlClash` 后，只借鉴设计/交互，不复制其 GPL 代码：

- 自定义 AppBar + 应用内操作，不伪装系统标题栏；
- 卡片统一圆角/内边距/轻阴影，卡片头用图标 + 标题 + trailing；
- 状态 chip（运行/连接/警告）与状态色；
- 桌面侧栏 + 移动端底部 Dock 的响应式切换；
- 断点驱动的密度变化（隐藏次要列）；
- 不使用装饰性渐变，层级靠 surface 色阶与描边/阴影。

## 对比度与可访问性

- 暗色：正文 `12.68:1`、次要文字 `9.60:1`、描边 `5.14:1`；
- 亮色：正文 `14.74:1`、次要文字 `8.01:1`、主按钮 `6.39:1`；
- 亮色警告色已从 `#8a6d00` 调到 `#7a5f00`（卡片上 `5.23:1`），满足 4.5:1；
- `color-scheme: dark/light` 让原生 `<select>`、滚动条、输入控件跟随主题；
- `:focus-visible` 统一 2px primary 焦点环，不再出现浏览器默认黄色；
- `prefers-reduced-motion: reduce` 关闭非必要过渡；
- 移动端操作按钮高度提升到 44px，底部导航项 ≥ 48px。

## 参考

- Material Design 3：color roles / typescale / shape / navigation rail / switch / snackbar；
- FlClash：首屏信息层级、节点列表、连接状态与一键操作。
