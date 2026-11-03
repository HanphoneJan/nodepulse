# UI 设计

NodePulse 桌面端只使用 Flutter；不再包含 Wails、Preact、HTML 或 CSS 界面。界面层直接复用 FlClash 的设计系统（GPL-3.0），Go 后端只提供本机 API。

## 结构

```text
desktop/
├── lib/
│   ├── main.dart              # 窗口、托盘、sidecar 启动
│   ├── app.dart               # MaterialApp、顶部栏、导航、页面切换
│   ├── api/api_client.dart    # 本地 HTTP API
│   ├── state/app_state.dart   # 设置、状态、测速、主题
│   ├── platform/sidecar.dart  # 启动/回收 Go 后端
│   ├── format.dart
│   ├── ui/
│   │   ├── app_theme.dart     # ThemeData、表面层级
│   │   ├── widgets.dart       # CommonCard、InfoHeader、按钮、选择器
│   │   └── flclash/           # 从 FlClash 移植的 GPL 文件
│   └── views/                 # dashboard / nodes / settings / history
├── assets/icon.png
├── test/
└── pubspec.yaml
```

## 设计系统

配色、字号和间距不自行发挥，直接采用 FlClash 的数值：

| 项目 | 值 |
|---|---|
| 默认强调色 | `#007AFF` |
| 配色算法 | `SchemeContent` vivid，低彩度表面 |
| 页面标题 | 24/30 semibold |
| 卡片标题 | 17/22 semibold |
| 正文 | 17/22 |
| 次级文字、按钮 | 15/20 |
| 说明、标签 | 13/18 |
| 极小字 | 11/13 |
| 卡片内边距 | 16px |
| 卡片间距 | 16px |
| 按钮高度 | 40px |
| 字段高度 | 56px |
| 列表行 | 56px |
| 页面工具栏 | 72px |

表面分四层，与 FlClash `SurfaceLevel` 一致：

- page：浅色为 `surfaceContainer`，深色为 `surface`；
- card：浅色为 `surfaceBright`，深色为 `surfaceContainer`；
- block：浅色为 `surfaceContainer`，深色为 `surfaceContainerHigh`；
- inset：`surfaceContainerHighest`。

## 组件

`desktop/lib/ui/widgets.dart` 提供：

- `CommonCard` + `InfoHeader`：圆角 12、无阴影、16px 内边距；
- `StatusChip`：28px 胶囊，成功/警告/错误三态；
- `ActionPill` / `GhostButton`：40px 胶囊按钮；
- `SegmentedChoice`：40px 分段控件；
- `AppSelect`：48px 下拉框；
- `SettingRow`：56px 列表行，标题 17、副标题 13；
- `SurfaceBox`：inset 层内容块；
- `ScoreRing`：自绘评分环；
- `EmptyState`、`LoadingList`。

新增组件必须复用同一套字号、间距和表面层级，不要写裸字号。

## 页面

- **仪表盘**：当前节点、状态、一键优化/只测速/全部订阅、强度；连接与环境检测；后台自动优化；下载模式；测速结果。
- **节点**：搜索、排名、选择、切换。
- **设置**：外观（主题模式 + 8 个预设色）、环境、自动选择、环境检查、备份与迁移、连接设置、控制口状态。
- **历史**：切换/回滚/错误、原因、时间、评分、回滚结果。

## 主题

- 主题设置保存在 Flutter `shared_preferences`；
- seed 色生成完整明暗色板；
- 顶部栏、卡片和浮层全部使用语义色，不硬编码颜色。

## 桌面能力

- `window_manager`：窗口尺寸、关闭到托盘、原生标题栏；
- `tray_manager`：托盘菜单（显示/退出）；
- `launch_at_startup`：开机自启；
- Go sidecar：Flutter 启动时生成随机 token 和端口，退出时回收。

## 可访问性

- Material 控件自带键盘与焦点行为；
- 所有图标按钮提供 tooltip 或语义标签；
- 触控目标不小于 40px；
- 危险操作需要二次确认；
- 状态变化通过 SnackBar 提示。

## 构建

```bash
make build      # Go CLI 和 sidecar
make desktop    # Flutter Linux release
make run        # Flutter 开发运行
make analyze    # flutter analyze
make test       # Go + Flutter 测试
```
