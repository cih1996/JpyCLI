# JPY CLI

JPY 中间件管理命令行工具，面向 AI/脚本设计，零配置，无状态。

## 安装

从 [Releases](https://github.com/cih1996/JpyCLI/releases) 下载对应平台的二进制文件：

| 平台 | 文件 |
|------|------|
| Linux x64 | jpy-linux-amd64.tar.gz |
| Linux ARM64 | jpy-linux-arm64.tar.gz |
| macOS x64 | jpy-darwin-amd64.tar.gz |
| macOS ARM64 | jpy-darwin-arm64.tar.gz |
| Windows x64 | jpy-windows-amd64.zip |

解压后将 `jpy` 放入 PATH 即可使用。

## 文档索引

- [`CLI_ZH_README.md`](./CLI_ZH_README.md)：中文完整使用手册
- [`CLI_README.md`](./CLI_README.md)：英文完整使用手册
- [`HTTP_REMOTE_CONTROL.md`](./HTTP_REMOTE_CONTROL.md)：纯 HTTP 远程控制文档
- [`HANDOFF.md`](./HANDOFF.md)：项目交接文档

## 功能板块

### 1. Device - 中间件设备管理

连接 JPY 中间件服务器，管理 Android 设备。

```bash
# 列出所有设备
jpy device list -s 192.168.1.1 -u admin -p 123456 -o json

# 在指定机位执行命令
jpy device shell "ls /sdcard" -s 192.168.1.1 -u admin -p 123456 --seat 3

# 重启设备
jpy device reboot -s 192.168.1.1 -u admin -p 123456 --seat 3

# 切换 USB 模式
jpy device usb -s 192.168.1.1 -u admin -p 123456 --mode host --seat 3

# 开关 ADB
jpy device adb -s 192.168.1.1 -u admin -p 123456 --set on --seat 3

# 查看服务器状态
jpy device status -s 192.168.1.1 -u admin -p 123456 -o json
```

### 2. Middleware - 中间件固件升级

上传本地固件到中间件，并立即调用升级接口执行更新。

```bash
# 上传固件并执行升级
jpy device middleware upgrade --file ./firmware.bin -s 192.168.1.1 -u admin -p 123456

# JSON 输出，方便 UI 或脚本接入
jpy device middleware upgrade --file ./firmware.bin -s 192.168.1.1 -u admin -p 123456 -o json

# 非强制升级
jpy device middleware upgrade --file ./firmware.bin -s 192.168.1.1 -u admin -p 123456 --required=false
```

### 3. Middleware ROM - 中间件接口刷机基础命令

通过中间件 HTTP + Guard WebSocket 接口管理 ROM 包并发起刷机。

```bash
# 上传 ROM 包
jpy device middleware rom upload --file ./rom.zip -s 192.168.1.1 -u admin -p 123456

# 查看 ROM 包列表
jpy device middleware rom list -s 192.168.1.1 -u admin -p 123456

# 发起刷机（image 使用 rom list 返回的 name）
jpy device middleware rom flash --seat 3 --sn ABC123 --image 1767856234 -s 192.168.1.1 -u admin -p 123456

# 查询刷机状态
jpy device middleware rom status -s 192.168.1.1 -u admin -p 123456 -o json

# 查询刷机详情日志
jpy device middleware rom detail --seat 3 --session session-id -s 192.168.1.1 -u admin -p 123456
```

### 4. COM - 串口硬件控制

操作 USB HUB 控制板（20路通道），通过 COM 串口通信。

```bash
# 列出可用串口
jpy com list -o json

# 查看设备通道状态
jpy com devices --port COM3 -o json

# 设置通道模式（HUB/OTG）
jpy com set-mode --port COM3 --mode hub --channel 5
jpy com set-mode --port COM3 --mode otg --channel 2-20  # 范围
jpy com set-mode --port COM3 --mode hub --channel 1,2,3  # 列表

# 重启通道
jpy com restart --port COM3 --channel 3
jpy com restart --port COM3 --channel 2-20  # 范围
```

### 3. Shell - 远程系统命令

在远端机器执行系统 shell 命令，支持同步和异步模式。

```bash
# 同步执行
jpy shell --remote 192.168.1.100:9090 -c "dir C:\Users"

# 异步执行（长任务）
jpy shell --remote 192.168.1.100:9090 -c "fastboot flash system system.img" --async --timeout 900

# 查询任务状态
jpy shell --remote 192.168.1.100:9090 --task <task_id>

# 列出所有任务
jpy shell --remote 192.168.1.100:9090 --tasks

# 终止任务
jpy shell --remote 192.168.1.100:9090 --kill <task_id>
```

### 5. Flash - 批量刷机

集成 device + com 操作，实现批量刷机自动化。

```bash
# 刷 COM3 通道1（IP: 172.25.0.11）
jpy flash run --com COM3 --ch 1 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y

# 刷 COM3 的 1-10 通道（IP: 172.25.0.11-20）
jpy flash run --com COM3 --ch 1-10 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd"

# 远程刷机（COM口在远程机器上）
jpy --remote 192.168.1.100:9090 flash run --com COM3 --ch 1 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y

# 模拟运行（查看 IP 映射）
jpy flash run --com COM3 --ch 1-5 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" --dry
```

**路径格式：** `--script` 支持 Unix 风格路径（`C:/path/to/002.cmd`），推荐使用以避免转义问题

**IP 计算规则：** `--ip-start` 指定通道1的起始IP，后续通道自动递增

**工作流程：**
1. 检查设备状态
2. 发送 reboot bootloader
3. 切换 COM 通道为 HUB 模式
4. 等待 fastboot 设备出现
5. 执行刷机脚本（传入设备序列号）
6. 切换回 OTG 模式

### 6. File - 远程文件传输

支持大文件传输（最大 5GB），流式传输不占内存。

```bash
# 上传本地文件到远程
jpy file push ./rom.zip --remote 192.168.1.100:9090 --dest D:\flash\rom.zip

# 让远程从 URL 下载文件
jpy file pull "https://example.com/rom.zip" --remote 192.168.1.100:9090 --dest D:\flash\rom.zip

# 大文件设置更长超时
jpy file push ./large.zip --remote 192.168.1.100:9090 --timeout 3600
```

### 7. Update - 远程更新

更新远程 jpy CLI 程序，支持分片上传（适合 FRPC 隧道）。

```bash
# 从本地文件更新远程（分片上传，1MB 分片）
jpy update ./jpy-windows-amd64.exe --remote 192.168.1.100:9090

# 从 URL 更新远程（远程直接下载）
jpy update https://example.com/jpy.exe --remote 192.168.1.100:9090
```

### 8. Stress - 压力测试

用户端改机压力测试，一行命令执行，无交互式 TUI。

```bash
# 测试所有设备，单次
jpy stress user -s wss://home.accjs.cn/ws -k YOUR_SECRET_KEY -c config.json

# 测试指定设备，循环3次，间隔5分钟
jpy stress user -k YOUR_SECRET_KEY -c config.json --device 123,456,789 --loop 3 --interval 5m

# 无限循环测试
jpy stress user -k YOUR_SECRET_KEY -c config.json --loop 0 --interval 3m

# 调试模式：遇到失败立即停止（配合 --loop 0 保留现场）
jpy stress user -k YOUR_SECRET_KEY -c config.json --loop 0 --debug

# 自定义超时和日志目录
jpy stress user -k YOUR_SECRET_KEY -c config.json --timeout 15m --log-dir /var/log/stress
```

**参数说明：**
- `-s, --server`: WebSocket 服务地址（默认 wss://home.accjs.cn/ws）
- `-k, --key`: 登录密钥（必填）
- `-c, --config`: 改机配置文件路径（必填，JSON 格式）
- `--device`: 指定设备 ID 列表（逗号分隔），不指定则测试所有设备
- `--loop`: 循环次数（0=无限循环，默认 1）
- `--interval`: 循环间隔时间（默认 3m）
- `--timeout`: 单轮超时时间（默认 10m）
- `--log-dir`: 日志目录（默认 ~/.jpy/logs/stress）
- `--debug`: 调试模式，遇到失败立即停止（配合 --loop 0 保留现场）

**日志文件：** 独立记录到 `~/.jpy/logs/stress/stress_user_YYYYMMDD_HHMMSS.log`，不受 SDK 内部日志干扰。

## 远程模式

结论：远程调用走 **HTTP**，不是 WS。被控端先启动 HTTP 服务：

```bash
jpy server --port 9090
```

启动后进程会一直挂起监听，默认端口 `9090`。

支持情况：

| 类型 | 是否支持远程 | 写法 |
|------|--------------|------|
| device / device middleware / com / flash / stress | 支持 | `jpy --remote <host:port> <命令>` |
| shell | 支持 | `jpy shell --remote <host:port> ...` |
| file / update / version | 支持 | 使用命令自己的 `--remote` 参数 |
| server | 不支持 | 防止递归启动服务 |

常用示例：

```bash
jpy --remote 192.168.1.100:9090 device list -s 192.168.1.1 -u admin -p 123456
jpy --remote 192.168.1.100:9090 com list
jpy --remote 192.168.1.100:9090 device middleware upgrade --file ./firmware.bin -s 192.168.1.1 -u admin -p 123456 -o json
jpy --remote 192.168.1.100:9090 flash run --com COM3 --ch 1-20 ... -y --async --async-timeout 0
jpy shell --remote 192.168.1.100:9090 --task <task_id>
```

**超时参数：**
- `--timeout N`：同步模式 HTTP 等待超时（秒），0=无限等待，默认 120
- `--async-timeout N`：异步模式任务超时（秒），0=无限，默认 600

## 自检模式（双击运行）

无参数运行 `jpy` 进入自检模式，自动完成以下操作：

```bash
jpy
# 输出：
# ========================================
#        JPY CLI 自检模式
# ========================================
# [1] FRPC 程序路径: ~/.jpy/frpc/frpc
#     状态: 已安装
# [2] FRPC 配置文件: ~/.jpy/frpc/frpc.ini
#     状态: 已配置
# [3] FRPC 运行状态: 运行中
# [4] JPY Server 状态: 运行中 (端口 9090)
# [5] 开机自启状态: 已启用
```

**自动行为：**
1. 检测 JPY Server 是否运行，未运行则自动后台启动
2. 检测 FRPC 是否运行，未运行则自动后台启动
3. 检测开机自启是否启用，未启用则自动注册

**开机自启实现：**
- Windows: 计划任务 (schtasks)，用户登录时自动运行
- macOS: LaunchAgent，开机自动运行
- Linux: systemd user service，开机自动运行

首次运行会引导配置 FRPC（服务器地址、端口、密钥、远程映射端口），配置完成后自动启动所有服务。

## HTTP 接口

server 模式提供完整的 HTTP API，支持纯 HTTP 调用（不走 `jpy --remote`）。完整文档见：[`HTTP_REMOTE_CONTROL.md`](./HTTP_REMOTE_CONTROL.md)。

| 接口 | 方法 | 说明 |
|------|------|------|
| /health | GET | 健康检查 |
| /version | GET | 版本信息 |
| /exec | POST | 执行 CLI 命令（同步） |
| /exec/async | POST | 执行 CLI 命令（异步） |
| /shell | POST | 执行系统命令（同步） |
| /shell/async | POST | 执行系统命令（异步） |
| /shell/task | GET | 查询异步任务 |
| /shell/tasks | GET | 列出所有任务 |
| /shell/kill | GET | 终止任务 |
| /file/upload | POST | 上传文件 |
| /file/download | POST | 下载文件 |
| /file/chunk/init | POST | 初始化分片上传 |
| /file/chunk/upload | POST | 上传分片 |
| /file/chunk/complete | POST | 完成分片上传 |

最常用调用方式：

```bash
curl -X POST http://192.168.1.100:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","list","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```

说明：

- `/exec` 的 `args` 是去掉 `jpy` 后的参数数组。
- 禁止在 `args` 内传 `server` 和 `--remote`。
- 文件类命令的路径都是被控端路径；主控端文件需先用 `/file/upload` 上传。
- 长任务用 `/exec/async`，再通过 `/shell/task` 查询。

**timeout 参数：** `/exec/async` 的 `timeout=0` 表示无限，不传或小于 0 表示默认 600 秒

## 输出格式

所有命令支持 `-o plain`（默认）和 `-o json` 两种输出模式。

AI 调用建议使用 `-o json` 便于解析。

## 退出码

| 码 | 含义 |
|----|------|
| 0 | 成功 |
| 1 | 失败 |
| 124 | 超时 |
| 137 | 被终止 |

## License

MIT
