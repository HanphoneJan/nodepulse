# UI 设计：Material Design 3

NodePulse 的 GUI 与本地 Web UI 共用同一套 UI 资源（`internal/uiassets/dist`），风格全面采用 **Material Design 3**，参考 FlClash 的桌面端信息架构。

## 设计原则

1. **一键优先**：首屏只保留“当前节点 + 一键优化”，高级参数收进“设置”。
2. **状态可见**：连接状态、当前节点、评分、推荐节点始终在首屏。
3. **渐进披露**：自动检测优先，检测不到时再展示手动配置与帮助。
4. **明暗双主题**：`data-theme="dark|light"` 切换，记忆在 `localStorage`。
5. **M3 语义色**：所有颜色通过角色变量引用，不在组件里写死。

## 色彩

使用 M3 角色（dark / light 两套）：

| 角色 | 用途 |
|---|---|
| `--md-primary` / `on-primary` | 主按钮、选中导航、评分环 |
| `--md-primary-container` | 强调背景（hero 渐变） |
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

## 参考

- Material Design 3：color roles / typescale / shape / navigation rail / switch / snackbar；
- FlClash：首屏信息层级、节点列表、连接状态与一键操作。
