# 开发

## 环境

- CLI/核心：Go 1.24+
- 客户端：Flutter stable（Dart 3.10+）；Linux 桌面需要 GTK3、`clang`、`cmake`、`ninja`、`pkg-config`；Android 需要 Android SDK、Java 21
- dsh 插件：Node.js >= 22
- `mihomo`：sidecar 测速/下载探测
- `nfpm`：`.deb`；`appimagetool`：AppImage
- Windows/macOS 安装包必须在对应平台或 CI runner 上构建；Android APK 在 Linux/CI 上构建

Linux 构建依赖：

```bash
sudo apt-get install libgtk-3-dev liblzma-dev libblkid-dev clang cmake ninja-build pkg-config
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
```

## 常用命令

```bash
make build      # bin/nodepulse
make desktop    # Flutter Linux release + sidecar
make android    # Flutter Android APK + arm64 Go sidecar
make run        # Flutter 开发运行
make test       # Go + Flutter 测试
make analyze    # flutter analyze
make race       # race 测试
make vet        # go vet
make fmt        # gofmt + dart format
make deb        # .deb
make appimage   # AppImage
make clean
```

Windows/macOS 构建见 [packaging.md](packaging.md)。

## 测试配置

```bash
./bin/nodepulse nodes -f testdata/subscription.yaml
./bin/nodepulse test -f testdata/subscription.yaml --mihomo /path/to/mihomo --rounds 3
./bin/nodepulse rank -i results.json
```

真实控制口：

```bash
./bin/nodepulse test -f sub.yaml --api http://127.0.0.1:9090 --secret <secret>
./bin/nodepulse daemon --api http://127.0.0.1:9090 --group "节点选择" --once
```

## 客户端

Flutter 客户端是独立工程 `desktop/`，FlClash 设计系统文件在 `desktop/lib/ui/flclash/`，GPL-3.0；桌面和 Android 共用同一套页面。

```bash
cd desktop
flutter pub get
flutter analyze
flutter test
flutter run -d linux
```

新增桌面依赖时必须：

1. 在 `desktop/pubspec.yaml` 固定版本；
2. 更新 `THIRD_PARTY_NOTICES.md`；
3. 同步打包脚本和许可证。

Android 构建：

```bash
make android   # 需要 Android SDK、Java 21
```

Go sidecar 以 `libnodepulse_core.so` 打包进 `jniLibs/arm64-v8a`，Flutter 通过 MainActivity 的 MethodChannel 取得 `nativeLibraryDir` 后启动它。Android 端不内置 TUN/VPN，仍需连接一个开放的 Clash/mihomo 控制口。

## 代码约定

- 公共包放 `internal/`，避免泄露实现细节。
- 自动选择统一使用 `internal/selector`；不要在 daemon/optimize 里重复阈值逻辑。
- 任何切换都必须验证并支持回滚。
- 不提交 secret、settings、results、签名下载 URL、测试截图。
- 面向用户的错误信息用中文；日志/包注释可英文。
- 提交信息使用 Conventional Commits（`feat:`、`fix:`、`docs:`、`build:` 等）。

## MCP 与 dsh

```bash
nodepulse mcp
```

MCP server 提供节点、测速、排名、切换、导出、状态、历史工具。dsh 插件在 `plugins/dsh/`，只封装 CLI，不重复核心逻辑。

## 发布前检查

```bash
go test ./...
go test -race ./...
go vet ./...
cd desktop && flutter analyze && flutter test
VERSION=x.y.z make build desktop deb
```

发布/签名/公证流程见 [packaging.md](packaging.md)。
