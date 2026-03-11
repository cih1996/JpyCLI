# JPY CLI — AI Tool Reference

> 版本：v5.0 | 更新：2026-03-11
> 本文档专为 AI Agent 设计，提供精确的命令签名、参数约束、输出 schema。

## 全局约束

### 认证参数（所有命令必填）

| 参数 | 短写 | 类型 | 说明 |
|------|------|------|------|
| `--server` | `-s` | string | 中间件服务器地址（自动补 `https://`） |
| `--username` | `-u` | string | 登录用户名 |
| `--password` | `-p` | string | 登录密码 |

### 输出模式

| 参数 | 短写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--output` | `-o` | string | `plain` | `plain`=制表符分隔 / `json`=JSON 单行 |

### 退出码

| 退出码 | 含义 |
|--------|------|
| 0 | 成功 |
| 1 | 失败（参数错误 / 认证失败 / 部分设备操作失败） |
| N>0 | shell 命令：透传设备端 exit code |

### 错误输出

- 错误信息输出到 stderr
- plain 模式的统计摘要输出到 stderr（`--- total: N, success: N, failed: N`）
- stdout 只包含数据

---

## 命令 1：device list

列出服务器上所有设备的详细状态。

### 签名

```
jpy device list -s <server> -u <user> -p <pass> [--ip <ip>] [--uuid <uuid>] [--seat <n>] [-l <n>] [-o plain|json]
```

### 参数

| 参数 | 短写 | 类型 | 默认值 | 必填 | 说明 |
|------|------|------|--------|------|------|
| `--ip` | — | string | `""` | 否 | 按 IP 模糊过滤 |
| `--uuid` | — | string | `""` | 否 | 按 UUID 模糊过滤 |
| `--seat` | — | int | `-1` | 否 | 按机位号精确过滤（-1=不过滤） |
| `--limit` | `-l` | int | `0` | 否 | 限制返回数量（0=不限） |

### plain 输出格式

```
SERVER\tSEAT\tUUID\tMODEL\tANDROID\tONLINE\tBIZ\tIP\tADB\tUSB
192.168.1.1\t1\tabc-123\tPixel 6\t13\ttrue\ttrue\t10.0.0.5\ttrue\tfalse
--- total: 1
```

- 分隔符：`\t`（制表符）
- 第一行：表头
- 最后一行（stderr）：`--- total: <N>`
- 空结果：`没有找到设备。`

### JSON 输出 schema

```json
{
  "total": 1,
  "devices": [
    {
      "server": "https://192.168.1.1",
      "seat": 1,
      "uuid": "abc-123",
      "model": "Pixel 6",
      "android": "13",
      "online": true,
      "biz_online": true,
      "ip": "10.0.0.5",
      "adb": true,
      "usb": false
    }
  ]
}
```

- 空结果：`{"total":0,"devices":[]}`

### 示例

```bash
# 列出所有设备
jpy device list -s 192.168.1.1 -u admin -p 123456

# JSON 输出 + 按 IP 过滤
jpy device list -s 192.168.1.1 -u admin -p 123456 -o json --ip 10.0.0

# 只看前 5 台
jpy device list -s 192.168.1.1 -u admin -p 123456 -l 5

# 查指定机位
jpy device list -s 192.168.1.1 -u admin -p 123456 --seat 3
```

---

## 命令 2：device shell

向指定设备发送 shell 命令并返回输出。必须通过 `--seat` 或 `--ip` 指定目标设备。

### 签名

```
jpy device shell "<command>" -s <server> -u <user> -p <pass> --seat <n> [-o plain|json]
jpy device shell -c "<command>" -s <server> -u <user> -p <pass> --ip <ip> [-o plain|json]
```

### 参数

| 参数 | 短写 | 类型 | 默认值 | 必填 | 说明 |
|------|------|------|--------|------|------|
| 位置参数 | — | string | — | 是* | shell 命令（与 `-c` 二选一） |
| `--command` | `-c` | string | `""` | 是* | shell 命令（与位置参数二选一） |
| `--seat` | — | int | `-1` | 是** | 目标机位号 |
| `--ip` | — | string | `""` | 是** | 目标设备 IP（自动解析为 seat） |

> *命令：位置参数和 `-c` 二选一，至少提供一个
> **定位：`--seat` 和 `--ip` 二选一，至少提供一个

### 执行降级链

1. f=14（Guard 批量命令通道）— 首选
2. f=289（DeviceAPI Shell）— f=14 失败时
3. Terminal（VT100 终端模拟）— f=289 返回 code 10 时

### plain 输出格式

```
<命令输出原文，保留换行>
```

- 直接输出命令结果到 stdout，无表头
- 错误信息输出到 stderr

### JSON 输出 schema

```json
{
  "server": "https://192.168.1.1",
  "seat": 3,
  "command": "ls -lh",
  "output": "total 4.0K\ndrwxr-xr-x 2 root root 4.0K ...",
  "success": true,
  "exit_code": 0
}
```

- 失败时：`"success": false, "error": "错误描述"`
- `exit_code` 字段仅在 f=14/f=289 通道可用

### 示例

```bash
# 按机位执行
jpy device shell "ls -lh /sdcard" -s 192.168.1.1 -u admin -p 123456 --seat 3

# 按 IP 执行
jpy device shell "getprop ro.build.version.release" -s 192.168.1.1 -u admin -p 123456 --ip 10.0.0.5

# JSON 输出
jpy device shell "df -h" -s 192.168.1.1 -u admin -p 123456 --seat 1 -o json

# 使用 -c 参数
jpy device shell -c "pm list packages" -s 192.168.1.1 -u admin -p 123456 --seat 3
```

---

## 命令 3：device reboot

重启指定设备。支持按 seat/ip/uuid 过滤，不指定过滤条件则操作所有设备。

### 签名

```
jpy device reboot -s <server> -u <user> -p <pass> [--seat <n>] [--ip <ip>] [--uuid <uuid>] [-o plain|json]
```

### 参数

| 参数 | 短写 | 类型 | 默认值 | 必填 | 说明 |
|------|------|------|--------|------|------|
| `--seat` | — | int | `-1` | 否 | 按机位号过滤（-1=不过滤） |
| `--ip` | — | string | `""` | 否 | 按 IP 过滤 |
| `--uuid` | — | string | `""` | 否 | 按 UUID 过滤 |

### plain 输出格式

```
SERVER\tSEAT\tUUID\tSTATUS
192.168.1.1\t3\tabc-123\tOK
192.168.1.1\t5\tdef-456\tFAIL:timeout
--- total: 2, success: 1, failed: 1
```

- STATUS 值：`OK` 或 `FAIL:<错误描述>`
- 摘要行输出到 stderr
- 有失败时退出码为 1

### JSON 输出 schema

```json
{
  "total": 2,
  "success": 1,
  "failed": 1,
  "results": [
    {"server": "192.168.1.1", "seat": 3, "uuid": "abc-123", "ok": true, "error": ""},
    {"server": "192.168.1.1", "seat": 5, "uuid": "def-456", "ok": false, "error": "timeout"}
  ]
}
```

### 示例

```bash
# 重启指定机位
jpy device reboot -s 192.168.1.1 -u admin -p 123456 --seat 3

# 重启所有设备
jpy device reboot -s 192.168.1.1 -u admin -p 123456

# JSON 输出
jpy device reboot -s 192.168.1.1 -u admin -p 123456 --seat 3 -o json
```

---

## 命令 4：device usb

切换设备 USB 模式（host/device）。

### 签名

```
jpy device usb -s <server> -u <user> -p <pass> --mode <host|device> [--seat <n>] [--ip <ip>] [--uuid <uuid>] [-o plain|json]
```

### 参数

| 参数 | 短写 | 类型 | 默认值 | 必填 | 说明 |
|------|------|------|--------|------|------|
| `--mode` | `-m` | string | — | 是 | `host`(=OTG) 或 `device`(=USB) |
| `--seat` | — | int | `-1` | 否 | 按机位号过滤 |
| `--ip` | — | string | `""` | 否 | 按 IP 过滤 |
| `--uuid` | — | string | `""` | 否 | 按 UUID 过滤 |

> mode 别名：`host`=`otg`, `device`=`usb`

### 输出格式

与 `device reboot` 完全一致（plain 和 JSON schema 相同）。

### 示例

```bash
# 切换到 host 模式
jpy device usb -s 192.168.1.1 -u admin -p 123456 --mode host --seat 3

# 切换到 device 模式（所有设备）
jpy device usb -s 192.168.1.1 -u admin -p 123456 --mode device
```

---

## 命令 5：device adb

控制设备 ADB 开关。

### 签名

```
jpy device adb -s <server> -u <user> -p <pass> --set <on|off> [--seat <n>] [--ip <ip>] [--uuid <uuid>] [-o plain|json]
```

### 参数

| 参数 | 短写 | 类型 | 默认值 | 必填 | 说明 |
|------|------|------|--------|------|------|
| `--set` | — | string | — | 是 | `on`(=true) 或 `off`(=false) |
| `--seat` | — | int | `-1` | 否 | 按机位号过滤 |
| `--ip` | — | string | `""` | 否 | 按 IP 过滤 |
| `--uuid` | — | string | `""` | 否 | 按 UUID 过滤 |

> set 别名：`on`=`true`, `off`=`false`

### 输出格式

与 `device reboot` 完全一致（plain 和 JSON schema 相同）。

### 示例

```bash
# 开启 ADB
jpy device adb -s 192.168.1.1 -u admin -p 123456 --set on --seat 3

# 关闭所有设备 ADB
jpy device adb -s 192.168.1.1 -u admin -p 123456 --set off
```

---

## 命令 6：device status

查看服务器和设备状态概览。

### 签名

```
jpy device status -s <server> -u <user> -p <pass> [--detail] [-o plain|json]
```

### 参数

| 参数 | 短写 | 类型 | 默认值 | 必填 | 说明 |
|------|------|------|--------|------|------|
| `--detail` | — | bool | `false` | 否 | 显示详细信息（固件版本、网速等） |

### plain 输出格式

```
SERVER\tSTATUS\tLICENSE\tDEVICES\tBIZ\tIP\tADB\tUSB\tOTG
192.168.1.1\tonline\tactive\t10\t8\t10\t5\t3\t7
```

- 分隔符：`\t`（制表符）
- 第一行：表头
- 无 stderr 摘要行

### JSON 输出 schema

```json
[
  {
    "server_url": "https://192.168.1.1",
    "status": "online",
    "license_status": "active",
    "device_count": 10,
    "biz_online_count": 8,
    "ip_count": 10,
    "adb_count": 5,
    "usb_count": 3,
    "otg_count": 7
  }
]
```

### 示例

```bash
# 基本状态
jpy device status -s 192.168.1.1 -u admin -p 123456

# 详细信息
jpy device status -s 192.168.1.1 -u admin -p 123456 --detail

# JSON 输出
jpy device status -s 192.168.1.1 -u admin -p 123456 -o json
```

---

## 错误处理

### 常见错误模式

| 错误信息 | 原因 | 处理方式 |
|----------|------|----------|
| `必须指定 --server / -s 参数` | 缺少 -s | 补充参数 |
| `必须指定 --username / -u 参数` | 缺少 -u | 补充参数 |
| `必须指定 --password / -p 参数` | 缺少 -p | 补充参数 |
| `登录 <server> 失败: ...` | 认证失败 | 检查凭证或服务器地址 |
| `没有匹配的设备` | 过滤条件无匹配 | 放宽过滤条件 |
| `必须指定 --ip 或 --seat` | shell 缺少目标 | 补充定位参数 |
| `必须指定 --mode 参数 (host/device)` | usb 缺少模式 | 补充 --mode |
| `必须指定 --set 参数 (on/off)` | adb 缺少状态 | 补充 --set |

### 错误输出位置

- 所有错误信息 → stderr
- 退出码 → 非零

---

## AI 调用最佳实践

### 1. 先查后操作

```bash
# 先列出设备，确认目标存在
jpy device list -s $SERVER -u $USER -p $PASS -o json --seat 3
# 再执行操作
jpy device shell "reboot" -s $SERVER -u $USER -p $PASS --seat 3
```

### 2. 用 JSON 解析结果

```bash
# 获取所有在线设备的 seat 列表
jpy device list -s $SERVER -u $USER -p $PASS -o json | jq '.devices[] | select(.online==true) | .seat'
```

### 3. 批量操作（不指定过滤条件 = 操作所有设备）

```bash
# 重启所有设备
jpy device reboot -s $SERVER -u $USER -p $PASS
# 所有设备切 host 模式
jpy device usb -s $SERVER -u $USER -p $PASS --mode host
```

### 4. 检查操作结果

```bash
# 通过退出码判断
jpy device reboot -s $SERVER -u $USER -p $PASS --seat 3 && echo "成功" || echo "失败"

# 通过 JSON 判断
jpy device reboot -s $SERVER -u $USER -p $PASS --seat 3 -o json | jq '.failed'
```

---

## 命令 7：server

启动 HTTP 代理服务，接收远程 CLI 命令转发。

### 签名

```
jpy server [--port <n>]
```

### 参数

| 参数 | 短写 | 类型 | 默认值 | 必填 | 说明 |
|------|------|------|--------|------|------|
| `--port` | — | int | `9090` | 否 | 监听端口 |

### HTTP 端点

#### POST /exec

请求体：
```json
{
  "args": ["device", "list", "-s", "192.168.1.1", "-u", "admin", "-p", "123", "-o", "json"]
}
```

响应体：
```json
{
  "exit_code": 0,
  "stdout": "{\"total\":5,\"devices\":[...]}\n",
  "stderr": ""
}
```

- HTTP 状态码始终 200（业务成功/失败通过 exit_code 区分）
- 请求体上限 1MB
- 禁止传入 `server` 和 `--remote` 参数（防递归）

#### GET /health

```json
{"status": "ok"}
```

### 示例

```bash
# 启动（默认 9090 端口）
jpy server

# 指定端口
jpy server --port 8888

# 直接 curl 调用
curl -X POST http://10.0.0.5:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","list","-s","192.168.1.1","-u","admin","-p","123","-o","json"]}'
```

---

## 全局参数：--remote

将命令转发到远端 jpy server 执行，stdout/stderr/退出码完整透传。

### 签名

```
jpy --remote <host:port> <任意命令及参数>
```

### 说明

- 在 Cobra 解析之前拦截，位置无关（放在命令前后均可）
- 自动补全 `http://` 前缀
- 支持 `--remote host:port` 和 `--remote=host:port` 两种格式
- 客户端超时 120 秒

### 示例

```bash
# 远端列出设备
jpy --remote 10.0.0.5:9090 device list -s 192.168.1.1 -u admin -p 123

# 远端执行 shell
jpy --remote 10.0.0.5:9090 device shell "ls /sdcard" -s 192.168.1.1 -u admin -p 123 --seat 3

# 远端重启 + JSON 输出
jpy --remote 10.0.0.5:9090 device reboot -s 192.168.1.1 -u admin -p 123 --seat 3 -o json

# --remote 放后面也行
jpy device list -s 192.168.1.1 -u admin -p 123 --remote 10.0.0.5:9090
```

---

## 命令 8：com list

列举本机所有可用 COM 串口。

### 签名

```
jpy com list [-o plain|json]
```

### 参数

无额外参数。仅支持全局 `-o` 输出模式。

### plain 输出格式

```
/dev/cu.wchusbserial124230
COM3
```

每行一个端口名，无表头。

### JSON 输出 schema

```json
{"ports": ["/dev/cu.wchusbserial124230", "COM3"]}
```

空结果：`{"ports": []}`

### 示例

```bash
jpy com list
jpy com list -o json
jpy --remote 10.0.0.5:9090 com list
```

---

## 命令 9：com devices

获取 COM 串口设备的通道状态（UID、版本、MAC、IP、20 路通道模式）。

### 签名

```
jpy com devices [--port <串口名>] [--skip-connect] [-o plain|json]
```

### 参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `--port` | string | `""` | 否 | 串口名称（不指定则自动选择唯一候选） |
| `--skip-connect` | bool | `false` | 否 | 跳过 0x02 建连指令 |

### plain 输出格式

```
PORT	UID	VERSION	MAC	IP
/dev/cu.wch	12345	1.23	00:11:22:33:44:55	192.168.1.100
CH	PLUG	MODE
1	有主板	HUB
2	无主板	OFF
...
```

### JSON 输出 schema

```json
{
  "port": "/dev/cu.wchusbserial124230",
  "uid": "12345",
  "version": "1.23",
  "mac": "00:11:22:33:44:55",
  "ip": "192.168.1.100",
  "mask": "255.255.255.0",
  "gateway": "192.168.1.1",
  "fan_mode": "0x00",
  "channels": [
    {"channel": 1, "plug": "有主板", "mode": "HUB", "mode_code": 1},
    {"channel": 2, "plug": "无主板", "mode": "OFF", "mode_code": 0}
  ]
}
```

### 示例

```bash
jpy com devices
jpy com devices --port /dev/cu.wchusbserial124230
jpy com devices --port COM3 --skip-connect -o json
```

---

## 命令 10：com set-mode

设置 COM 设备通道工作模式（HUB/OTG），单次有效。

### 签名

```
jpy com set-mode --port <串口名> --mode <hub|otg> [--channel <N>] [--skip-connect] [-o plain|json]
```

### 参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `--port` | string | `""` | 否 | 串口名称 |
| `--mode` | string | — | 是 | `hub` 或 `otg` |
| `--channel` | int | `0` | 否 | 通道号（0=所有通道，1-20=指定通道） |
| `--skip-connect` | bool | `false` | 否 | 跳过建连 |

### plain 输出格式

```
PORT	CHANNEL	STATUS
COM3	0	OK
--- total: 1, success: 1, failed: 0
```

### JSON 输出 schema

```json
{"port": "COM3", "channel": 0, "success": true, "message": "已设置为 hub 模式"}
```

### 示例

```bash
# 所有通道切 HUB
jpy com set-mode --port COM3 --mode hub

# 通道 5 切 OTG
jpy com set-mode --port COM3 --mode otg --channel 5

# JSON 输出
jpy com set-mode --port COM3 --mode hub -o json
```

---

## 命令 11：com restart

重启 COM 设备通道。

### 签名

```
jpy com restart --port <串口名> [--channel <N>] [--skip-connect] [-o plain|json]
```

### 参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `--port` | string | `""` | 否 | 串口名称 |
| `--channel` | int | `0` | 否 | 通道号（0=所有通道，1-20=指定通道） |
| `--skip-connect` | bool | `false` | 否 | 跳过建连 |

### plain 输出格式

```
PORT	CHANNEL	STATUS
COM3	0	OK
--- total: 1, success: 1, failed: 0
```

### JSON 输出 schema

```json
{"port": "COM3", "channel": 0, "success": true, "message": "重启成功"}
```

### 示例

```bash
# 重启所有通道
jpy com restart --port COM3

# 重启通道 3
jpy com restart --port COM3 --channel 3

# 远程重启
jpy --remote 10.0.0.5:9090 com restart --port COM3
```

---

## 命令 12：shell

在远端机器上执行系统 shell 命令。支持同步（等结果）和异步（后台跑，轮询日志）两种模式。必须配合 `--remote` 使用。

### 签名

```
# 同步执行
jpy shell --remote <host:port> -c "<命令>" [--timeout N] [-o plain|json]

# 异步提交
jpy shell --remote <host:port> -c "<命令>" --async [--timeout N] [-o plain|json]

# 查询任务
jpy shell --remote <host:port> --task <task_id> [-o plain|json]

# 列出所有任务
jpy shell --remote <host:port> --tasks [-o plain|json]

# 终止任务
jpy shell --remote <host:port> --kill <task_id>
```

### 参数

| 参数 | 短写 | 类型 | 默认值 | 必填 | 说明 |
|------|------|------|--------|------|------|
| `--cmd` | `-c` | string | — | 是* | shell 命令 |
| `--timeout` | — | int | `120` | 否 | 超时秒数（异步默认 600） |
| `--async` | — | bool | `false` | 否 | 异步模式 |
| `--task` | — | string | — | 否 | 查询任务 ID |
| `--tasks` | — | bool | `false` | 否 | 列出所有任务 |
| `--kill` | — | string | — | 否 | 终止任务 ID |

> *执行模式下必填，查询/列表/终止模式下不需要

### 同步模式输出

plain：直接输出命令的 stdout，stderr 输出到 stderr，退出码透传。

JSON：
```json
{"exit_code": 0, "stdout": "命令输出...", "stderr": ""}
```

超时退出码为 124（与 Linux timeout 命令一致）。

### 异步模式输出

提交：
```json
{"task_id": "a1b2c3d4e5f6", "status": "running"}
```

查询（running）：
```json
{"task_id": "a1b2c3d4e5f6", "status": "running", "exit_code": 0, "stdout": "进行中的输出...", "stderr": "", "elapsed": "12.3s", "command": "fastboot flash system system.img"}
```

查询（done）：
```json
{"task_id": "a1b2c3d4e5f6", "status": "done", "exit_code": 0, "stdout": "完整输出...", "stderr": "", "elapsed": "180.5s", "command": "fastboot flash system system.img"}
```

### AI 调用最佳实践（刷机场景）

```bash
# 1. 提交刷机任务（异步，超时 15 分钟）
jpy shell --remote 10.0.0.5:9090 -c "fastboot flash system system.img" --async --timeout 900 -o json

# 2. 定期查询进度
jpy shell --remote 10.0.0.5:9090 --task a1b2c3d4e5f6 -o json

# 3. 如果需要终止
jpy shell --remote 10.0.0.5:9090 --kill a1b2c3d4e5f6

# 4. 查看所有任务
jpy shell --remote 10.0.0.5:9090 --tasks -o json
```

### 示例

```bash
# 同步：查看远端目录
jpy shell --remote 10.0.0.5:9090 -c "dir C:\Users"

# 同步：查看远端进程
jpy shell --remote 10.0.0.5:9090 -c "tasklist | findstr fastboot"

# 异步：刷机
jpy shell --remote 10.0.0.5:9090 -c "cd D:\flash && flash.bat" --async --timeout 900

# 查询任务
jpy shell --remote 10.0.0.5:9090 --task a1b2c3d4e5f6
```

---

## 命令 13：flash run

批量刷机工具，集成 device + com 操作，支持按 COM 口和通道批量刷机。

### 签名

```
jpy flash run --com <COM口> --mw <中间件> --script <脚本路径> [--ch <通道>] [--ip-base <IP基数>] [--dry] [-y] [-o plain|json]
```

### 参数

| 参数 | 短写 | 类型 | 默认值 | 必填 | 说明 |
|------|------|------|--------|------|------|
| `--com` | — | string | — | 是 | COM口: COM3, COM4, COM6 或 all |
| `--mw` | — | string | — | 是 | 中间件地址 |
| `--script` | — | string | — | 是 | 刷机脚本路径 |
| `--ch` | — | string | `all` | 否 | 通道: 1,2,3 或 1-20 或 all |
| `--ip-base` | — | string | `11` | 否 | IP 基数（如 11 表示 192.168.11.x） |
| `--user` | `-u` | string | `admin` | 否 | 中间件用户名 |
| `--pass` | `-p` | string | `admin` | 否 | 中间件密码 |
| `--timeout` | — | int | `600` | 否 | 单台刷机超时(秒) |
| `--retry` | — | int | `1` | 否 | 失败重试次数 |
| `--skip-offline` | — | bool | `true` | 否 | 跳过离线设备 |
| `--dry` | — | bool | `false` | 否 | 模拟运行 |
| `--yes` | `-y` | bool | `false` | 否 | 跳过确认直接执行 |
| `--remote` | — | string | `""` | 否 | 远程 jpy server 地址（COM口在远程时使用） |
| `--jpy` | — | string | `jpy` | 否 | jpy工具路径 |

### COM 口与 IP 映射规则

| COM口 | IP偏移 | 通道1 IP | 通道20 IP |
|-------|--------|----------|-----------|
| COM3 | 0 | 192.168.11.1 | 192.168.11.20 |
| COM4 | 20 | 192.168.11.21 | 192.168.11.40 |
| COM6 | 40 | 192.168.11.41 | 192.168.11.60 |

### 工作流程

1. 检查设备状态（通过 `jpy device list`）
2. 发送 `reboot bootloader`（通过 `jpy device shell`）
3. 切换 COM 通道为 HUB 模式（通过 `jpy com set-mode`）
4. 执行刷机脚本
5. 刷机成功后切换回 OTG 模式

### 日志格式

```
[时间戳] [COM口-通道] [级别] 消息
[20:02:59] [COM4-CH01] [INFO] 检查设备状态...
[20:03:05] [COM4-CH01] [ERROR] 刷机失败: 超时
```

### plain 输出格式

日志输出到 stderr，汇总信息输出到 stderr。

### JSON 输出 schema

```json
{
  "results": [
    {
      "com": "COM4",
      "channel": 1,
      "ip": "11.21",
      "uuid": "abc-123",
      "success": true,
      "error": "",
      "duration": 180000000000
    }
  ],
  "total_time": "5m30s"
}
```

### 示例

```bash
# 刷 COM4 所有通道
jpy flash run --com COM4 --mw 192.168.255.2 --script D:\flash\flash.cmd

# 刷 COM4 的 1-10 通道
jpy flash run --com COM4 --ch 1-10 --mw 192.168.255.2 --script D:\flash\flash.cmd

# 刷 COM3,COM4 的 1,2,3 通道
jpy flash run --com COM3,COM4 --ch 1,2,3 --mw 192.168.255.2 --script D:\flash\flash.cmd

# 模拟运行（不实际执行）
jpy flash run --com COM4 --mw 192.168.255.2 --script D:\flash\flash.cmd --dry

# 跳过确认直接执行
jpy flash run --com COM4 --mw 192.168.255.2 --script D:\flash\flash.cmd -y

# 远程执行（COM口在远程机器上）
jpy flash run --remote 192.168.1.100:9090 --com COM4 --mw 192.168.255.2 --script D:\flash\flash.cmd

# 自定义 IP 基数（如 192.168.12.x）
jpy flash run --com COM4 --mw 192.168.255.2 --script D:\flash\flash.cmd --ip-base 12

# JSON 输出
jpy flash run --com COM4 --ch 1-3 --mw 192.168.255.2 --script D:\flash\flash.cmd -y -o json
```

---

## 命令 14：file push

上传本地文件到远程 jpy server。支持大文件传输（最大 5GB），使用流式上传避免内存溢出。

### 签名

```
jpy file push <local-file> --remote <host:port> [--dest <远程路径>] [--timeout N] [-o plain|json]
```

### 参数

| 参数 | 短写 | 类型 | 默认值 | 必填 | 说明 |
|------|------|------|--------|------|------|
| 位置参数 | — | string | — | 是 | 本地文件路径 |
| `--remote` | — | string | — | 是 | 远程 jpy server 地址 |
| `--dest` | — | string | `""` | 否 | 远程目标路径（默认使用原文件名，放临时目录） |
| `--timeout` | — | int | `1800` | 否 | 上传超时秒数（默认 30 分钟） |

### plain 输出格式

```
OK	D:\flash\rom.zip	1234567890 bytes
```

失败时输出到 stderr：
```
FAIL	上传失败: connection refused
```

### JSON 输出 schema

```json
{"success": true, "path": "D:\\flash\\rom.zip", "size": 1234567890}
```

失败：
```json
{"success": false, "path": "", "size": 0, "error": "上传失败: connection refused"}
```

### 示例

```bash
# 上传到远程默认目录（临时目录）
jpy file push ./rom.zip --remote 192.168.1.100:9090

# 上传到指定路径
jpy file push ./rom.zip --remote 192.168.1.100:9090 --dest D:\flash\rom.zip

# 大文件设置更长超时（1小时）
jpy file push ./large.zip --remote 192.168.1.100:9090 --timeout 3600

# JSON 输出
jpy file push ./rom.zip --remote 192.168.1.100:9090 -o json
```

---

## 命令 15：file pull

让远程 jpy server 从指定 URL 下载文件。

### 签名

```
jpy file pull <url> --remote <host:port> [--dest <远程路径>] [--timeout N] [-o plain|json]
```

### 参数

| 参数 | 短写 | 类型 | 默认值 | 必填 | 说明 |
|------|------|------|--------|------|------|
| 位置参数 | — | string | — | 是 | 下载 URL |
| `--remote` | — | string | — | 是 | 远程 jpy server 地址 |
| `--dest` | — | string | `""` | 否 | 远程保存路径（默认使用 URL 文件名，放临时目录） |
| `--timeout` | — | int | `600` | 否 | 下载超时秒数（默认 10 分钟） |

### plain 输出格式

```
OK	D:\flash\rom.zip	1234567890 bytes
```

### JSON 输出 schema

```json
{"success": true, "path": "D:\\flash\\rom.zip", "size": 1234567890}
```

### 示例

```bash
# 让远程从 URL 下载文件
jpy file pull "https://example.com/rom.zip" --remote 192.168.1.100:9090

# 下载到指定路径
jpy file pull "https://example.com/rom.zip" --remote 192.168.1.100:9090 --dest D:\flash\rom.zip

# 设置超时
jpy file pull "https://example.com/rom.zip" --remote 192.168.1.100:9090 --timeout 1200

# JSON 输出
jpy file pull "https://example.com/rom.zip" --remote 192.168.1.100:9090 -o json
```
