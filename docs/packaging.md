# 跨平台打包

桌面端使用 Flutter，Go 后端作为本机 sidecar 随 Flutter 应用一起分发。

## Linux

依赖：

```bash
sudo apt-get install libgtk-3-dev liblzma-dev libblkid-dev clang cmake ninja-build pkg-config
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
```

```bash
VERSION=0.3.0 make build desktop deb
VERSION=0.3.0 make appimage
```

产物：

- `dist/NodePulse-linux-amd64-0.3.0.deb`
- `dist/NodePulse-linux-amd64-0.3.0.AppImage`

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
$env:VERSION = "0.3.0"
.\packaging\build-windows.ps1
```

要求：Flutter、Go、NSIS（`makensis`）。脚本会：

1. `flutter build windows --release`；
2. 构建 `bin/nodepulse.exe` 并复制成 `nodepulse-core.exe`；
3. 生成便携 zip；
4. 用 `packaging/nodepulse.nsi` 生成 NSIS 安装包。

产物：

- `dist/NodePulse-windows-amd64-<version>.zip`
- `dist/NodePulse-windows-amd64-<version>.exe`

Flutter Windows 不能从 Linux 交叉构建。

## macOS

在 macOS runner 上：

```bash
VERSION=0.3.0 bash packaging/build-macos.sh
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
- 每个安装包都带 `LICENSE`（GPL-3.0）和 `THIRD_PARTY_NOTICES.md`（含 FlClash 归属）。
