# 跨平台打包

NodePulse 由 CLI 和 Wails 桌面 GUI 组成。CLI 可以纯 Go 交叉编译；GUI 依赖各平台原生 WebView，因此以对应平台的 CI 构建为准，不在 Linux 上伪造 Windows/macOS 产物。

## Linux

本地构建：

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
VERSION=0.3.0 make build gui deb
VERSION=0.3.0 make appimage
```

产物：

- `dist/nodepulse_<version>_amd64.deb`
- `dist/NodePulse-<version>-x86_64.AppImage`

依赖：`libgtk-3-dev`、`libwebkit2gtk-4.1-dev`、`pkg-config`、`nfpm`。GUI 构建标签固定为 `production,webkit2_41`。

## Windows

CI 在 `windows-latest` 上安装 Wails 后运行 `packaging/build-windows.ps1`：

```powershell
Push-Location gui
wails build -platform windows/amd64 -s -skipbindings -m -tags production `
  -ldflags "-s -w -H windowsgui -X github.com/HanphoneJan/nodepulse/internal/version.Version=$env:VERSION"
Copy-Item build\bin\nodepulse.exe ../dist/nodepulse.exe
Pop-Location
```

Wails 会用 `gui/build/appicon.png` 生成 `icon.ico` 和清单，因此 exe 带图标。没有 Wails 时脚本回退到纯 Go 构建，但不嵌入图标。

产物是 zip 内的单个 `nodepulse.exe`。当前不做 MSI/安装器；代码签名需要发布者证书，未签名时 SmartScreen 会提示。

本地辅助脚本：`packaging/build-windows.ps1`。

## macOS

CI 在 `macos-latest` 上运行 `packaging/build-macos.sh`：

- 分别用 `GOARCH=amd64` 和 `GOARCH=arm64` 构建；
- 用 `lipo` 合并为 universal 二进制；
- 生成 `NodePulse.app/Contents/Info.plist`；
- 使用 ad-hoc 签名后 `ditto` 打包为 zip。

产物是 zip 内的 `.app`。未签名/未公证的 app 在 Gatekeeper 下仍可能需要用户手动放行；正式发布应配置 Apple Developer ID 和 notarization。

本地辅助脚本：`packaging/build-macos.sh`。

## 图标与元数据

- 图标源：`packaging/nodepulse-1024.png`，由 `go run packaging/genicon.go` 生成；
- Linux 使用 `packaging/nodepulse.svg`；
- Windows 使用 Wails 从 `gui/build/appicon.png` 生成 exe 图标，并在 zip 内附带 `packaging/nodepulse.ico`；
- macOS 脚本用 `sips` + `iconutil` 从 1024 PNG 生成 `.icns` 并写入 `Info.plist`；
- 托盘图标为 `gui/tray-icon.png`，与主图标同一套形状和配色；
- 版本号统一由 `internal/version.Version` 注入。

## CI

`.github/workflows/desktop.yml` 在 tag 发布时构建 Windows/macOS：

- Windows：zip；
- macOS：zip。

Linux 的 `.deb` / AppImage 继续由 `.github/workflows/release.yml` 构建。Windows/macOS 产物只作为构建产物上传，不代表已完成签名或商店分发。
