# JPY CLI 使用手册

> 一条命令，管控跨越多台中间件服务器的全部设备；加上筛选条件，精准到单台设备单个机位。

---

## 目录

1. [核心理念：集群控制](#1-核心理念集群控制)
2. [如何自助查看帮助](#2-如何自助查看帮助)
3. [控制粒度：从全局到单点](#3-控制粒度从全局到单点)
4. [命令总览](#4-命令总览)
5. [设备集群控制（核心功能）](#5-设备集群控制核心功能)
   - [通用筛选标志完整参考](#51-通用筛选标志完整参考)
   - [device list — 列出设备](#52-device-list--列出设备)
   - [device status — 服务器统计](#53-device-status--服务器统计)
   - [device reboot — 重启设备](#54-device-reboot--重启设备)
   - [device usb — 切换 USB 模式](#55-device-usb--切换-usb-模式)
   - [device adb — 控制 ADB](#56-device-adb--控制-adb)
   - [device shell — 发送 Shell 命令](#57-device-shell--发送-shell-命令)
   - [device export — 导出设备信息](#58-device-export--导出设备信息)
   - [device log — 查看设备日志](#59-device-log--查看设备日志)
6. [中间件服务器管理](#6-中间件服务器管理)
7. [输出模式 --output](#7-输出模式---output)
8. [实战场景手册](#8-实战场景手册)
9. [其他命令参考](#9-其他命令参考)

---

## 1. 核心理念：集群控制

JPY CLI 的核心设计是**跨多台中间件服务器的集群批量控制**。

```
┌──────────────────────────────────────────────────────────────┐
│  一个活动分组（Group）可以包含多台中间件服务器（Middleware）   │
│  每台服务器管理 N 个设备插槽（Seat）                          │
│                                                              │
│  Group: production                                           │
│  ├── 192.168.1.201:443  → Seat 1~30 → 30 台设备             │
│  ├── 192.168.1.202:443  → Seat 1~30 → 30 台设备             │
│  ├── 192.168.1.203:443  → Seat 1~30 → 30 台设备             │
│  └── ... N 台中间件                                          │
└──────────────────────────────────────────────────────────────┘
```

**不加任何筛选条件** = 对当前活动分组内**所有中间件的所有设备**执行操作：

```bash
# 对分组内所有设备列出状态
jpy middleware device list

# 重启所有设备
jpy middleware device reboot --all
```

**加上筛选条件** = 精准控制到指定范围：

```bash
# 只看某台中间件上的设备
jpy middleware device list --server 192.168.1.201

# 只重启 3 号机位的设备
jpy middleware device reboot --server 192.168.1.201 --seat 3 --all
```

---

## 2. 如何自助查看帮助

**`--help` 是探索命令的第一工具。** 任意命令加 `--help` 即可查看完整的参数列表。

```bash
# 查看顶层命令列表
jpy --help

# 查看 middleware 下有哪些子命令
jpy middleware --help

# 查看 device 下有哪些子命令
jpy middleware device --help

# 查看具体命令的全部参数（最常用）
jpy middleware device list --help
jpy middleware device reboot --help
jpy middleware device shell --help
```

以 `device list` 为例，`--help` 会输出：

```
Flags:
      --all                          跳过交互模式并处理所有匹配设备
      --authorized string[="true"]   筛选授权状态 (true/false)
      --filter-adb string            筛选ADB状态 (true/false)
      --filter-has-ip string         筛选IP存在状态 (true/false)
      --filter-online string         筛选在线状态 (true/false)
      --filter-usb string            筛选USB模式 (true=USB, false=OTG)
      --filter-uuid string           筛选UUID存在状态 (true/false)
  -g, --group string                 目标服务器分组
  -h, --help                         help for list
  -i, --interactive                  交互式选择模式
  -l, --limit int                    限制显示数量 (default 100)
  -o, --output string                输出模式 (tui/plain/json) (default "tui")
      --seat int                     机位号 (default -1)
  -s, --server string                服务器地址匹配模式 (例如: 192.168.1)
  -u, --uuid string                  设备UUID (模糊匹配)
```

> **原则**：遇到不熟悉的命令，先 `--help`，再执行。

---

## 3. 控制粒度：从全局到单点

下面以 `device reboot` 为例，展示从最宽泛到最精准的控制范围：

| 粒度 | 命令示例 | 控制范围 |
|------|----------|----------|
| **全局** | `jpy middleware device reboot --all` | 当前分组所有中间件的所有设备 |
| **按分组** | `jpy middleware device reboot -g production --all` | 指定分组内所有设备 |
| **按服务器网段** | `jpy middleware device reboot -s "192.168.1" --all` | IP 含 192.168.1 的服务器上所有设备 |
| **按单台服务器** | `jpy middleware device reboot -s "192.168.1.201" --all` | 单台中间件的所有设备 |
| **按状态筛选** | `jpy middleware device reboot --filter-online true --all` | 所有在线设备 |
| **按单台设备** | `jpy middleware device reboot -s "192.168.1.201" --seat 5 --all` | 单台中间件的 5 号机位设备 |
| **按 UUID** | `jpy middleware device reboot -u "ABCD1234" --all` | 指定 UUID 设备（精确到一台） |

**同样的粒度控制逻辑适用于所有 `device` 子命令**：`list`、`status`、`reboot`、`usb`、`adb`、`shell`、`export`。

---

## 4. 命令总览

```
jpy
├── middleware                     中间件管理（局域网集群控制）
│   ├── list                       列出当前分组的中间件服务器
│   ├── auth                       认证与服务器管理
│   │   ├── login                  添加/登录中间件服务器
│   │   ├── create                 批量生成并添加服务器配置
│   │   ├── list                   列出已配置的服务器
│   │   ├── select                 切换活动分组
│   │   ├── import                 从 JSON 文件批量导入
│   │   ├── export                 导出当前分组配置
│   │   └── template               生成配置文件模板
│   ├── device                     ★ 设备集群控制（核心）
│   │   ├── list                   列出设备详情（支持多维筛选）
│   │   ├── status                 服务器健康状态与设备统计
│   │   ├── reboot                 批量/单台重启设备
│   │   ├── usb                    批量/单台切换 USB 模式
│   │   ├── adb                    批量/单台控制 ADB 状态
│   │   ├── shell                  向指定设备发送 shell 命令
│   │   ├── export                 导出设备信息到文件
│   │   └── log                    实时查看单个设备日志
│   ├── admin
│   │   ├── auto-auth              自动扫描并授权服务器
│   │   └── update-cluster         批量更新集控平台地址
│   ├── remove                     移除/软删除中间件服务器
│   ├── relogin                    重新连接已软删除的服务器
│   ├── restart                    重启 boxCore 服务
│   └── ssh                        SSH 连接中间件（自动获取密码）
├── cloud                          集控平台远程 API
│   ├── config                     查看/修改集控平台配置
│   │   └── init-configs           创建示例改机配置文件
│   └── stress                     改机压力测试
├── admin middleware
│   ├── generate                   生成授权码
│   ├── list                       列出授权码
│   └── get-root-password          获取 Root 密码
├── config                         本地配置读写
│   ├── list / get / set
├── server ssh / web               后台服务（SSH 跳板机 / Web）
├── proxy                          透明代理
├── tools middleware create        批量生成中间件配置文件
├── tools completion-install       安装 Shell 自动补全
└── log                            查看 CLI 操作日志
```

> 每一层都支持 `--help` 查看子命令列表。

---

## 5. 设备集群控制（核心功能）

### 5.1 通用筛选标志完整参考

以下筛选标志**对所有 `device` 子命令均有效**（`list`、`status`、`reboot`、`usb`、`adb`、`export`）：

| 标志 | 别名 | 类型 | 说明 | 示例 |
|------|------|------|------|------|
| `--group` | `-g` | string | 指定服务器分组名 | `-g production` |
| `--server` | `-s` | string | **中间件服务器**地址模糊匹配（含该字符串） | `-s "192.168.1"` |
| `--uuid` | `-u` | string | 设备 UUID 模糊匹配 | `-u "ABCD1234"` |
| `--ip` | — | string | **目标设备** IP 模糊匹配（中间件内的设备 IP） | `--ip "192.168.10.195"` |
| `--seat` | — | int | 指定机位号（精确） | `--seat 5` |
| `--authorized` | — | bool | 只操作已授权的服务器 | `--authorized` |
| `--filter-online` | — | true/false | 按业务在线状态筛选 | `--filter-online true` |
| `--filter-adb` | — | true/false | 按 ADB 开启状态筛选 | `--filter-adb true` |
| `--filter-usb` | — | true/false | 按 USB 模式筛选（true=USB, false=OTG） | `--filter-usb false` |
| `--filter-has-ip` | — | true/false | 按设备 IP 是否存在筛选 | `--filter-has-ip false` |
| `--filter-uuid` | — | true/false | 按 UUID 是否存在筛选 | `--filter-uuid true` |
| `--all` | — | bool | 对所有匹配结果批量执行（跳过交互确认） | `--all` |
| `--interactive` | `-i` | bool | 进入 TUI 交互式选择模式 | `-i` |
| `--output` | `-o` | string | 输出模式：tui / plain / json | `-o json` |

> **`--server` vs `--ip` 区别**：`--server`（`-s`）匹配**中间件服务器**地址；`--ip` 匹配**目标设备**在中间件内的 IP——两者完全不同，勿混淆。
>
> **`--server` 是模糊匹配**：`-s "192.168"` 会匹配所有 URL 中含 `192.168` 的服务器。
>
> **查看任意命令的筛选标志**：`jpy middleware device <子命令> --help`

---

### 5.2 `device list` — 列出设备

**默认行为**：列出当前活动分组内所有中间件的所有设备（TUI 分页浏览）。

```bash
# 基本用法
jpy middleware device list

# 查看全部可用筛选参数
jpy middleware device list --help
```

**常用示例**：

```bash
# 列出某台中间件的所有设备
jpy middleware device list -s 192.168.1.201

# 列出所有在线且有 UUID 的设备（JSON 格式）
jpy middleware device list --filter-online true --filter-uuid true -o json

# 列出所有无 IP 的设备（用于诊断上线问题）
jpy middleware device list --filter-has-ip false -o plain

# 列出某台中间件 5 号机位的设备
jpy middleware device list -s 192.168.1.201 --seat 5

# 限制显示数量
jpy middleware device list -l 50
```

**JSON 输出结构**：
```json
{
  "total": 120,
  "devices": [
    {
      "server": "192.168.1.201:443",
      "seat": 1,
      "uuid": "ABCD1234",
      "ip": "10.0.0.101",
      "online": true,
      "biz_online": true,
      "usb_mode": "otg",
      "adb": false,
      "model": "Redmi Note 12",
      "android": "13"
    }
  ]
}
```

---

### 5.3 `device status` — 服务器统计

**默认行为**：对所有中间件做一次健康检查，汇总设备统计数据。

```bash
# 查看所有中间件状态
jpy middleware device status

# 查看详细信息（含 SN、集控地址）
jpy middleware device status --detail

# 查看帮助（含高级筛选标志）
jpy middleware device status --help
```

**`status` 专属高级筛选标志**（在通用筛选基础上追加）：

| 标志 | 说明 | 示例 |
|------|------|------|
| `--auth-failed` | 只显示授权失败的服务器 | `--auth-failed` |
| `--fw-has <版本>` | 固件版本含指定字符串 | `--fw-has "v3.2"` |
| `--fw-not <版本>` | 固件版本不含指定字符串 | `--fw-not "v2"` |
| `--speed-gt <N>` | 网络速率 > N Mbps | `--speed-gt 100` |
| `--speed-lt <N>` | 网络速率 < N Mbps | `--speed-lt 10` |
| `--cluster-contains <关键词>` | 集控地址含指定字符串 | `--cluster-contains "10.0.1"` |
| `--cluster-not-contains <关键词>` | 集控地址不含指定字符串 | `--cluster-not-contains "old"` |
| `--ip-count-gt <N>` | 已获取 IP 数 > N | `--ip-count-gt 25` |
| `--ip-count-lt <N>` | 已获取 IP 数 < N | `--ip-count-lt 5` |
| `--biz-online-gt <N>` | 业务在线设备数 > N | `--biz-online-gt 20` |
| `--biz-online-lt <N>` | 业务在线设备数 < N | `--biz-online-lt 10` |
| `--uuid-count-gt <N>` | UUID 数量 > N | `--uuid-count-gt 28` |
| `--uuid-count-lt <N>` | UUID 数量 < N | `--uuid-count-lt 5` |

```bash
# 找出 IP 缺失较多的服务器（超过5个设备没有IP）
jpy middleware device status --ip-count-lt 25 -o json

# 找出业务在线数异常低的服务器
jpy middleware device status --biz-online-lt 10 -o plain
```

---

### 5.4 `device reboot` — 重启设备

**默认行为**：进入 TUI 交互式选择，手动勾选后确认重启。

```bash
# 查看全部可用参数
jpy middleware device reboot --help

# TUI 交互式选择
jpy middleware device reboot

# 批量重启所有设备（跳过确认）
jpy middleware device reboot --all

# 只重启某台服务器上的设备
jpy middleware device reboot -s 192.168.1.201 --all

# 只重启在线设备
jpy middleware device reboot --filter-online true --all

# 精准重启单台：指定服务器 + 指定机位
jpy middleware device reboot -s 192.168.1.201 --seat 5 --all

# JSON 输出（含逐台结果）
jpy middleware device reboot --filter-has-ip false --all -o json
```

**退出码**：`0`=全部成功，`1`=部分失败，`2`=全部失败

---

### 5.5 `device usb` — 切换 USB 模式

```bash
# 查看参数
jpy middleware device usb --help

# 将所有设备切换到 OTG（Host）模式
jpy middleware device usb --mode host --all

# 将指定服务器的所有设备切换到 USB（Device）模式
jpy middleware device usb --mode device -s 192.168.1.201 --all

# 只切换当前处于 USB 模式的设备（切回 OTG）
jpy middleware device usb --mode host --filter-usb true --all

# 只切换已授权服务器上无 IP 且处于 OTG 的设备（设备上线修复）
jpy middleware device usb --mode device --authorized --filter-has-ip false --filter-usb false --all
```

| `--mode` 值 | 含义 |
|-------------|------|
| `host` | OTG 模式（Host，设备挂载到 PC） |
| `device` | USB 模式（Device，设备作为从机） |

---

### 5.6 `device adb` — 控制 ADB

```bash
# 查看参数
jpy middleware device adb --help

# 开启所有设备的 ADB
jpy middleware device adb --set on --all

# 关闭所有在线设备的 ADB（安全加固）
jpy middleware device adb --set off --filter-online true --all

# 只关闭指定服务器上的 ADB
jpy middleware device adb --set off -s 192.168.1.201 --all
```

---

### 5.7 `device shell` — 发送 Shell 命令

向指定设备发送一条 shell 命令并返回执行结果。适用于不能使用 ADB 时，通过中间件下发指令。

> **参数区分**：`--server` 指定**中间件服务器**地址（如 192.168.255.1），`--ip` 指定**目标设备**在中间件内的 IP（如 192.168.10.195），两者含义不同。

```bash
# 查看参数
jpy middleware device shell --help

# 推荐写法：命令作为位置参数（更简洁）
jpy middleware device shell "ls -lh" --server 192.168.255.1 --ip 192.168.10.195

# 让设备重启到 fastboot 模式（刷机前置）
jpy middleware device shell "reboot bootloader" -s 192.168.255.1 --ip 192.168.10.195

# 通过机位号定位（不需要知道设备 IP）
jpy middleware device shell "reboot bootloader" -s 192.168.255.1 --seat 3

# 查询设备型号（JSON 输出）
jpy middleware device shell "getprop ro.product.model" -s 192.168.255.1 --ip 192.168.10.195 -o json

# 也可以用 --command / -c 传入命令（兼容旧写法）
jpy middleware device shell -s 192.168.255.1 --ip 192.168.10.195 -c "getprop ro.product.model"
```

**标志说明**：

| 标志 | 别名 | 必填 | 说明 |
|------|------|------|------|
| `--server` | `-s` | ✅ | **中间件服务器**地址（IP 或 URL 关键词） |
| `--ip` | — | 二选一 | **目标设备** IP（中间件内的设备 IP，非服务器 IP） |
| `--seat` | — | 二选一 | 设备机位号（与 `--ip` 二选一） |
| `[command]` | — | ✅ | 要执行的 shell 命令（位置参数，推荐） |
| `--command` | `-c` | ✅ | 要执行的 shell 命令（与位置参数等效） |
| `--output` | `-o` | — | plain（默认）/ json |

> **执行通道**：工具自动选择最兼容的通道：f=14（设备连接，老/新系统）→ f=289（新系统）→ Terminal（兜底，无返回值）。老系统无需额外配置，自动适配。

**JSON 输出结构**：
```json
{
  "server": "192.168.255.1:443",
  "seat": 3,
  "command": "ls -lh",
  "output": "total 84K\n...",
  "exit_code": 0,
  "success": true
}
```

---

### 5.8 `device export` — 导出设备信息

```bash
# 查看参数
jpy middleware device export --help

# 导出所有设备（默认导出所有字段）
jpy middleware device export output.txt

# 只导出有 UUID 的设备的 UUID 和 IP
jpy middleware device export uuids.txt --export-uuid --export-ip --filter-uuid true

# 智能导出（自动补齐缺失 IP，仅用于 DHCP 配置）
jpy middleware device export dhcp.txt --export-auto

# JSON 格式导出
jpy middleware device export -o json > devices.json
```

**导出字段标志**：

| 标志 | 说明 |
|------|------|
| `--export-id` | 导出设备 ID（由服务器地址生成） |
| `--export-uuid` | 导出设备 UUID（SN） |
| `--export-ip` | 导出设备 IP 地址 |
| `--export-seat` | 导出机位号 |
| `--export-auto` | 智能模式：自动补齐缺失 IP，只含有 UUID 的设备 |

---

### 5.9 `device log` — 查看设备日志

查看**单个设备**的实时日志流。自动完成 USB 切换 → ADB 开启 → 终端连接流程。

```bash
# 必须精确指定一台设备（服务器 + 机位）
jpy middleware device log -s 192.168.1.201 --seat 12

# 通过 UUID 定位
jpy middleware device log -u "ABCD1234"
```

> 按 `Ctrl+C` 退出，CLI 会自动关闭 ADB 并切回 OTG 模式。

---

## 6. 中间件服务器管理

### 分组与服务器

```bash
# 查看有哪些分组，当前活动哪个
jpy middleware auth select -o json

# 切换到指定分组
jpy middleware auth select production

# 查看当前分组的服务器列表
jpy middleware list -o json

# 查看详情（含连接状态）
jpy middleware auth list --details -o plain
```

### 添加服务器

```bash
# 单台添加
jpy middleware auth login "192.168.1.201" -u admin -p admin -g mygroup

# 批量添加（IP 段）
jpy middleware auth create --ip "192.168.1.201-230,192.168.2.101-110" -o json
```

### 维护

```bash
# 软删除所有报错的服务器（不影响设备，可恢复）
jpy middleware remove --has-error --all -o json

# 尝试恢复软删除的服务器（建议循环 3 次）
jpy middleware relogin -o json

# 永久删除指定服务器
jpy middleware remove --search "10.0.0.5" --force --all

# 批量更新集控平台地址
jpy middleware admin update-cluster "10.0.1.100" --authorized

# SSH 连接中间件（自动获取 root 密码）
jpy middleware ssh 192.168.1.201
```

---

## 7. 输出模式 `--output`

大多数命令支持三种输出模式，**Win7 SSH 环境必须使用 plain 或 json**（TUI 会因 raw mode 崩溃）：

| 模式 | 参数 | 说明 | 适用场景 |
|------|------|------|----------|
| TUI | `tui`（默认） | 交互式界面，支持筛选/分页 | 人工操作 |
| Plain | `plain` | 纯文本，Tab 分隔 | Shell 脚本、Win7 SSH |
| JSON | `json` | 结构化 JSON | AI 解析、程序集成 |

```bash
# 查看支持 --output 的命令
jpy middleware device list -o json       # JSON
jpy middleware device status -o plain    # Plain
jpy middleware remove --has-error --all -o json
jpy middleware relogin -o json
jpy middleware auth login "..." -o json
jpy middleware auth select -o json
jpy middleware device shell -s ... -c "..." -o json
```

---

## 8. 实战场景手册

### 场景 A：新服务器批量上线

```bash
# 1. 批量添加服务器
jpy middleware auth create --ip "192.168.1.201-230" -o json

# 2. 触发状态扫描（自动登录，查看健康状态）
jpy middleware device status -o json

# 3. 软删除连接失败的服务器
jpy middleware remove --has-error --all -o json

# 4. 尝试恢复（重试 3 次）
jpy middleware relogin -o json
jpy middleware relogin -o json
jpy middleware relogin -o json
```

### 场景 B：设备初次上线 IP 修复（USB 循环法）

```bash
# 1. 查看当前状态
jpy middleware device status -o json

# 2. 将无 IP + OTG 的设备切换到 USB 模式
jpy middleware device usb --mode device --authorized --filter-has-ip false --filter-usb false --all -o json

# 3. 等待 15 秒后切回 OTG
jpy middleware device usb --mode host --authorized --filter-has-ip false --filter-usb true --all -o json

# 4. 检查 IP 获取情况，重复步骤 2-3 直到 IP 数量不再增长（最多 3 轮）
jpy middleware device status -o json

# 5. 把所有还在 USB 模式的设备切回 OTG
jpy middleware device usb --mode host --authorized --filter-usb true --all -o json

# 6. 若仍有无 IP 设备，强制重启后再来一轮
jpy middleware device reboot --filter-has-ip false --all -o json
```

### 场景 C：批量刷机前准备（重启到 fastboot）

```bash
# 1. 查看目标服务器的设备列表，确认目标设备 IP
jpy middleware device list -s 192.168.255.1 --filter-online true -o json

# 2. 通过 IP 让设备进入 fastboot（命令作位置参数，更简洁）
jpy middleware device shell "reboot bootloader" -s 192.168.255.1 --ip 192.168.10.195

# 3. 或批量通过机位号操作
jpy middleware device shell "reboot bootloader" -s 192.168.255.1 --seat 1
jpy middleware device shell "reboot bootloader" -s 192.168.255.1 --seat 2
```

### 场景 D：AI/自动化远程管控（JSON 模式）

```bash
# 获取所有服务器状态（供 AI 解析）
jpy middleware device status -o json

# 获取设备列表
jpy middleware device list --filter-online true -o json

# 对无 IP 设备执行操作，获取结果
jpy middleware device reboot --filter-has-ip false --all -o json
# 退出码: 0=全部成功, 1=部分失败, 2=全部失败

# 查询指定设备属性
jpy middleware device shell "getprop ro.build.version.release" -s 192.168.255.1 --ip 192.168.10.195 -o json
```

### 场景 E：设备导出与 DHCP 配置

```bash
# 导出所有有 UUID 的设备 UUID 和 IP
jpy middleware device export dhcp_list.txt --export-uuid --export-ip --filter-uuid true

# 智能导出（自动补齐 IP，适合 DHCP 服务器配置）
jpy middleware device export dhcp_auto.txt --export-auto -s "192.168.1"
```

### 场景 F：安全维护（关闭 ADB）

```bash
# 关闭所有在线设备的 ADB
jpy middleware device adb --set off --filter-online true --all -o json

# 验证
jpy middleware device list --filter-adb true -o plain
```

### 场景 G：排障抽查

```bash
# 列出无 IP 的设备
jpy middleware device list --filter-has-ip false -o plain

# 抽查其中一台设备的日志
jpy middleware device log -s 192.168.10.206 --seat 12

# 查看某台服务器的详细状态
jpy middleware device status -s 192.168.1.201 --detail -o json
```

---

## 9. 其他命令参考

### 本地配置

```bash
jpy config list                    # 列出所有配置
jpy config get max_concurrency     # 获取配置项
jpy config set max_concurrency 20  # 增大并发数（加快批量操作）
jpy config set connect_timeout 5   # 设置连接超时（秒）
```

### 集控平台 Cloud

```bash
jpy cloud config -o json           # 查看集控平台配置
jpy cloud config init-configs      # 创建改机配置示例
jpy cloud stress -o json           # 改机压力测试
```

### 系统管理

```bash
jpy admin middleware generate      # 生成授权码
jpy admin middleware list          # 列出授权码
```

### 日志

```bash
jpy log                            # 查看 CLI 操作日志
jpy log -f                         # 实时跟踪日志
jpy log -n 50 --grep "ERROR"       # 过滤错误日志
```

### 工具

```bash
jpy proxy -p 8888                  # 启动透明代理
jpy server ssh                     # 启动 SSH 跳板机
jpy tools completion-install       # 安装 Shell 自动补全（Tab 键提示）
```

---

## 构建参考

```bash
make build        # 开发编译（当前平台）
make dist         # 跨平台发布（Linux/macOS/Windows amd64+arm64）
make dist-win7    # Win7 兼容版（Go 1.19，不含 cloud 模块）
```

默认配置目录：`~/.jpy/`
日志文件：`~/.jpy/logs/jpy.log`
