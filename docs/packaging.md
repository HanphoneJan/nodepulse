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

CI 在 `windows-latest` 上运行 `packaging/build-windows.ps1`：

```powershell
Push-Location gui
go build -trimpath -tags "production,desktop" `
  -ldflags "-s -w -H windowsgui -X github.com/HanphoneJan/nodepulse/internal/version.Version=$env:VERSION" `
  -o ../dist/nodepulse.exe .
Pop-Location
```

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

- Windows/macOS 使用与托盘一致的 `gui/tray-icon.png` 作为当前图标源；
- 正式发布前应补充 `.ico` / `.icns` 和多尺寸图标；
- 版本号统一由 `internal/version.Version` 注入。

## CI

`.github/workflows/desktop.yml` 在 tag 发布时构建 Windows/macOS：

- Windows：zip；
- macOS：zip。

Linux 的 `.deb` / AppImage 继续由 `.github/workflows/release.yml` 构建。Windows/macOS 产物只作为构建产物上传，不代表已完成签名或商店分发。
