# UI 设计

NodePulse 的桌面 GUI 和本地 Web UI 共用 `internal/uiassets/dist` 下同一套前端资源。配色和控件密度直接对齐 FlClash：默认强调色 `#007AFF`，`SchemeContent` vivid 方案，低彩度中性色表面；预设色也沿用 FlClash 的 8 个颜色。

## 原则

1. **一键优先**：首屏保留当前节点、推荐节点、一键优化和必要的测速入口；高级参数收进设置。
2. **状态可见**：连接状态、当前节点、评分、最近自动决策始终可查。
3. **渐进披露**：自动检测优先；失败时再展示手动配置和检查结果。
4. **明暗主题**：跟随系统，支持 system/light/dark；按 FlClash 的 vivid 方案从 seed 生成色板。
5. **语义 token**：颜色、间距、字号、圆角、阴影、层级、动效都走 token，组件不写裸值。
6. **离线优先**：不加载远程字体、CDN 或远程图标；三方资源本地固定版本。

## 技术结构

```text
internal/uiassets/
├── dist/
│   ├── index.html
│   ├── app.js                 # 入口
│   ├── css/
│   │   ├── 00-tokens.css      # primitive / M3 / semantic / component token
│   │   ├── 01-base.css        # reset、字体、焦点、滚动条、reduced-motion
│   │   ├── 02-layout.css      # rail、topbar、content、dock
│   │   ├── 03-components.css
│   │   ├── 04-views.css
│   │   └── 05-responsive.css
│   ├── js/
│   │   ├── main.js            # App root、状态、动作
│   │   ├── api.js             # Wails / HTTP 唯一后端边界
│   │   ├── events.js          # daemon/autostart 事件
│   │   ├── theme.js           # 三态主题 + seed 色板
│   │   ├── components.js      # Button/Select/Switch/ProfileMenu 等
│   │   ├── tables.js          # 节点/分组表格
│   │   ├── views.js           # Dashboard/Nodes/Settings/History
│   │   └── html.js            # Preact + htm
│   ├── vendor/                # Preact、htm、Floating UI、hooks
│   ├── icons.svg              # 由 genicons 生成
│   └── material-color-utilities.js
├── icons/                     # manifest + Material Symbols 子集
└── genicons/                  # make icons
```

前端使用 **Preact 10 + htm**，无 Node 构建。Floating UI 只负责下拉定位；其余控件优先使用原生 input/radio/checkbox + token 样式。

```bash
make icons     # 重新生成 icons.svg 与 favicon.svg
make ui-check  # JS 语法、图标资源和前端测试
```

新增前端依赖必须本地固定版本，并同步 `THIRD_PARTY_NOTICES.md`、`packaging/nfpm.yaml` 和 AppImage 许可证复制。

## 布局

- 桌面：固定 icon rail + 顶部 Top App Bar + 独立滚动内容区。
- 窄屏：底部导航 dock；内容单列，卡片全宽。
- 顶部栏不覆盖内容；滚动容器独立，避免滚动时标题遮挡。
- 卡片使用 surface 分层 + 细边框，12px 圆角、16px 内边距，不使用装饰性渐变。
- 控件密度对齐 FlClash：按钮 38px、字段 48px、列表行 42px、顶部栏 56px。
- 页面只保留标题，不放副标题和说明段；解释性文字只保留一行，详细说明放进可展开的“没检测到？”。
- Windows 桌面窗口标题栏跟随应用明暗主题；主题切换入口只在“设置 → 外观”，顶栏不再放重复按钮。

## 视图

### 仪表盘

- 当前节点、连接状态、推荐节点、评分环；
- 一键优化、只测速、全部订阅、强度选择、取消；
- 连接/控制口、订阅、后台自动优化、最近自动决策；
- 下载模式：目标文件 URL、测速并选节点、直连/节点探测结果。

### 节点

- 按成功率、评分、P90、抖动排序；
- 搜索、刷新控制口状态、选择节点、切换；
- 联合测速结果显示来源标签；下载模式结果禁用误切换。

### 设置

- 主题：模式 + seed 色；
- 环境：保存/新建/重命名/删除，快速切换；
- 自动选择：余量、确认轮数、最低成功率、检查间隔；
- 后台与通知（桌面版）：关闭到托盘、切换/回滚通知、开机自启；
- 环境检查：控制口、Selector 组、订阅、mihomo、测试 URL；
- 导出影响预览、确认导出、最小配置警告；
- 备份与迁移：导出设置、从备份恢复；
- 连接参数、订阅、控制口状态。

### 历史

- cycle/switch/rollback/error 事件；
- 原因码、前后评分、置信度、回滚结果、时间。

## 交互

- **一键优化**：测速 → 评分 → 安全切换 → 验证/回滚 → 结果和历史。
- **只测速**：不切换，展示排名。
- **全部订阅**：读取本地 profiles，联合 sidecar 测速，只推荐。
- **下载模式**：目标文件 Range 探测，淘汰不可达节点，比较直连，只在开始前切换一次。
- **进度/取消**：测速和下载探测都有进度、当前节点、取消。
- **通知**：只在窗口隐藏/最小化时发送切换、回滚、失败。

## 主题与 token

- primitive → M3 role → semantic → component token 四层。
- 颜色全部使用语义 token；`color-mix()` 提供 `@supports` fallback。
- 字体使用系统字体；图标使用本地 Material Symbols 子集。
- 动效短、可关闭；`prefers-reduced-motion` 生效。
- GUI 卡片不使用大范围模糊阴影，避免 WebKitGTK 滚动重绘；浮层保留轻量阴影。
- Linux Wayland 默认关闭 WebKitGTK DMA-BUF 渲染以规避部分 Intel 驱动滚动卡顿；可用 `NODEPULSE_WEBKIT_DMABUF=1` 恢复。

## 可访问性

- 所有交互控件有可见 label 或 `aria-label`；
- 下拉、环境菜单支持方向键、Home/End、Escape；
- 表格使用 listbox/option 语义，支持上下键选择；
- 焦点可见，对比度满足 M3 语义色；
- 状态变化通过 `aria-live` 或 Snackbar 通知；
- 触摸目标满足移动端最小尺寸。

## GUI / Web 差异

| 能力 | GUI | Web |
|---|---|---|
| 托盘/通知/开机自启 | 有 | 无 |
| 关闭窗口驻留 | 有 | — |
| 文件选择器 | 原生对话框 | 浏览器下载/上传 |
| 设置保存 | Wails binding | HTTP API |
| 事件 | Wails EventsOn | 轮询 `/api/history` |

Web 与 GUI 共享同一套视图和 API 封装；组件不直接访问 `window.go` 或 `fetch`。
