# 跨平台打包

桌面和 Android 客户端使用 Flutter，Go 后端作为本机 sidecar 随应用一起分发。

## Linux

依赖：

```bash
sudo apt-get install libgtk-3-dev liblzma-dev libblkid-dev clang cmake ninja-build pkg-config
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
```

```bash
VERSION=0.3.5 make build desktop deb
VERSION=0.3.5 make appimage
```

产物：

- `dist/NodePulse-linux-amd64-0.3.5.deb`
- `dist/NodePulse-linux-amd64-0.3.5.AppImage`

安装布局：

```text
/usr/bin/nodepulse              Go CLI
/usr/bin/nodepulse-desktop      启动器 -> /opt/nodepulse/nodepulse
/opt/nodepulse/nodepulse        Flutter 桌面端
/opt/nodepulse/nodepulse-core   Go sidecar
/opt/nodepulse/data, lib        Flutter 资源
```

## Windows

在 Windows runner 上：

```powershell
$env:VERSION = "0.3.5"
.\packaging\build-windows.ps1
```

要求：Flutter、Go、NSIS（`makensis`）。脚本会：

1. `flutter build windows --release`；
2. 构建 `bin/nodepulse.exe` 并复制成 `nodepulse-core.exe`；
3. 生成便携 zip；
4. 用 `packaging/nodepulse.nsi` 生成 NSIS 安装包。

安装器会显示组件页：开始菜单快捷方式始终创建，桌面快捷方式默认勾选，用户可以取消。

安装器细节：

- `packaging/nodepulse.nsi` 必须保存为 **UTF-8 with BOM**，`makensis` 调用带 `-INPUTCHARSET UTF8`；否则 Windows runner 会按本地代码页解释脚本里的中文，组件名显示为乱码；
- 声明 `ManifestDPIAware true` 与 `ManifestDPIAwareness "PerMonitorV2,System"`，避免 Windows 在 125% / 150% 下把整个安装器当位图拉伸；
- `MUI_ICON` / `MUI_UNICON` 使用 `packaging/nodepulse.ico`（16–256 多尺寸）；
- `MUI_HEADERIMAGE_BITMAP` 使用 `packaging/nodepulse-header.bmp`（300×114，2x），由 `go run packaging/genicon.go` 生成。

产物：

- `dist/NodePulse-windows-amd64-<version>.zip`
- `dist/NodePulse-windows-amd64-<version>.exe`

Flutter Windows 不能从 Linux 交叉构建。

## Android

在 Linux 或 CI runner 上：

```bash
VERSION=0.3.5 make android
```

脚本会：

1. 用 `GOOS=android GOARCH=arm64 CGO_ENABLED=0` 构建 Go sidecar；
2. 复制为 `desktop/android/app/src/main/jniLibs/arm64-v8a/libnodepulse_core.so`；
3. 执行 `flutter build apk --release --target-platform android-arm64`；
4. 输出 `dist/NodePulse-android-arm64-<version>.apk`。

APK 使用 debug 签名，适合直接下载安装；上架或稳定增量更新前需要配置 release keystore。当前只打包 arm64-v8a，且 Android 客户端不内置 TUN/VPN，需要连接已有的 Clash/mihomo 控制口。

Android 构建目前固定 Flutter 3.47.9、Gradle 8.14.3、AGP 8.13.0、Kotlin 2.2.20，以兼容 `file_picker`；升级前先验证它与 `launch_at_startup` 的 win32 依赖冲突。

## macOS

在 macOS runner 上：

```bash
VERSION=0.3.5 bash packaging/build-macos.sh
```

脚本会构建 `desktop/build/macos/Build/Products/Release/NodePulse.app`，把 amd64/arm64 的 Go sidecar 合并为 `Contents/MacOS/nodepulse-core`，生成 zip 和 dmg。

产物：

- `dist/NodePulse-macos-universal-<version>.zip`
- `dist/NodePulse-macos-universal-<version>.dmg`

未配置 Developer ID 签名和公证；当前只做 ad-hoc 签名。

## CI

- `.github/workflows/release.yml`：Linux deb/AppImage + GoReleaser CLI 归档；
- `.github/workflows/desktop.yml`：Windows NSIS/zip、macOS zip/dmg。

## 图标和许可证

- 源图标：`packaging/nodepulse-1024.png`，生成器：`go run packaging/genicon.go`；
- Windows 图标：`packaging/nodepulse.ico`；
- Windows 安装器页眉：`packaging/nodepulse-header.bmp`（同样由 `genicon.go` 生成）；
- 每个安装包都带 `LICENSE`（GPL-3.0）和 `THIRD_PARTY_NOTICES.md`（含 FlClash 归属）。
