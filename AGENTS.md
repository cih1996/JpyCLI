# JPY CLI (Go Version)

JPY 中间件管理命令行工具（Go 语言版）。
本工具复用了 TypeScript SDK 的协议定义，实现了与 JPY 中间件的高效通信。

## 目录结构

packages/jpy-cli-go/
├── cmd/
│   └── jpy-cli/
│       └── main.go           # 应用入口 (Entry Point)
├── internal/
│   └── cmd/                  # Cobra 命令定义 (Command Definitions)
│       ├── admin/            # 管理员相关命令 (generate, list, root_pwd)
│       │   └── middleware/   # 管理员中间件操作 (update-cluster)
│       ├── config/           # 配置管理命令
│       ├── log/              # 日志管理命令
│       ├── middleware/       # 中间件业务命令
│       │   ├── admin/        # 管理员子命令 (auto-auth, update-cluster)
│       │   ├── auth/         # 认证相关 (login, import, list)
│       │   ├── device/       # 设备管理 (list, control, log, status)
│       │   ├── middleware.go # 中间件命令入口
│       │   ├── relogin.go    # 重连命令
│       │   ├── remove.go     # 删除命令
│       │   ├── server_list.go# 服务器列表命令
│       │   └── ssh.go        # SSH 命令
│       ├── tools/            # 工具箱命令
│       │   ├── middleware/   # 中间件工具 (create)
│       │   └── tools.go      # 工具箱入口
│       └── root.go           # 根命令定义
├── pkg/
│   ├── admin-middleware/     # 管理后台中间件逻辑
│   ├── client/               # 基础网络客户端
│   │   ├── http/             # HTTP 客户端封装
│   │   └── ws/               # WebSocket 客户端封装
│   ├── config/               # 配置加载与存储
│   ├── logger/               # 日志工具
│   ├── middleware/           # 中间件核心业务逻辑
│   │   ├── connector/        # 连接服务 (Connection Service)
│   │   ├── device/           # 设备领域层
│   │   │   ├── api/          # 设备 API 交互
│   │   │   ├── controller/   # 设备控制逻辑 (Reboot, ADB, USB)
│   │   │   ├── fetcher/      # 设备信息抓取
│   │   │   ├── selector/     # 设备筛选器 (含 TUI 表格)
│   │   │   └── terminal/     # 终端连接会话管理
│   │   ├── model/            # 数据模型定义
│   │   └── protocol/         # 通信协议定义
│   └── tui/                  # 通用 TUI 组件 (Progress, Selection)
├── Makefile                  # 构建脚本
├── go.mod                    # 依赖定义
└── jpy-config.yaml           # 开发配置文件

## 打包命令

### 1. 开发编译
编译当前平台的可执行文件到 `bin/` 目录：
```bash
make build
```

### 2. 交叉编译 (发布)
同时编译 Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64) 版本到 `dist/` 目录，并自动压缩体积：
```bash
make dist
```

### 3. 手动编译
如果未安装 `make` 工具，可使用 Go 命令直接编译：
```bash
# 编译当前平台
go build -o jpy cmd/jpy-cli/main.go

# 编译 Linux amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o jpy-linux cmd/jpy-cli/main.go
```

## 开发规范
1. **目录一致性**：必须严格遵循当前目录结构进行扩展。业务逻辑应放入 `pkg/` 下的对应领域包中，`internal/cmd/` 仅负责参数解析和流程串联。
2. **复用原则**：新增功能前请检查现有模块（如 `pkg/client`, `pkg/tui`），优先复用已有逻辑。
3. **输出规范**：列表输出必须支持排序。
   - **服务器列表**：优先按地址（IP/域名）排序。
   - **设备列表**：在服务器排序的基础上，按 Seat（盘位）顺序排序。
