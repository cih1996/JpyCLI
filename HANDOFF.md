# jpy_local_cli 项目交接文档

更新时间：2026-05-13  
当前负责会话：116 / CLI工具开发

本文档用于把当前项目结构、近期改动、已完成事项和后续接手重点交代清楚。

---

## 1. 项目定位

`jpy_local_cli` 是 JPY 中间件管理命令行工具，Go 语言实现，基于 Cobra。

核心定位：

- 面向 AI、脚本、自动化系统、上层 UI 调用。
- 零配置、无状态。
- 命令一行完成，不做交互式 TUI。
- 默认 `plain` 输出，支持 `-o json`。
- 可通过 `jpy server` 提供 HTTP 远程控制能力。

项目路径：

```text
/Users/cih1996/work/jpy-zyd/jpy_local_cli
```

正确开发调试入口：

```bash
go run ./cmd/jpy-cli/main.go ...
```

不要使用：

```bash
go run .
```

原因：根目录存在其它调试 main，可能会连接 `localhost:8000/ws` 并失败。

---

## 2. 当前核心目录结构

```text
cmd/jpy-cli/main.go             CLI 入口
internal/cmd/root.go            根命令、全局 --remote 拦截、命令注册
internal/cmd/remote.go          全局 --remote 客户端转发逻辑
internal/cmd/server.go          HTTP 代理服务、/exec、/exec/async、/shell、文件传输、任务管理
internal/cmd/shell.go           远程系统 shell 命令
internal/cmd/version.go         本地/远程版本查询
internal/cmd/update.go          远程更新 jpy CLI 程序

internal/cmd/device/            中间件设备管理命令
internal/cmd/device/root.go     device 父命令、-s/-u/-p/-o 公共参数
internal/cmd/device/middleware.go  新增：中间件维护命令、固件升级
internal/cmd/device/rom.go      新增：中间件 ROM 包管理和接口刷机

internal/cmd/com/               COM 串口控制
internal/cmd/flash/             COM 口批量刷机
internal/cmd/file/              文件上传/下载命令
internal/cmd/stress/            压力测试命令
pkg/auth/                       无状态凭证解析和登录
pkg/client/                     HTTP / WebSocket 客户端
pkg/middleware/                 中间件 SDK/协议/模型
pkg/comport/                    COM 串口协议层
sdk/                            Go SDK 客户端
```

---

## 3. 近期新增功能

### 3.1 新增 `device middleware` 命令组

注册位置：

```go
// internal/cmd/device/root.go
cmd.AddCommand(NewMiddlewareCmd())
```

新增文件：

```text
internal/cmd/device/middleware.go
internal/cmd/device/rom.go
```

命令树：

```bash
jpy device middleware upgrade --file <firmware-file> -s <server> -u <user> -p <pass> [--required=true|false] [-o plain|json]

jpy device middleware rom upload --file <rom-file> -s <server> -u <user> -p <pass> [-o plain|json]
jpy device middleware rom list -s <server> -u <user> -p <pass> [-o plain|json]
jpy device middleware rom flash --seat <seat> --sn <sn> --image <image> -s <server> -u <user> -p <pass> [--mode 2] [-o plain|json]
jpy device middleware rom status -s <server> -u <user> -p <pass> [--seat <seat>] [--sn <sn>] [-o plain|json]
jpy device middleware rom detail --seat <seat> --session <session> -s <server> -u <user> -p <pass> [-o plain|json]
```

说明：这些命令继承 `device` 父命令的公共参数：

```text
-s, --server
-u, --username
-p, --password
-o, --output plain|json
```

---

## 4. 中间件固件升级实现说明

命令：

```bash
jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456 -o json
```

实现文件：

```text
internal/cmd/device/middleware.go
```

内部流程：

1. 通过 `resolveCredentials()` 复用现有登录/凭证逻辑。
2. 打开本地固件文件。
3. 上传到中间件接口：

```text
POST /sys/upload
```

4. 从响应 `data` 中解析 `package_id`。
5. 调用升级接口：

```text
POST /sys/update?required=<true|false>&id=<package_id>
```

6. 输出 `plain` 或 `json`。

关键参数：

| 参数 | 说明 |
|------|------|
| `--file` | 本地固件路径，必填 |
| `--required` | 是否强制升级，默认 `true` |
| `-o json` | 推荐 UI/脚本使用 |

当前状态：已实现 CLI 命令和输出模式；没有做升级后轮询，因为桌面端链路只确认到提交升级动作。

---

## 5. 中间件 ROM 接口刷机实现说明

实现文件：

```text
internal/cmd/device/rom.go
```

这条链路不是原有 COM 口刷机 `flash run`，而是通过中间件 HTTP + Guard WebSocket 接口完成。

### 5.1 上传 ROM 包

```bash
jpy device middleware rom upload --file ./rom.zip -s 172.25.0.251 -u admin -p 123456 -o json
```

内部接口：

```text
POST /box/upload
```

### 5.2 查看 ROM 包列表

```bash
jpy device middleware rom list -s 172.25.0.251 -u admin -p 123456 -o json
```

内部链路：

```text
WebSocket /box/guard
SendRequest(113)
```

返回的 `name` 通常作为后续 `--image`。

### 5.3 发起刷机

```bash
jpy device middleware rom flash --seat 3 --sn ABC123 --image 1767856234 -s 172.25.0.251 -u admin -p 123456 -o json
```

内部链路：

```text
WebSocket /box/guard
SendRequest(119)
```

请求字段：

```json
{
  "seat": 3,
  "sn": "ABC123",
  "image": "1767856234",
  "mode": 2
}
```

### 5.4 查询刷机状态

```bash
jpy device middleware rom status -s 172.25.0.251 -u admin -p 123456 -o json
```

内部链路：

```text
WebSocket /box/guard
SendRequest(117)
```

支持过滤：

```bash
--seat 3
--sn ABC123
```

当前 `status` 是中间件原始状态码，CLI 没有做二次翻译，方便 UI 按真实值判断。

### 5.5 查询刷机详情日志

```bash
jpy device middleware rom detail --seat 3 --session session-id -s 172.25.0.251 -u admin -p 123456 -o json
```

内部接口：

```text
GET /box/detail?id=<seat>&session=<session>
```

如果响应是 `text/event-stream`，当前代码会提取 `data:` 行并合并成普通文本。

当前状态：基础命令已完成。自动轮询、关键字判断、最终成功/失败归因还没封装成一条完整命令。

---

## 6. 远程调用能力

### 6.1 CLI `--remote` 调用

被控端启动：

```bash
jpy server --port 9090
```

主控端调用：

```bash
jpy --remote 10.0.0.5:9090 device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456 -o json
```

远程通讯通道是 HTTP，不是 WebSocket。

支持情况：

| 命令类型 | 是否支持远程 | 说明 |
|----------|--------------|------|
| `device` | 支持 | 全局 `jpy --remote <host:port> device ...` |
| `device middleware` | 支持 | 本次新增升级和 ROM 刷机命令都支持 |
| `com` | 支持 | COM 口在被控端时使用 |
| `flash` | 支持 | 脚本、fastboot、COM 都在被控端时使用 |
| `stress` | 支持 | 长任务建议异步 |
| `shell` | 支持 | 专用写法：`jpy shell --remote <host:port> ...` |
| `file` | 支持 | 命令自带 `--remote` |
| `update` | 支持 | 命令自带 `--remote` |
| `version` | 支持 | 命令自带 `--remote` |
| `server` | 不支持 | 代码禁止远程执行，防递归 |

### 6.2 纯 HTTP 远程控制

新增文档：

```text
HTTP_REMOTE_CONTROL.md
```

用于不走 `jpy --remote`，直接通过 HTTP 请求控制被控端。

典型请求：

```bash
curl -X POST http://被控端IP:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","middleware","rom","status","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```

注意：

- `/exec` 的 `args` 不包含 `jpy` 本身。
- `/exec` 和 `/exec/async` 禁止包含 `server` 或 `--remote`。
- 所有 `--file` 参数都是被控端本机路径。
- 主控端文件要先通过 `/file/upload` 上传到被控端。

---

## 7. 本轮文档整理情况

新增文档：

```text
CLI_ZH_README.md              中文完整使用手册
CLI_README.md                 英文完整使用手册
HTTP_REMOTE_CONTROL.md        纯 HTTP 远程控制文档
HANDOFF.md                    当前交接文档
```

更新文档：

```text
README.md                     增加新增命令、远程模式、HTTP 文档索引
常用命令.md                   增加中间件升级、ROM 刷机、远程调用常用命令
```

说明：项目规则里写的是 `常用CLI.md`，当前仓库实际存在的是：

```text
常用命令.md
```

本次按实际文件更新。

---

## 8. 当前工作区状态

截至本交接文档生成时，当前改动还未提交。

已跟踪文件变更：

```text
README.md
internal/cmd/device/root.go
常用命令.md
```

新增未跟踪文件：

```text
CLI_README.md
CLI_ZH_README.md
HTTP_REMOTE_CONTROL.md
HANDOFF.md
internal/cmd/device/middleware.go
internal/cmd/device/rom.go
```

注意：`git diff --stat` 默认不显示未跟踪文件，因此看统计时要结合 `git status --short`。

---

## 9. 已做验证

已执行过帮助命令验证，新增命令可被 Cobra 正常识别：

```bash
go run ./cmd/jpy-cli/main.go device middleware --help
go run ./cmd/jpy-cli/main.go device middleware upgrade --help
go run ./cmd/jpy-cli/main.go device middleware rom --help
go run ./cmd/jpy-cli/main.go device middleware rom upload --help
go run ./cmd/jpy-cli/main.go device middleware rom list --help
go run ./cmd/jpy-cli/main.go device middleware rom flash --help
go run ./cmd/jpy-cli/main.go device middleware rom status --help
go run ./cmd/jpy-cli/main.go device middleware rom detail --help
go run ./cmd/jpy-cli/main.go server --help
go run ./cmd/jpy-cli/main.go shell --help
go run ./cmd/jpy-cli/main.go version --help
```

当前未重新执行完整 `make dist`。项目规则要求最终交付前跑：

```bash
make dist
```

如果只做快速开发验证，可先跑：

```bash
make build
```

---

## 10. 后续接手建议

优先级从高到低：

1. **跑构建验证**  
   执行 `make build`，最终交付前执行 `make dist`。

2. **确认真实中间件环境接口返回**  
   尤其是：
   - `/sys/upload` 返回的 `data` 是否始终是数字 package id。
   - `/sys/update` 是否只代表提交升级，还是可判断升级完成。
   - Guard WS 的 `113/117/119` 返回字段是否和当前解析完全一致。

3. **补齐 ROM 刷机一键流程**  
   目前是拆分命令，后续可以封装：
   - 上传 ROM
   - 获取 ROM list
   - 发起 flash
   - 周期查询 status
   - 获取 detail
   - 根据状态码、last_error、详情关键字判断最终成功/失败

4. **增强错误 JSON 输出**  
   当前失败多数直接返回 error，由 Cobra 打印错误。若 UI 需要稳定解析失败结果，可统一封装错误 JSON。

5. **考虑超时参数**  
   ROM 上传已设置 2 小时 HTTP 超时；固件升级上传目前使用默认 client，没有单独暴露 timeout。

6. **更新项目规则里的文档文件名**  
   `.claude/CLAUDE.md` 写的是 `常用CLI.md`，仓库实际是 `常用命令.md`。后续可统一规则或补一个同名文档。

---

## 11. 快速上手命令

本地开发调试：

```bash
go run ./cmd/jpy-cli/main.go device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456 -o json
```

启动被控端 HTTP 服务：

```bash
jpy server --port 9090
```

CLI 远程调用：

```bash
jpy --remote 10.0.0.5:9090 device middleware rom status -s 172.25.0.251 -u admin -p 123456 -o json
```

HTTP 远程调用：

```bash
curl -X POST http://10.0.0.5:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","middleware","rom","status","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```
