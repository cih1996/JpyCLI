# JPY CLI 中文使用手册

JPY 中间件管理命令行工具，面向 AI、脚本和上层 UI 调用设计，零配置、无状态、命令一行完成。

## 基本约定

### 中间件连接参数

调用中间件接口的命令统一使用以下参数：

| 参数 | 必填 | 说明 |
|------|------|------|
| `-s, --server` | 是 | 中间件地址，例如 `172.25.0.251` 或 `http://172.25.0.251` |
| `-u, --user` | 是 | 中间件用户名 |
| `-p, --password` | 是 | 中间件密码 |
| `-o, --output` | 否 | 输出格式：`plain` 或 `json`，默认 `plain` |

### 输出格式

所有主要命令统一支持两种输出：

```bash
# 默认 plain，适合人直接查看
jpy device list -s 172.25.0.251 -u admin -p 123456

# JSON，适合 UI、脚本、自动化系统解析
jpy device list -s 172.25.0.251 -u admin -p 123456 -o json
```

### 远程调用

被控端先启动 HTTP 服务：

```bash
jpy server --port 9090
```

启动后服务会一直挂起监听，默认端口是 `9090`。远程通讯方式是 **HTTP，不是 WebSocket**。

主控端调用方式：

```bash
jpy --remote 192.168.1.100:9090 device list -s 172.25.0.251 -u admin -p 123456 -o json
```

远程支持范围：

| 类型 | 是否支持远程 | 说明 |
|------|--------------|------|
| `device` | 支持 | 走全局 `jpy --remote <host:port> device ...` |
| `device middleware` | 支持 | 包括中间件固件升级、ROM 上传/列表/刷机/状态/详情 |
| `com` | 支持 | 适合 COM 口在被控端机器上的场景 |
| `flash` | 支持 | 适合刷机脚本、COM 口、fastboot 环境都在被控端的场景 |
| `stress` | 支持 | 长时间任务建议异步 |
| `shell` | 支持 | 专用写法：`jpy shell --remote <host:port> ...` |
| `file` | 支持 | 自带 `--remote` 参数，用于上传/下载文件 |
| `update` | 支持 | 自带 `--remote` 参数，用于更新被控端 jpy 程序 |
| `version` | 支持 | 自带 `--remote` 参数，用于查看被控端版本 |
| `server` | 不支持远程执行 | 防止递归启动服务 |

注意：`file`、`update`、`version` 这三个命令的 `--remote` 是命令自己的参数，不走全局转发；其它普通命令使用 `jpy --remote <host:port> <命令>`。

长任务建议使用异步：

```bash
jpy --remote 192.168.1.100:9090 flash run ... --async --async-timeout 0
jpy shell --remote 192.168.1.100:9090 --task <task_id>
jpy shell --remote 192.168.1.100:9090 --kill <task_id>
```

远程接口对应关系：

| 场景 | HTTP 接口 |
|------|-----------|
| 普通命令同步执行 | `POST /exec` |
| 普通命令异步执行 | `POST /exec/async` |
| 系统 shell 同步执行 | `POST /shell` |
| 系统 shell 异步执行 | `POST /shell/async` |
| 查询任务 | `GET /shell/task?id=<task_id>` |
| 终止任务 | `GET /shell/kill?id=<task_id>` |
| 文件上传/下载 | `/file/*` |
| 版本/健康检查 | `GET /version`、`GET /health` |

---

## 文档索引

- [`README.md`](./README.md)：项目总览
- [`HTTP_REMOTE_CONTROL.md`](./HTTP_REMOTE_CONTROL.md)：纯 HTTP 远程控制文档
- [`HANDOFF.md`](./HANDOFF.md)：项目交接文档

## 命令总览

```bash
jpy device list    -s <server> -u <user> -p <pass> [--ip] [--uuid] [--seat] [-l] [-o plain|json]
jpy device shell   "<cmd>" -s <server> -u <user> -p <pass> [--seat | --ip] [-o plain|json]
jpy device reboot  -s <server> -u <user> -p <pass> [--seat] [--ip] [--uuid]
jpy device usb     -s <server> -u <user> -p <pass> --mode host|device [--seat] [--ip]
jpy device adb     -s <server> -u <user> -p <pass> --set on|off [--seat] [--ip]
jpy device status  -s <server> -u <user> -p <pass> [--detail] [-o plain|json]

jpy device middleware upgrade --file <firmware-file> -s <server> -u <user> -p <pass> [--required=true|false] [-o plain|json]
jpy device middleware rom upload --file <rom-file> -s <server> -u <user> -p <pass> [-o plain|json]
jpy device middleware rom list -s <server> -u <user> -p <pass> [-o plain|json]
jpy device middleware rom flash --seat <seat> --sn <sn> --image <image> -s <server> -u <user> -p <pass> [--mode 2] [-o plain|json]
jpy device middleware rom status -s <server> -u <user> -p <pass> [--seat <seat>] [--sn <sn>] [-o plain|json]
jpy device middleware rom detail --seat <seat> --session <session> -s <server> -u <user> -p <pass> [-o plain|json]

jpy com list       [-o plain|json]
jpy com devices    [--port COM3] [--skip-connect] [-o plain|json]
jpy com set-mode   --port COM3 --mode hub|otg [--channel 1|1,2,3|2-20|0] [--skip-connect] [-o plain|json]
jpy com restart    --port COM3 [--channel 1|1,2,3|2-20|0] [--skip-connect] [-o plain|json]

jpy flash run      --com COM3 --mw <server> --ip-start <起始IP> --script <刷机脚本> [--ch 1-10] [--dry] [-y]
jpy file push      <local-file> --remote <host:port> [--dest <path>] [--timeout N]
jpy file pull      <url> --remote <host:port> [--dest <path>] [--timeout N]
jpy update         <本地文件|URL> --remote <host:port>
jpy stress user    -s <ws-server> -k <secret-key> -c <config.json> [--device 1,2,3] [--loop N] [--interval 3m] [--timeout 10m]
jpy shell          --remote <host:port> -c "<cmd>" [--timeout N] [--async] [--task ID] [--tasks] [--kill ID]
jpy server         [--port 9090]
jpy version        [--remote <host:port>] [-o plain|json]
jpy --remote <host:port> <任意命令> [--async] [--async-timeout N]
```

---

## 1. 设备管理：`device`

### 1.1 查看设备列表

```bash
jpy device list -s 172.25.0.251 -u admin -p 123456
jpy device list -s 172.25.0.251 -u admin -p 123456 -o json
jpy device list -s 172.25.0.251 -u admin -p 123456 --seat 3
jpy device list -s 172.25.0.251 -u admin -p 123456 --ip 10.0.0.5
jpy device list -s 172.25.0.251 -u admin -p 123456 --uuid 7b9f2b7a
```

### 1.2 执行设备命令

```bash
jpy device shell "ls /sdcard" -s 172.25.0.251 -u admin -p 123456 --seat 3
jpy device shell "getprop ro.product.model" -s 172.25.0.251 -u admin -p 123456 --ip 10.0.0.5 -o json
```

### 1.3 重启设备

```bash
jpy device reboot -s 172.25.0.251 -u admin -p 123456
jpy device reboot -s 172.25.0.251 -u admin -p 123456 --seat 3
jpy device reboot -s 172.25.0.251 -u admin -p 123456 --ip 10.0.0.5
jpy device reboot -s 172.25.0.251 -u admin -p 123456 --uuid 7b9f2b7a
```

### 1.4 USB / ADB 控制

```bash
jpy device usb -s 172.25.0.251 -u admin -p 123456 --mode host --seat 3
jpy device usb -s 172.25.0.251 -u admin -p 123456 --mode device --seat 3
jpy device adb -s 172.25.0.251 -u admin -p 123456 --set on --seat 3
jpy device adb -s 172.25.0.251 -u admin -p 123456 --set off --seat 3
```

### 1.5 查看中间件状态

```bash
jpy device status -s 172.25.0.251 -u admin -p 123456
jpy device status -s 172.25.0.251 -u admin -p 123456 --detail -o json
```

---

## 2. 中间件固件升级：`device middleware upgrade`

该命令用于把本地中间件固件上传到目标中间件，并立即调用升级接口。

### 命令格式

```bash
jpy device middleware upgrade --file <firmware-file> -s <server> -u <user> -p <pass> [--required=true|false] [-o plain|json]
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--file` | 是 | 本地固件文件路径 |
| `--required` | 否 | 是否强制执行升级，默认 `true` |
| `-s, --server` | 是 | 中间件地址 |
| `-u, --user` | 是 | 用户名 |
| `-p, --password` | 是 | 密码 |
| `-o, --output` | 否 | `plain` 或 `json` |

### 示例

```bash
# 上传固件并执行强制升级
jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456

# JSON 输出，推荐给 UI 使用
jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456 -o json

# 非强制升级
jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456 --required=false
```

### plain 输出示例

```text
SERVER	172.25.0.251
PACKAGE_ID	123
REQUIRED	true
MESSAGE	upgrade submitted
STATUS	success
```

### JSON 输出示例

```json
{"success":true,"server":"http://172.25.0.251","package_id":123,"required":true,"message":"upgrade submitted"}
```

### 当前内部流程

1. 登录中间件，获取 token。
2. 上传本地固件到 `/sys/upload`。
3. 从上传响应中读取 `package_id`。
4. 调用 `/sys/update?required=<true|false>&id=<package_id>` 执行升级。

---

## 3. 中间件接口 ROM 刷机：`device middleware rom`

这组命令用于通过中间件自身接口管理 ROM 包并发起刷机。它和 `flash run` 的 COM 口刷机不是同一条链路。

当前提供基础拆分命令：上传、列包、发起刷机、查状态、查详情。自动轮询和关键字判断后续可以在这些基础能力上继续封装。

### 3.1 上传 ROM 包

```bash
jpy device middleware rom upload --file ./rom.zip -s 172.25.0.251 -u admin -p 123456
jpy device middleware rom upload --file ./rom.zip -s 172.25.0.251 -u admin -p 123456 -o json
```

参数：

| 参数 | 必填 | 说明 |
|------|------|------|
| `--file` | 是 | 本地 ROM 包路径 |
| `-s, --server` | 是 | 中间件地址 |
| `-u, --user` | 是 | 用户名 |
| `-p, --password` | 是 | 密码 |
| `-o, --output` | 否 | `plain` 或 `json` |

plain 输出示例：

```text
SERVER	172.25.0.251
FILE	rom.zip
MESSAGE	upload success
STATUS	success
```

JSON 输出示例：

```json
{"success":true,"server":"http://172.25.0.251","file":"rom.zip","message":"upload success"}
```

内部接口：`POST /box/upload`。

### 3.2 查看 ROM 包列表

```bash
jpy device middleware rom list -s 172.25.0.251 -u admin -p 123456
jpy device middleware rom list -s 172.25.0.251 -u admin -p 123456 -o json
```

plain 输出字段：

```text
NAME	MODEL	VERSION	DESC
```

JSON 输出结构：

```json
{"server":"http://172.25.0.251","total":1,"packages":[{"name":"1767856234","model":"xxx","version":"xxx","desc":"xxx"}]}
```

说明：后续 `rom flash --image` 通常填写这里返回的 `name`。

内部链路：Guard WebSocket `/box/guard`，指令码 `113`。

### 3.3 发起 ROM 刷机

```bash
jpy device middleware rom flash --seat 3 --sn ABC123 --image 1767856234 -s 172.25.0.251 -u admin -p 123456
jpy device middleware rom flash --seat 3 --sn ABC123 --image 1767856234 --mode 2 -s 172.25.0.251 -u admin -p 123456 -o json
```

参数：

| 参数 | 必填 | 说明 |
|------|------|------|
| `--seat` | 是 | 盘位号，必须大于 0 |
| `--sn` | 是 | 设备序列号 |
| `--image` | 是 | ROM 镜像 ID，通常使用 `rom list` 返回的 `name` |
| `--mode` | 否 | 刷机模式，默认 `2` |
| `-s, --server` | 是 | 中间件地址 |
| `-u, --user` | 是 | 用户名 |
| `-p, --password` | 是 | 密码 |
| `-o, --output` | 否 | `plain` 或 `json` |

plain 输出示例：

```text
SERVER	172.25.0.251
SEAT	3
SN	ABC123
IMAGE	1767856234
MODE	2
MESSAGE	flash request submitted
STATUS	success
```

JSON 输出示例：

```json
{"success":true,"server":"http://172.25.0.251","seat":3,"sn":"ABC123","image":"1767856234","mode":2,"message":"flash request submitted"}
```

内部链路：Guard WebSocket `/box/guard`，指令码 `119`。

### 3.4 查询 ROM 刷机状态

```bash
jpy device middleware rom status -s 172.25.0.251 -u admin -p 123456
jpy device middleware rom status -s 172.25.0.251 -u admin -p 123456 --seat 3
jpy device middleware rom status -s 172.25.0.251 -u admin -p 123456 --sn ABC123 -o json
```

可选过滤参数：

| 参数 | 说明 |
|------|------|
| `--seat` | 只看指定盘位 |
| `--sn` | 只看指定设备序列号 |

plain 输出字段：

```text
SEAT	SN	MODE	STATUS	SESSION	IMAGE	QUEUE	START	END	ERROR
```

JSON 输出结构：

```json
{
  "server":"http://172.25.0.251",
  "total":1,
  "items":[
    {
      "seat":3,
      "sn":"ABC123",
      "mode":2,
      "status":1,
      "session":"session-id",
      "image":"1767856234",
      "queue_time":1710000000,
      "start_time":1710000001,
      "end_time":0,
      "last_error":""
    }
  ]
}
```

说明：`status` 是中间件原始状态码，CLI 当前不擅自翻译，便于 UI 按真实值判断。

内部链路：Guard WebSocket `/box/guard`，指令码 `117`。

### 3.5 查询 ROM 刷机详情日志

```bash
jpy device middleware rom detail --seat 3 --session session-id -s 172.25.0.251 -u admin -p 123456
jpy device middleware rom detail --seat 3 --session session-id -s 172.25.0.251 -u admin -p 123456 -o json
```

参数：

| 参数 | 必填 | 说明 |
|------|------|------|
| `--seat` | 是 | 盘位号，必须大于 0 |
| `--session` | 是 | 刷机会话 ID，通常来自 `rom status` 返回的 `session` |
| `-s, --server` | 是 | 中间件地址 |
| `-u, --user` | 是 | 用户名 |
| `-p, --password` | 是 | 密码 |
| `-o, --output` | 否 | `plain` 或 `json` |

plain 输出示例：

```text
SERVER	172.25.0.251
SEAT	3
SESSION	session-id
DETAIL
刷机详情日志内容...
```

JSON 输出示例：

```json
{"success":true,"server":"http://172.25.0.251","seat":3,"session":"session-id","detail":"刷机详情日志内容..."}
```

内部接口：`GET /box/detail?id=<seat>&session=<session>`。

---

## 4. COM 串口硬件控制：`com`

COM 命令用于控制 USB HUB 控制板，和中间件 ROM 接口刷机不同。

```bash
jpy com list -o json
jpy com devices --port COM3 -o json
jpy com set-mode --port COM3 --mode hub --channel 5
jpy com set-mode --port COM3 --mode otg --channel 2-20
jpy com set-mode --port COM3 --mode hub --channel 1,2,3
jpy com restart --port COM3 --channel 3
jpy com restart --port COM3 --channel 2-20
```

通道写法：

| 写法 | 说明 |
|------|------|
| `1` | 单通道 |
| `1,2,3` | 多个指定通道 |
| `2-20` | 范围通道 |
| `0` | 全部通道 |

---

## 5. COM 口批量刷机：`flash run`

该命令用于传统 COM 口刷机链路，会组合中间件设备操作、COM 通道切换和本地刷机脚本执行。

```bash
# 单通道刷机
jpy flash run --com COM3 --ch 1 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y

# 多通道刷机
jpy flash run --com COM3 --ch 1-10 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y

# 指定多个通道
jpy flash run --com COM3 --ch 1,3,5,7 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y

# 多 COM 口
jpy flash run --com COM3,COM4 --ch 1-20 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y

# 所有 COM 口
jpy flash run --com all --ch 1-20 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y

# 模拟运行，先看任务分配
jpy flash run --com COM3 --ch 1-5 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" --dry
```

工作流程：

1. 检查设备状态。
2. 发送 `reboot bootloader`。
3. 切换 COM 通道为 HUB 模式。
4. 等待 fastboot 设备出现。
5. 执行刷机脚本。
6. 切换回 OTG 模式。

---

## 6. 远程文件传输：`file`

```bash
# 上传本地文件到远程机器
jpy file push ./rom.zip --remote 192.168.1.100:9090 --dest D:\flash\rom.zip

# 让远程机器从 URL 下载文件
jpy file pull "https://example.com/rom.zip" --remote 192.168.1.100:9090 --dest D:\flash\rom.zip

# 大文件增加超时时间
jpy file push ./large.zip --remote 192.168.1.100:9090 --timeout 3600
```

---

## 7. 远程更新 JPY CLI：`update`

```bash
# 从本地文件更新远程 jpy 程序
jpy update ./jpy-windows-amd64.exe --remote 192.168.1.100:9090

# 从 URL 更新远程 jpy 程序
jpy update https://example.com/jpy.exe --remote 192.168.1.100:9090
```

注意：这个命令更新的是远程机器上的 `jpy` 程序，不是中间件固件。中间件固件升级请使用：

```bash
jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456
```

---

## 8. 压力测试：`stress user`

```bash
jpy stress user -s wss://home.accjs.cn/ws -k YOUR_SECRET_KEY -c config.json
jpy stress user -k YOUR_SECRET_KEY -c config.json --device 123,456,789 --loop 3 --interval 5m
jpy stress user -k YOUR_SECRET_KEY -c config.json --loop 0 --interval 3m
jpy stress user -k YOUR_SECRET_KEY -c config.json --loop 0 --debug
jpy stress user -k YOUR_SECRET_KEY -c config.json --timeout 15m --log-dir /var/log/stress
```

参数：

| 参数 | 说明 |
|------|------|
| `-s, --server` | WebSocket 服务地址 |
| `-k, --key` | 登录密钥，必填 |
| `-c, --config` | 改机配置文件路径，必填 |
| `--device` | 指定设备 ID 列表，逗号分隔 |
| `--loop` | 循环次数，`0` 表示无限循环 |
| `--interval` | 循环间隔时间 |
| `--timeout` | 单轮超时时间 |
| `--log-dir` | 日志目录 |
| `--debug` | 调试模式，遇到失败立即停止 |

---

## 9. 远程 Shell：`shell`

```bash
jpy shell --remote 192.168.1.100:9090 -c "dir C:\Users"
jpy shell --remote 192.168.1.100:9090 -c "fastboot devices" --timeout 60
jpy shell --remote 192.168.1.100:9090 -c "long-running-command" --async --timeout 900
jpy shell --remote 192.168.1.100:9090 --task <task_id>
jpy shell --remote 192.168.1.100:9090 --tasks
jpy shell --remote 192.168.1.100:9090 --kill <task_id>
```

---

## 10. Server 模式：`server`

被控端启动方式：

```bash
jpy server --port 9090
```

说明：

- 默认端口：`9090`
- 启动后进程会一直挂起监听
- 通讯协议：HTTP
- 监听地址：`:端口`，局域网其它机器可通过 `http://被控端IP:端口` 访问
- 不支持通过远程调用再启动 `server`

常用 HTTP API（完整纯 HTTP 调用文档见 [`HTTP_REMOTE_CONTROL.md`](./HTTP_REMOTE_CONTROL.md)）：

| 接口 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/version` | GET | 版本信息 |
| `/exec` | POST | 同步执行 CLI 命令 |
| `/exec/async` | POST | 异步执行 CLI 命令 |
| `/shell` | POST | 同步执行系统命令 |
| `/shell/async` | POST | 异步执行系统命令 |
| `/shell/task` | GET | 查询异步任务 |
| `/shell/tasks` | GET | 列出异步任务 |
| `/shell/kill` | GET | 终止异步任务 |
| `/file/upload` | POST | 上传文件 |
| `/file/download` | POST | 下载文件 |
| `/file/chunk/init` | POST | 初始化分片上传 |
| `/file/chunk/upload` | POST | 上传分片 |
| `/file/chunk/complete` | POST | 完成分片上传 |

---

## 退出码

| 退出码 | 含义 |
|--------|------|
| `0` | 成功 |
| `1` | 失败 |
| `124` | 超时 |
| `137` | 被终止 |
