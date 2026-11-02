# 开发与发布

## 环境

- Go 1.24+
- `mihomo` 可执行文件（集成测试 / sidecar）
- `make`、`gcc`（交叉编译时可选）

国内网络建议设置模块代理：

```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

## 常用命令

```bash
make build     # 输出 bin/nodepulse
make test      # go test ./...
make vet       # go vet ./...
make fmt       # gofmt
make clean
```

## 本地验证

### 用测试配置

```bash
./bin/nodepulse nodes -f testdata/subscription.yaml
```

### 用真实订阅 + sidecar

```bash
./bin/nodepulse test \
  -f ~/my-subscription.yaml \
  --mihomo /path/to/mihomo \
  --rounds 5 \
  --out results.json

./bin/nodepulse rank -i results.json
./bin/nodepulse export -f ~/my-subscription.yaml -i results.json -o stable.yaml
```

### 用已有控制口

```bash
./bin/nodepulse test \
  -f ~/my-subscription.yaml \
  --api http://127.0.0.1:9090 \
  --secret your-secret \
  --rounds 10
```

## 测试

```bash
go test ./...
go test -race ./...
```

- `internal/score`：评分、百分位、排序
- `internal/subscription`：解析、去重、信息节点
- `internal/clash`：用 `httptest` 模拟控制口
- `internal/tester`：用 fake `Delayer` 验成功率
- `internal/export`：过滤与导出

## GUI

```bash
make run          # 构建并启动
make gui          # 只构建
./bin/nodepulse-gui
```

GUI 使用 Wails 生产构建标签 `production,webkit2_41`；如果手动构建，别忘了标签：

```bash
cd gui && go build -tags production,webkit2_41 -o ../bin/nodepulse-gui .
```

GUI 与 CLI 共用核心；绑定代码在 `gui/app.go`，前端在 `gui/frontend/dist`。首次启动会自动检测控制口、mihomo 和订阅文件。

## MCP

```bash
./bin/nodepulse mcp
```

用 JSON-RPC stdio 与 agent 客户端通信。工具列表见 `internal/mcp/tools.go`。

## 本地 Web UI

```bash
./bin/nodepulse serve --addr 127.0.0.1:8099
```

浏览器打开 `http://127.0.0.1:8099`，可以刷新状态、测试、切换和查看历史。

## dsh 插件

`plugins/dsh` 是 DeepSeek Harness 的薄封装，调用 `nodepulse` CLI：

```bash
cd plugins/dsh && npm pack
dsh plugin --profile web add ./hanphone-dsh-nodepulse-0.1.0.tgz
```

## 打包

```bash
make deb        # 需要 nfpm
make appimage   # 需要 appimagetool
```

`.deb` 内容见 `packaging/nfpm.yaml`；AppImage 脚本见 `packaging/build-appimage.sh`。

## 发布（Phase 3）

计划使用 GoReleaser + nfpm + appimagetool：

```bash
goreleaser release --clean
```

产物：

- `nodepulse_<version>_linux_amd64.deb`
- `nodepulse_<version>_linux_arm64.deb`
- `NodePulse-<version>-x86_64.AppImage`
- `nodepulse_<version>_linux_amd64.tar.gz`
- `checksums.txt`

GitHub Actions 在 tag 上执行，发布到 GitHub Releases。

## 代码约定

- 核心包不依赖 CLI/GUI；
- 错误带上下文（`fmt.Errorf("...: %w", err)`）；
- 日志不打印订阅 URL、UUID、密码；
- 新增客户端适配器实现 `Delayer`（和可选的 `Switcher`）；
- 新功能先补文档，再补测试。
