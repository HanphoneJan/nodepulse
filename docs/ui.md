# UI 设计

NodePulse 客户端（桌面/Android）只使用 Flutter；界面层沿用 FlClash 的设计语言（GPL-3.0），Go 后端只提供本机 API。Android 复用同一套页面，窄屏使用底部 Dock。

这层设计语言不是“色板 + 字号”那么简单。真正决定相似度的是六层：**形状、表面、字体、密度、图标、动效**。只改颜色和字号，结果仍会是 Material 默认件。

## 结构

```text
desktop/
├── lib/
│   ├── main.dart              # 窗口、托盘、sidecar 启动
│   ├── app.dart               # MaterialApp、壳层、侧栏、顶栏、页面容器
│   ├── api/api_client.dart    # 本地 HTTP API
│   ├── state/app_state.dart   # 设置、状态、测速、主题
│   ├── platform/sidecar.dart  # 启动/回收 Go 后端
│   ├── format.dart
│   ├── ui/
│   │   ├── app_theme.dart     # ThemeData 装配、AppType、AppSurfaces
│   │   ├── window_header.dart # Windows 32px 窗口头与置顶/最小化/最大化/关闭
│   │   ├── widgets.dart       # 卡片、按钮、分段、开关、输入、节点卡
│   │   └── flclash/           # 从 FlClash 移植的 GPL 文件
│   │       ├── color.dart
│   │       ├── metrics.dart
│   │       ├── typography.dart
│   │       ├── shape.dart     # AppCorner / AppShape / AppInputBorder
│   │       ├── surface.dart   # SurfaceLevel / SurfaceScope / FocusRing
│   │       └── font.dart      # CJK / emoji fallback
│   └── views/                 # dashboard / nodes / settings / history
├── assets/
│   ├── icon.png
│   └── fonts/                 # Twemoji、JetBrains Mono
├── test/
└── pubspec.yaml
```

## 设计系统

### 形状

统一使用超椭圆 `RoundedSuperellipseBorder`，不再用普通 `RoundedRectangleBorder`。

| 用途 | 圆角 |
|---|---|
| 普通卡片 / 设置分组 / 输入框 / 菜单 | 16 |
| 仪表盘卡片 | 32 |
| 卡内块 / SurfaceBox | 20 |
| Tab / 分段容器 | 16 |
| 徽标 | 14 |
| 对话框 | 34 |
| 底部 Sheet | 28 |
| SnackBar / Tooltip | 4 |
| 按钮 / Chip / 分段滑块 | 全圆 |

### 表面

沿用 `SurfaceLevel` 四层：page → card → block → inset；floating 浅色取 card、深色取 block。

- `AppCard` 默认取 `context.raisedSurfaceLevel`，卡内嵌套自动递进；
- `SurfaceBox` 默认取下一层，不再到处硬编码 `scheme.inset`；
- 仪表盘卡片使用 e1 三层阴影；普通卡片保持平面。

### 字体

- Windows 主字体 `Microsoft YaHei UI`，Linux 主字体 `Noto Sans CJK SC`，macOS 主字体 `PingFang SC`；CJK 字体放在 primary 而不是只做 fallback，避免 Windows 回退到宋体后 `置` 这类多横字显得拥挤、横线过细；
- fallback 再依次补 `Microsoft YaHei` / `DengXian` / `SimHei`、Linux CJK 字体与系统拉丁字体；
- 内置 `Twemoji.Mozilla.ttf`，修复节点名 emoji 显示成方框；
- 内置 `JetBrainsMono-Regular.ttf`，用于需要等宽的少量场景；
- 数字统一 `FontFeature.tabularFigures()`，大数字加 `letterSpacing: -0.02 * size`；
- 所有文字角色使用 `TextLeadingDistribution.even` 与字母基线，和 FlClash 的 `ThemeData._textStyle` 一致；行框内的文字不会上下偏。

字号角色：

| 角色 | 值 |
|---|---|
| 页面标题 | 24 / 30 semibold |
| 设置分组标题 | 17 / 22 semibold，`onSurfaceVariant` 80% |
| 区块标题 | 17 / 22 semibold |
| 卡片标题 | 13 / 18 semibold，`onSurfaceVariant` |
| 节点名 | 28 / 34 semibold |
| 大数字 | 40 / 44 bold，tabular |
| 正文 | 17 / 22 |
| 行标题 / 次级文字 | 15 / 20 |
| 说明 / 卡片标签 | 13 / 18 |
| 辅助标签 | 12 / 16 |

内容区不再重复页面栏已经显示过的“节点 / 历史”标题。

### 密度与间距

- 间距只用 4 / 8 / 12 / 16 / 20 / 24；
- 桌面卡片间距 16，笔记本 12，窄屏 8；
- 仪表盘卡内边距 22/20（普通卡 20），窄屏 20/18；
- 页面内边距 16 / 24 / 24；
- 行高 56，控件高 40 / 44，字段高 48，工具栏 72；
- 桌面使用 `VisualDensity.compact` + `MaterialTapTargetSize.shrinkWrap`，Android 保持标准密度；
- 固定高度控件按 `MediaQuery.textScaler` 计算最小高度，Windows 文本大小调到 125% / 150% 时不裁剪；
- 页面最大宽度：仪表盘 1440，节点 1280，设置/历史 960，结果表 1080；
- 响应式断点：<760 单列，<900 紧凑，≥900 四列仪表盘 / ≥1080 三列节点。

### 图标

- 使用 `CupertinoIcons` + `Glyphs` 统一常量，保持 1.5–2px 线性、圆头端点；
- 尺寸 18 / 20 / 24，颜色只用语义色；
- 不再直接使用 `Icons.*`。

### 动效

- 200ms 标准过渡，250ms 分段滑块，320ms 侧栏 indicator；
- 统一 `Curves.easeOutCubic`；
- 只保留焦点环、选中态、进度和页面切换，不做装饰性动画。

## 组件

`desktop/lib/ui/widgets.dart` 提供：

- `AppCard`：表面感知、16/32 圆角、可选选中/错误/阴影/点击；
- `AppCardHeader` / `InfoHeader`：18 图标 + 13/18 卡片标题 + trailing；`section: true` 时使用 17/22 设置分组标题；
- `StatusChip`：成功/警告/错误/中性四态，26px 胶囊（随文本缩放增高）；
- `AppButton` / `ActionPill` / `GhostButton`：40px 胶囊，filled/tonal/outline/ghost；
- `SegmentedChoice`：44px、4px inset、primary 滑块、250ms 动画；
- `AppSwitch`：52×32 自绘轨道，替换 Material `Switch`；
- `AppTextField`：48px、标签在字段上方（15/20 `onSurfaceVariant`）、字段内 15/20 正文、1px 细边 + 焦点主色 1.5px、密码显隐；
- `AppSelect`：48px、同样的标签/描边/菜单样式；
- `SettingRow`：56px 行高、15/20 副标题、右 trailing；
- `NodeCard` / `NodeRow`：节点卡片与紧凑结果行，共用评分/P90/抖动/成功率；
- `SurfaceBox`：自动递进的内容块；
- `ScoreRing`：自绘评分环；
- `EmptyState`、`LoadingList`、`showAppConfirm`。

新增组件必须复用同一套形状、表面、字号和间距，不要写裸圆角/裸字号/裸颜色。

## 页面

- **应用壳**：Windows 顶部是 FlClash 风格的 32px 自绘窗口头（拖动、双击最大化、置顶、最小化、最大化/还原、关闭），下面是 72px 页面栏；macOS / Linux 保留原生标题栏。侧栏 48/220 可展开（40px item、10px 圆角、3px indicator、文字标签）；窄屏使用 64px 浮动 Dock。
- **仪表盘**：当前节点（2 列）+ 推荐评分 + 连接状态；控制条（强度/只测速/全部订阅/自动优化/间隔）；当前环境/最近决策/结果摘要；订阅 + 下载模式；测速结果表。
- **节点**：计数 + 搜索 + 全部/达标/失败筛选（页面标题只在顶部页面栏出现）；桌面 2–3 列 `NodeCard`，窄屏单列；选中后显示切换操作条。
- **设置**：960 宽分组卡；外观（主题色 + 三态模式）、环境、测速与自动选择、连接设置、环境检查、后台与通知、备份与迁移、关于/检查更新。
- **历史**：日期分组时间线；计数 + 全部/切换/回滚/错误筛选（不重复页面栏标题）；事件 chip、原因、分数、回滚结果。

## 主题

- 主题设置保存在 Flutter `shared_preferences`；
- seed 色生成完整明暗色板；
- 顶部栏、卡片、浮层全部使用语义色，不硬编码颜色；
- 深色模式使用 FlClash 的表面递进，不做纯黑（除非后续单独提供）。

## 平台能力

- Android：Go sidecar 以 `libnodepulse_core.so` 打包，APK 需要连接开放的 Clash/mihomo 控制口；
- `window_manager`：窗口尺寸、关闭到托盘；Windows 使用 32px 自绘窗口头（拖动/双击最大化/右键系统菜单/置顶/最小化/最大化/关闭），macOS 与 Linux 保留原生标题栏；
- `tray_manager`：托盘菜单（显示/退出）；
- `launch_at_startup`：开机自启；
- 更新检查：启动后 24 小时内最多检查一次 GitHub Release，设置页可手动检查并打开下载页；
- Go sidecar：Flutter 启动时生成随机 token 和端口，退出时回收。

## 可访问性

- 所有图标按钮提供 tooltip 或语义标签；
- 触控目标不小于 40px；
- 危险操作使用 34 圆角确认对话框；
- 状态变化通过 SnackBar 提示；
- 表单保留键盘焦点与显式标签。

## 构建

```bash
make build      # Go CLI 和 sidecar
make desktop    # Flutter Linux release
make run        # Flutter 开发运行
make analyze    # flutter analyze
make test       # Go + Flutter 测试
```
