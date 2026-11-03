# 开发

## 环境

- CLI/核心：Go 1.24+
- GUI：Go 1.26+，GTK3、WebKit2GTK 4.1、`pkg-config`
- `mihomo`：sidecar 测速/下载探测
- Node.js：`make ui-check` 的 JS 语法检查
- `nfpm`：`.deb`；`appimagetool`：AppImage；Wails CLI：Windows 构建

Linux 构建依赖：

```bash
sudo apt-get install libgtk-3-dev libwebkit2gtk-4.1-dev pkg-config
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
```

## 常用命令

```bash
make build      # bin/nodepulse
make gui        # bin/nodepulse-gui
make run        # 构建并启动 GUI
make web        # 本地 Web UI
make test       # 根模块测试
make race       # race 测试
make vet        # go vet
make fmt        # gofmt
make ui-check   # JS 语法 + 前端资源测试
make icons      # 重新生成图标 sprite
make deb        # .deb
make appimage   # AppImage
make clean
```

GUI 是独立 module，测试/构建需要标签：

```bash
cd gui
go test -tags production,webkit2_41 ./...
go build -tags production,webkit2_41 ./...
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

## 前端资源

GUI/Web 共用 `internal/uiassets/dist`，使用 Preact + htm，无 Node 构建。

```bash
make icons
make ui-check
```

新增前端依赖时必须：

1. 本地固定版本，提交 vendor 文件；
2. 更新 `THIRD_PARTY_NOTICES.md`；
3. 更新 `packaging/nfpm.yaml` 与 `packaging/build-appimage.sh` 的许可证复制。

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
make ui-check
go test ./...
go test -race ./...
go vet ./...
cd gui && go test -tags production,webkit2_41 ./...
VERSION=x.y.z make build gui deb
timeout --kill-after=2 6 ./bin/nodepulse-gui
```

发布/签名/公证流程见 [packaging.md](packaging.md)。
