# UI 设计：Material Design 3

NodePulse 的 GUI 与本地 Web UI 共用 `internal/uiassets/dist` 下同一套前端资源，风格采用 **Material Design 3**，信息密度与侧栏节奏参考 **FlClash**。

## 设计原则

1. **一键优先**：首屏只保留“当前节点 + 一键优化”；高级参数收进设置。
2. **状态可见**：连接状态、当前节点、评分、推荐节点始终在首屏。
3. **渐进披露**：自动检测优先，检测不到时再展示手动配置与帮助。
4. **明暗双主题**：默认跟随系统，支持系统/亮/暗三态。
5. **语义 token**：颜色、间距、字号、圆角、阴影、层级、动效全部走 token，组件里不写裸值。
6. **离线优先**：不加载远程字体、CDN、远程图标；所有三方资源本地固定版本打包。

## 技术结构

```
internal/uiassets/
├── dist/
│   ├── index.html          # 只保留 shell + 样式/脚本入口
│   ├── app.js              # 入口，import js/main.js
│   ├── css/
│   │   ├── 00-tokens.css   # primitive / M3 / semantic / component token
│   │   ├── 01-base.css     # reset、字体、焦点、滚动条、reduced-motion
│   │   ├── 02-layout.css   # Scaffold、rail、topbar、content、dock
│   │   ├── 03-components.css
│   │   ├── 04-views.css
│   │   └── 05-responsive.css
│   ├── js/
│   │   ├── main.js         # App root、状态、动作
│   │   ├── api.js          # Wails / HTTP 唯一后端边界
│   │   ├── events.js       # daemon 事件（Wails EventsOn / web 轮询）
│   │   ├── theme.js        # 三态主题 + seed 色板
│   │   ├── components.js   # Button/Select/Switch/Segmented/Seed/State
│   │   ├── tables.js       # 节点列表 / 控制口状态表
│   │   ├── views.js        # 四个视图
│   │   └── html.js         # Preact + htm
│   ├── vendor/             # Preact、htm、Floating UI（本地固定版本）
│   ├── icons.svg           # 由 genicons 生成，禁止手写
│   └── material-color-utilities.js
├── icons/                  # manifest + 固定版本 Material Symbols 子集
└── genicons/               # `go run ./internal/uiassets/genicons`
```

前端使用 **Preact 10 + htm**（无 Node 构建），只引入行为复杂的三方库：Floating UI 负责下拉定位；其余控件优先使用原生 input/radio/checkbox + token 样式。`make icons` 生成图标 sprite，`make ui-check` 做静态一致性检查。

## 色彩

使用 M3 角色 + 语义映射：

| 层级 | Light | Dark |
|---|---|---|
| page | `surface-container` | `surface-dim` |
| card | `surface-bright` | `surface-container` |
| block / inset | `surface-container` / `highest` | `surface-container-high` / `highest` |
| menu | `surface-container-lowest` | `surface-container-high` |

卡片在亮色下比 page 亮、在暗色下比 page 亮，配合 FlClash 风格的 e1 阴影，避免“卡片陷下去”。所有组件只消费语义 token；`color-mix()` 均有 `@supports` fallback。

## 排版

只用系统字体栈：

```css
system-ui, -apple-system, "Segoe UI", "Noto Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif
```

| 级别 | 用法 |
|---|---|
| 24 / 32 · 600 | 页面标题 |
| 26 / 32 · 600 | 当前节点 |
| 16 / 24 · 600 | 卡片标题 |
| 14 / 20 · 400 | 正文 |
| 12 / 16 · 400 | 辅助文本 |
| 11 / 16 · 500 | 标签、表头 |

字号用 rem，间距用 4/8/12/16/20/24 的 4px 基线。

## 主题

- 默认 **跟随系统**（`prefers-color-scheme`）；
- 三态：跟随系统 → 亮色 → 暗色；设置页有 40px 分段控件，顶栏有循环切换按钮；
- **主题按钮只在顶栏**，侧栏和移动端底部 dock 都没有；
- 选择保存在 `localStorage`（`nodepulse.themeMode.v2`）；系统模式时监听系统变化；
- `<head>` 内联脚本先设 `data-theme`，并把上次生成的色板缓存注入 `<style>`，减少自定义 seed 的首屏闪烁。

## 主题色 Seed

- 设置页提供 7 个预设：蓝 / 紫 / 青 / 绿 / 橙 / 红 / 粉；默认 `#0b57d0`；
- “自定义…”按钮触发视觉隐藏的 `input[type=color]`（不是 `display:none`），并显示当前 hex；
- 使用 Google 官方 `@material/material-color-utilities` 0.3.0 生成 primary/secondary/tertiary/error/surface 全组角色；
- 由于 `themeFromSourceColor()` 返回的是 legacy Scheme，surface containers 从同一 neutral palette 的 M3 tone stop（4/6/10/12/17/22/24/87/90/92/94/96/98/100）补齐；这保留了默认蓝的 legacy primary，同时让红色 seed 的卡片不再停留蓝灰；
- 自定义 seed 存 `localStorage`（`nodepulse.seed.v1`），生成 CSS 缓存存 `nodepulse.seedCss.v1`；
- seed 是 **client-local**：GUI 与 Web UI 各自保存自己的选择。

## 顶部操作栏

- 自定义 Material 3 Top App Bar，保留 Wails 原生窗口边框；
- 右侧有**环境快速切换**按钮（swap 图标 + 当前环境名），菜单位于屏幕内并支持滚动；每项显示环境名、客户端 host、订阅文件名；底部“管理环境…”进入设置；
- Top App Bar 属于 Scaffold，固定在内容区上方，内容在下方独立滚动；
- 高度 72px（移动 56/52px），标题 24/32；
- 右侧：环境切换、主题循环按钮、连接状态 chip、版本号；版本来自 `internal/version`。

## 导航栏

- 桌面：72px 紧凑 Navigation Rail，56×56 项，24px 图标，无文字标签，原生 `title` 提示；
- 选中：8% on-surface 填充 + 左侧 3×16px primary 指示条；
- hover 4% / pressed 6% / focus 10% + inset 焦点环；
- 移动：底部 4 项 Navigation Dock（仪表盘/节点/设置/历史），无主题按钮；
- 两个导航都有 `aria-label`，当前项 `aria-current="page"`。

## 响应式

四档：

| 断点 | 行为 |
|---|---|
| >1000px | rail 72；内容 1120；设置 2 列；节点表 8 列；hero 2 列 |
| 720–1000px | 内容 padding 20；节点表 6 列（隐藏类型/抖动）；hero score 88 |
| <720px | 隐藏 rail，显示 4 项 dock；单列；hero 纵向；设置 1 列；snackbar 在 dock 上方 |
| <520px | 节点表 5 列（rank/name/OK/P90/score）；组表隐藏类型；seed 换行；搜索栏换行；顶栏副标题隐藏 |

`.app` 使用 `100vh` + `100dvh` fallback；内容底部和 `scroll-padding-bottom` 都计入 dock 与安全区。

## 形状与组件

- 卡片：16px 圆角、1px outline-variant、e1 阴影；
- 按钮：pill，高度 48/40/36；hover 4% / pressed 6% / focus 10%；loading 用 spinner + `aria-busy`；
- 输入框：52px filled + 12/12/0/0 圆角；focus 用 2px inset 环 + 2px 底部线；placeholder 用语义 token，保证 4.5:1；
- 下拉：自定义 M3 select-only combobox；`role=combobox` + `listbox`/`option`、`aria-expanded/controls/activedescendant`、方向键/Home/End/Enter/Escape/Tab、typeahead；Floating UI 做 flip/shift，fixed 定位避免被滚动容器裁切；
- 开关：原生 checkbox + `role=switch`，文字在 label 内，44px 命中区；
- 分段：40px 容器 + 32px 项 + 12.5px 字，原生 radio + 方向键；
- 节点列表：ARIA listbox + roving tabindex，Enter/Space 选择，Arrow/Home/End 移动；表头 28px、行 40/44px；
- 控制口状态：真实 `<table>`，固定布局，长文本 ellipsis + title；
- 评分环：SVG circle + stroke-dashoffset（不是 conic-gradient）；
- Snackbar：`role=status/alert` + `aria-live`，错误态 error-container，移动端避让 dock；
- 空/加载/错误：统一 `.state` / `.skeleton`，加载时 `aria-busy="true"`。

## 图标

- 图标来自固定版本 `@material-symbols/svg-400@0.48.2` 的 Rounded filled 子集；
- `internal/uiassets/icons/manifest.txt` 是唯一来源；`make icons` 生成 `dist/icons.svg` 与 `favicon.svg`；
- 禁止在业务 HTML/JS 里手写 `<path d="...">`；需要新图标时改 manifest + vendor SVG + 重新生成；
- 图标只通过 `<Icon name="...">` 使用，统一 `aria-hidden="true" focusable="false"`；图标按钮必须有 `aria-label`。

## 可访问性

- 全局焦点环使用 inset 2px primary（filled 按钮用 standoff 2px），不再出现外扩双线框；
- 文本输入、下拉、节点行都有可见焦点；
- 节点行可用键盘选择；分段用原生 radio；开关用原生 checkbox；
- 紧凑控件视觉尺寸 32–40px，触摸设备通过 44px 命中区/容器补偿；主要按钮、输入、导航 ≥44px；
- `prefers-reduced-motion: reduce` 关闭 spinner/评分环/菜单动效；
- 对比度：正文与次要文本在明暗主题均 ≥4.5:1；placeholder ≥4.5:1；焦点环 ≥3:1；
- 状态不只靠颜色：chip 有文字、推荐行有图标 + `sr-only` 文本、错误有文案。

## 环境（客户端 + 订阅）

- 一个**环境** = 控制口 + secret + 订阅配置 + Selector 组 + mihomo 路径 + 测试 URL / 轮次 / 并发 / 超时；
- 顶栏按钮和设置页“环境”卡片都可以一键切换；切换会立即刷新控制口状态；
- 设置页可重命名当前环境、保存当前设置到环境、新建环境（复制当前）和删除环境；至少保留一个；
- 环境保存在 `settings.json` 的 `profiles` / `active_profile`，顶层字段始终是当前环境，兼容旧调用方与旧配置文件；
- 新环境默认命名为“环境 N”，可立即重命名。

## 自动选择

- 仪表盘“后台自动优化”卡片显示最近一次自动选择：保持/切换/回滚、原因、当前与候选评分、置信度、回滚结果；
- 设置页“自动选择”卡片提供切换余量、连续确认轮数、最低成功率和检查间隔；保存后下次开启/重启自动优化生效；
- 自动优化开关与这些参数持久化在 `settings.json`，页面重载后若开关为开会自动恢复 daemon；
- 手动测速/一键优化与后台自动优化互斥；切换环境时会停止旧 daemon。

## 驻留与通知（桌面版）

- 设置页“后台与通知”卡片：关闭窗口驻留托盘、开机自动启动、切换/回滚通知；
- 开机自启默认关闭；开启后登录时隐藏启动，只显示托盘；
- 托盘菜单：打开面板、开机启动、退出；
- 通知只在窗口隐藏/最小化时发送切换、回滚、检查失败。

## 导出影响预览

- 点击“导出稳定配置”先进入预览，不直接写文件；
- 预览展示保留/排除数量、被排除节点示例、失去成员的分组、回退 `DIRECT` 的分组；
- 展示规则/DNS/TUN/rule-providers 是否保留；最小配置会明确警告这些内容将被替换；
- 用户点“确认导出”后才写入；覆盖已有文件前自动备份。

## 备份与迁移

- 设置页“备份与迁移”卡片提供“导出设置备份”“从备份恢复”；
- 恢复前自动备份当前设置；导入完成后重新加载环境和自动优化状态；
- 设置备份包含 secret，界面明确提示妥善保管。

## 多订阅联合测速

- 仪表盘提供“全部订阅”测速入口，不需要先把两个机场塞进同一个 Selector；
- 结果表在节点名下显示来源（FlClash/Verge profile 或订阅文件名）；
- 只做跨订阅排名和推荐，不自动切换未激活 profile；
- 未激活订阅通过本机 profile 文件读取，控制口只负责当前激活配置。

## 下载模式

- 仪表盘提供“下载模式”：粘贴直接文件 URL → 测速并选择节点；
- 结果展示 p25 吞吐、首字节、缓存命中、淘汰节点数；
- 下载模式只切换一次，不在下载过程中自动切换；
- 目标站点不可达的节点不会进入推荐；
- 默认一键优化/后台自动优化保持浏览优先逻辑。

## 结果持久化

- GUI 重启后恢复上次测速结果；
- 只有结果文件中的配置路径与当前环境一致时才恢复；
- 切换环境或清除结果会删除结果文件。

## 环境检查

- 设置页“环境检查”卡片显示控制口、订阅、mihomo、Selector 组的状态；
- 每项给出 ok/warn/error 和可执行建议；
- 可手动重新检查，不在每次打开页面时阻塞；
- 控制口不可达但存在本地订阅时，提示可以走 sidecar 测速。

## daemon 互斥

- GUI、Web、CLI 共用 `~/.local/share/nodepulse/daemon.lock`；
- 已有实例运行时，新 daemon 启动直接报错，不静默重复运行；
- 进程崩溃遗留锁会按 PID 存活情况清理。

## GUI 性能

- 滚动区域中的卡片不再使用大范围模糊 `box-shadow`，保留边框和表面分层；
- 菜单/提示等浮层保留轻量阴影；
- 不引入装饰性渐变，保持现有 Material 3 表面层次；
- Linux Wayland 默认关闭 WebKitGTK DMA-BUF 渲染以规避部分 Intel 驱动的滚动卡顿；需要恢复时设置 `NODEPULSE_WEBKIT_DMABUF=1`。

## 交互

- **一键优化**：测速 → 评分 → 自动切换 → Snackbar + 评分环；
- **只测速**：显示 skeleton + 已运行秒数，结果加载后原地替换；
- **强度**：快速 3 轮 / 标准 5 轮 / 精确 10 轮；
- **自动优化**：M3 Switch；右侧说明连续确认、失败回滚、滞后阈值；
- **订阅导入**：URL 下载 → 保存快照 → 刷新订阅列表；
- **导出**：默认保留原 rules / proxy-groups / DNS / TUN / rule-providers，过滤组内失效节点并追加 `NodePulse` url-test 组；CLI `--minimal` 可回到最小配置；Web 导出写到 `~/.local/share/nodepulse/exports/`；
- **错误**：Snackbar + 就地错误；表单字段不依赖 Snackbar 作为唯一提示；
- **主题**：顶栏按钮循环三态；设置页分段控件。

## 构建与验证

```bash
make icons          # 生成 icons.svg / favicon.svg
make ui-check       # 图标引用、无远程资源、无手写 path 等静态检查
make build gui deb
./bin/nodepulse serve --addr 127.0.0.1:8099
```

- GUI：`make gui` 使用 `production,webkit2_41`；
- Web UI 与 GUI 共享同一套 `dist`；
- 每次 UI 改动至少验证 1440×900、1280×800、390×844 与 light/dark；
- 提交前检查 `document.documentElement.scrollWidth === innerWidth`，且控制台无错误。

## 三方资源

| 依赖 | 版本 | 许可证 | 用途 |
|---|---|---|---|
| `@material/material-color-utilities` | 0.3.0 | Apache-2.0 | seed 色板 |
| `preact` | 10.26.9 | MIT | 组件/状态层 |
| `htm` | 3.1.1 | Apache-2.0 | 模板语法 |
| `@floating-ui/core` / `dom` | 1.7.3 / 1.7.4 | MIT | 下拉定位 |
| `@material-symbols/svg-400` | 0.48.2 | Apache-2.0 | 图标子集 |

许可证全文在 `internal/uiassets/dist/vendor/LICENSES/` 与 `dist/icons.LICENSE.txt`；`THIRD_PARTY_NOTICES.md`、`packaging/nfpm.yaml`、`packaging/build-appimage.sh` 均已同步。不得使用 CDN、远程字体或图标字体。

## 参考

- Material Design 3：color roles / typescale / shape / navigation rail / switch / snackbar / combobox；
- FlClash：紧凑 side rail、表面层级、状态层、卡片节奏、底部 dock；
- WAI-ARIA APG：Select-Only Combobox、Listbox、Radio Group。
