# 打包

NodePulse 由 CLI（纯 Go）和 Wails 桌面 GUI 组成。GUI 依赖各平台原生 WebView，因此 Windows/macOS 由对应平台的 CI 构建。

## Linux

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
VERSION=0.3.0 make build gui deb
VERSION=0.3.0 make appimage
```

产物：

- `dist/NodePulse-linux-amd64-<version>.deb`
- `dist/NodePulse-linux-amd64-<version>.AppImage`

依赖：`libgtk-3-dev`、`libwebkit2gtk-4.1-dev`、`pkg-config`、`nfpm`。GUI 标签固定为 `production,webkit2_41`。deb 推荐 `libnotify-bin` 作为 Linux 通知回退。

AppImage 使用 `AppImage/appimagetool` 与 `type2-runtime`。本地构建时把 `APPIMAGE_RUNTIME_FILE` 指向下载好的 `runtime-x86_64`，或让 appimagetool 自行下载运行时。

## Windows

`.github/workflows/desktop.yml` 在 Windows runner 安装 Wails 和 NSIS 后运行：

```powershell
Push-Location gui
wails build -platform windows/amd64 -s -skipbindings -m -tags production -nsis `
  -ldflags "-s -w -H windowsgui -X github.com/HanphoneJan/nodepulse/internal/version.Version=$env:VERSION"
Copy-Item build\bin\nodepulse.exe ../dist/nodepulse.exe
Copy-Item build\bin\nodepulse-amd64-installer.exe ..\dist\NodePulse-windows-amd64-$env:VERSION.exe
Pop-Location
```

Wails 用 `gui/build/appicon.png` 生成 exe 图标和清单，`-nsis` 生成带开始菜单项、桌面快捷方式和卸载项的安装程序。没有 Wails 时脚本回退到纯 Go 构建，但 exe 不嵌入图标、也没有安装包。产物是 `dist/NodePulse-windows-amd64-<version>.exe`（NSIS 安装包）和 `dist/NodePulse-windows-amd64-<version>.zip`（便携版，zip 内为 `nodepulse.exe`）；不做 MSI，代码签名未配置。

## macOS

`packaging/build-macos.sh` 在 macOS 上：

- 分别构建 `amd64` / `arm64`，用 `lipo` 合并 universal 二进制；
- 生成 `Info.plist` 和 `.icns`；
- ad-hoc 签名后 `ditto` 打包 zip。

产物是 `dist/NodePulse-macos-universal-<version>.dmg`（拖入 Applications 安装）和 `dist/NodePulse-macos-universal-<version>.zip`（zip 内为 `.app`）。未签名/未公证时 Gatekeeper 可能要求手动放行；正式发布需要 Apple Developer ID + notarization。

## 图标

- 源图标：`packaging/nodepulse-1024.png`，生成器：`go run packaging/genicon.go`；
- Linux：`packaging/nodepulse.svg`；
- Windows：Wails 生成 exe 图标，zip 附带 `packaging/nodepulse.ico`；
- macOS：`sips` + `iconutil` 生成 `.icns`；
- 托盘：`gui/tray-icon.png`。

版本统一通过 `internal/version.Version` 注入。

## 发布产物命名

GitHub Release 的附件统一使用 `应用名称-平台-版本号`：

- `NodePulse-linux-amd64-<version>.tar.gz` / `NodePulse-linux-arm64-<version>.tar.gz`（CLI，GoReleaser）
- `NodePulse-linux-amd64-<version>.deb`
- `NodePulse-linux-amd64-<version>.AppImage`
- `NodePulse-windows-amd64-<version>.exe`（NSIS 安装包）
- `NodePulse-windows-amd64-<version>.zip`（便携版）
- `NodePulse-macos-universal-<version>.dmg`
- `NodePulse-macos-universal-<version>.zip`

## 发布状态

- Linux `.deb` / AppImage / CLI tar.gz：本地与 release workflow 构建。
- Windows `.exe`（NSIS）/ zip：desktop workflow 构建；待首次实跑与签名。
- macOS `.dmg` / zip：desktop workflow 构建；待首次实跑、签名与公证。
