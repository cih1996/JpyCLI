# JPY CLI 命令参考手册

JPY 中间件管理命令行工具，支持局域网中间件设备管控和集控平台远程 API 操作。

## 命令总览

```
jpy
├── middleware                # 中间件管理（局域网）
│   ├── list                  # 列出当前分组的服务器列表
│   ├── auth                  # 认证和服务器管理
│   │   ├── login             # 登录服务器
│   │   ├── create            # 批量生成服务器配置
│   │   ├── list              # 列出已配置服务器
│   │   ├── select            # 选择/切换活动分组
│   │   ├── import            # 从 JSON 文件导入配置
│   │   ├── export            # 导出当前分组配置
│   │   └── template          # 生成配置模板
│   ├── device                # 设备管理
│   │   ├── list              # 列出设备详情
│   │   ├── status            # 服务器状态与设备统计
│   │   ├── export            # 导出设备信息到文件
│   │   ├── reboot            # 重启设备（电源循环）
│   │   ├── usb               # 切换 USB 模式
│   │   ├── adb               # 控制 ADB 状态
│   │   └── log               # 查看单个设备日志
│   ├── admin                 # 管理员命令
│   │   ├── auto-auth         # 自动扫描并授权服务器
│   │   └── update-cluster    # 批量更新集控平台地址
│   ├── remove                # 移除/软删除服务器
│   ├── relogin               # 重新连接已软删除服务器
│   ├── restart               # 重启 boxCore 服务
│   └── ssh                   # SSH 连接中间件（自动获取密码）
├── cloud                     # 集控平台远程 API
│   ├── config                # 查看/修改集控平台配置
│   │   └── init-configs      # 创建示例改机配置文件
│   └── stress                # 改机压力测试
├── admin                     # 系统管理命令
│   └── middleware
│       ├── generate          # 生成授权码
│       ├── list              # 列出授权码
│       └── get-root-password # 获取 Root 密码
├── config                    # 本地配置管理
│   ├── list                  # 列出所有配置
│   ├── get                   # 获取配置项
│   └── set                   # 设置配置项
├── server                    # 后台服务
│   ├── ssh                   # 启动 SSH 跳板机
│   └── web                   # 启动 Web 服务
├── proxy                     # 透明代理
├── tools                     # 工具集
│   ├── middleware create     # 批量生成中间件配置
│   └── completion-install    # 安装 Shell 补全
└── log                       # 查看 CLI 操作日志
```

---

## 全局标志

| 标志 | 类型 | 说明 |
|------|------|------|
| `--debug` | bool | 启用调试日志（同时输出到控制台） |
| `--log-level` | string | 日志级别：`debug`/`info`/`warn`/`error` |

---

## 输出模式 `--output` / `-o`

大部分命令支持三种输出模式：

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| `tui` | 默认，交互式 TUI 界面 | 人工操作 |
| `plain` | 纯文本 Tab 分隔 | 脚本解析、Win7 SSH 环境 |
| `json` | 结构化 JSON | AI/程序解析 |

### 支持 `--output` 的命令一览

| 命令 | tui | plain | json |
|------|-----|-------|------|
| `middleware list` | ✅ | ✅ | ✅ |
| `middleware remove` | ✅ | ✅ | ✅ |
| `middleware relogin` | ✅ | ✅ | ✅ |
| `middleware auth login` | ✅ | ✅ | ✅ |
| `middleware auth create` | ✅ | ✅ | ✅ |
| `middleware auth list` | ✅ | ✅ | ✅ |
| `middleware auth select` | ✅ | ✅ | ✅ |
| `middleware device list` | ✅ | ✅ | ✅ |
| `middleware device status` | ✅ | ✅ | ✅ |
| `middleware device export` | ✅ | ✅ | ✅ |
| `middleware device reboot` | ✅ | ✅ | ✅ |
| `middleware device usb` | ✅ | ✅ | ✅ |
| `middleware device adb` | ✅ | ✅ | ✅ |
| `cloud config` | - | ✅ | ✅ |
| `cloud stress` | ✅ | ✅ | ✅ |

> **Win7 兼容**：Win7 SSH 环境不支持 TUI raw mode，使用 `-o plain` 或 `-o json` 避免崩溃。

---

## 中间件管理 (`middleware`)

### `middleware list` — 列出服务器

```bash
# TUI 交互式浏览
jpy middleware list

# JSON 输出（AI 解析）
jpy middleware list -o json

# Plain 纯文本（脚本）
jpy middleware list -o plain
```

**JSON 结构**:
```json
{
  "group": "default",
  "total": 3,
  "servers": [
    { "url": "192.168.1.201:443", "username": "admin", "status": "ok" }
  ]
}
```

---

### `middleware auth` — 认证与分组管理

#### `auth login` — 登录服务器

```bash
jpy middleware auth login <url> -u <用户名> -p <密码> [-g <分组>] [-o json]
```

| 标志 | 别名 | 默认值 | 说明 |
|------|------|--------|------|
| `--username` | `-u` | — | 用户名（必填） |
| `--password` | `-p` | — | 密码（必填） |
| `--group` | `-g` | `default` | 目标分组 |
| `--output` | `-o` | `tui` | 输出模式 |

```bash
# 示例
jpy middleware auth login "192.168.1.100:443" -u admin -p admin -g production -o json
```

**JSON 输出**:
```json
{ "url": "192.168.1.100:443", "group": "production", "success": true }
```

**Plain 输出**: `OK\t192.168.1.100:443\tproduction`

---

#### `auth create` — 批量生成服务器配置

```bash
jpy middleware auth create --ip <IP范围> [-P <端口>] [-u <用户名>] [-p <密码>] [-o json]
```

| 标志 | 别名 | 默认值 | 说明 |
|------|------|--------|------|
| `--ip` | `-i` | — | IP 范围，逗号分隔（如 `192.168.1.201-210,192.168.2.100`） |
| `--port` | `-P` | `443` | 端口 |
| `--username` | `-u` | `admin` | 用户名 |
| `--password` | `-p` | `admin` | 密码 |
| `--output` | `-o` | `tui` | 输出模式 |

> **注意**: `-o json` 或 `-o plain` 模式必须提供 `--ip` 参数。未提供 `--ip` 时进入交互模式。

```bash
# 批量添加
jpy middleware auth create --ip "192.168.1.201-220,192.168.2.101-110" -o json
```

**JSON 输出**:
```json
{ "group": "default", "added": 30, "duplicate": 0, "urls": ["192.168.1.201:443", ...] }
```

---

#### `auth list` — 列出已配置服务器

```bash
jpy middleware auth list [--details] [-o json]
```

| 标志 | 说明 |
|------|------|
| `--details` | 显示详细信息（包含 Token 状态等） |
| `--output` / `-o` | 输出模式 |

---

#### `auth select` — 选择/切换活动分组

```bash
# 查看所有分组
jpy middleware auth select -o json

# 切换分组
jpy middleware auth select <分组名> -o json
```

**JSON 输出（查看）**:
```json
{
  "active_group": "default",
  "groups": [
    { "name": "default", "count": 10, "active": true },
    { "name": "production", "count": 25, "active": false }
  ]
}
```

---

#### `auth import` — 导入服务器配置

```bash
jpy middleware auth import <文件路径>
```

#### `auth export` — 导出服务器配置

```bash
jpy middleware auth export [-o <输出文件>]
```

> 注意：`auth export` 的 `-o` 是输出文件路径，不是输出模式。

#### `auth template` — 生成配置模板

```bash
jpy middleware auth template
```

---

### `middleware remove` — 移除服务器

```bash
jpy middleware remove [--search <关键词>] [--has-error] [--force] [--all] [-o json]
```

| 标志 | 说明 |
|------|------|
| `--search` | 匹配服务器 URL 的关键词 |
| `--has-error` | 仅匹配连接错误的服务器 |
| `--force` | 硬删除（永久移除，非软删除） |
| `--all` | 对所有匹配项执行，跳过确认 |
| `--output` / `-o` | 输出模式 |

> **非交互模式**（`-o json/plain`）必须提供 `--all`、`--has-error` 或 `--search` 之一。

```bash
# 软删除所有报错服务器
jpy middleware remove --has-error --all -o json

# 硬删除指定服务器
jpy middleware remove --search "10.0.0.5" --force --all -o json
```

---

### `middleware relogin` — 重新连接已软删除服务器

```bash
jpy middleware relogin [-o json]
```

**JSON 输出**:
```json
{
  "total": 5,
  "restored": 3,
  "failed": 2,
  "results": [
    { "url": "192.168.1.201:443", "status": "restored" },
    { "url": "192.168.1.202:443", "status": "failed", "error": "connection refused" }
  ]
}
```

---

### `middleware ssh` — SSH 连接中间件

```bash
jpy middleware ssh <IP>
```

自动获取 Root 密码，生成 SSH 连接命令。若系统安装了 `sshpass` 则直接生成可执行命令。

---

### `middleware restart` — 重启 boxCore 服务

```bash
jpy middleware restart [通用筛选标志]
```

支持通用设备筛选标志（`-g`、`-s`、`-u`、`--seat` 等）。

---

## 中间件设备管理 (`middleware device`)

### 通用筛选标志

所有设备子命令共享以下标志：

| 标志 | 别名 | 说明 | 示例 |
|------|------|------|------|
| `--group` | `-g` | 服务器分组名 | `-g production` |
| `--server` | `-s` | 服务器地址模糊匹配 | `-s "192.168.1"` |
| `--uuid` | `-u` | 设备 UUID 模糊匹配 | `-u "f73a9c"` |
| `--seat` | — | 指定机位号 | `--seat 5` |
| `--authorized` | — | 仅已授权服务器 | `--authorized` |
| `--filter-online` | — | 按在线状态筛选 | `--filter-online true` |
| `--filter-adb` | — | 按 ADB 状态筛选 | `--filter-adb true` |
| `--filter-usb` | — | 按 USB 模式筛选（true=USB, false=OTG） | `--filter-usb false` |
| `--filter-has-ip` | — | 按 IP 存在状态筛选 | `--filter-has-ip false` |
| `--filter-uuid` | — | 按 UUID 存在状态筛选 | `--filter-uuid true` |
| `--interactive` | `-i` | 交互式选择模式 | `-i` |
| `--all` | — | 对所有匹配设备执行 | `--all` |
| `--output` | `-o` | 输出模式（tui/plain/json） | `-o json` |

---

### `device list` — 列出设备详情

```bash
jpy middleware device list [筛选标志] [-o json]
```

**JSON 输出**:
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

**Plain 输出**: `SERVER\tSEAT\tUUID\tMODEL\tANDROID\tONLINE\tBIZ\tIP\tADB\tUSB`

---

### `device status` — 服务器状态统计

```bash
jpy middleware device status [筛选标志] [--detail] [-o json]
```

**专属高级筛选标志**:

| 标志 | 说明 |
|------|------|
| `--detail` | 显示 SN、集控地址、授权名称 |
| `--auth-failed` | 筛选授权失败服务器 |
| `--fw-has` / `--fw-not` | 按固件版本筛选 |
| `--speed-gt` / `--speed-lt` | 按网络速率筛选（Mbps） |
| `--cluster-contains` / `--cluster-not-contains` | 按集控地址筛选 |
| `--sn-gt` / `--sn-lt` | 按 SN 筛选（字符串比较） |
| `--ip-count-gt` / `--ip-count-lt` | 按 IP 数量筛选 |
| `--biz-online-gt` / `--biz-online-lt` | 按业务在线数筛选 |
| `--uuid-count-gt` / `--uuid-count-lt` | 按 UUID 数量筛选 |

**JSON 输出**:
```json
{
  "summary": {
    "total_servers": 10,
    "online_servers": 9,
    "total_devices": 300,
    "biz_online": 280,
    "ip_count": 295,
    "uuid_count": 298,
    "adb_count": 0,
    "usb_count": 5,
    "otg_count": 295
  },
  "servers": [
    {
      "address": "192.168.1.201:443",
      "status": "Online",
      "authorized": true,
      "license_status": "成功",
      "sn": "SN12345",
      "device_count": 30,
      "biz_online_count": 28,
      "ip_count": 29,
      "uuid_count": 30,
      "adb_count": 0,
      "usb_count": 0,
      "otg_count": 30
    }
  ]
}
```

---

### `device export` — 导出设备信息

```bash
jpy middleware device export [文件名] [筛选标志] [导出标志] [-o json]
```

**导出标志**:
| 标志 | 说明 |
|------|------|
| `--export-id` | 导出设备 ID |
| `--export-ip` | 导出设备 IP |
| `--export-uuid` | 导出设备 UUID |
| `--export-seat` | 导出机位号 |
| `--export-auto` | 智能导出（自动补齐缺失 IP，仅含有 UUID 的设备） |

> 不指定导出标志时默认导出全部字段：`ID\tUUID\tIP\tSeat`

---

### `device reboot` — 重启设备

```bash
jpy middleware device reboot [筛选标志] [--all] [-o json]
```

**JSON 输出**:
```json
{
  "total": 5,
  "success": 4,
  "failed": 1,
  "results": [
    { "server": "192.168.1.201:443", "seat": 1, "status": "ok" },
    { "server": "192.168.1.201:443", "seat": 3, "status": "failed", "error": "timeout" }
  ]
}
```

退出码：0=全部成功，1=部分失败，2=全部失败

---

### `device usb` — 切换 USB 模式

```bash
jpy middleware device usb --mode <host|device> [筛选标志] [--all] [-o json]
```

| 标志 | 别名 | 说明 |
|------|------|------|
| `--mode` | `-m` | 目标模式：`host`(OTG) / `device`(USB) |

---

### `device adb` — 控制 ADB 状态

```bash
jpy middleware device adb --set <on|off> [筛选标志] [--all] [-o json]
```

| 标志 | 说明 |
|------|------|
| `--set` | 目标状态：`on` / `off` |

---

### `device log` — 查看设备日志

```bash
jpy middleware device log -s <服务器IP> --seat <机位号>
```

> 必须指定单个设备。自动完成 USB 切换 → ADB 开启 → Shell 连接 → Tail 日志全流程。

---

## 中间件管理员 (`middleware admin`)

### `admin auto-auth` — 自动授权

```bash
jpy middleware admin auto-auth
```

自动扫描并授权待处理的中间件服务器。

### `admin update-cluster` — 更新集控平台地址

```bash
jpy middleware admin update-cluster <新地址> [--server <匹配>] [--group <分组>] [--authorized] [--force]
```

| 标志 | 说明 |
|------|------|
| `--server` | 服务器地址匹配（正则） |
| `--group` | 指定分组 |
| `--authorized` | 按授权状态筛选 |
| `--force` | 即使地址一致也重新提交 |

---

## 集控平台远程 API (`cloud`)

### `cloud config` — 配置管理

```bash
# 查看当前配置
jpy cloud config -o json

# 创建示例改机配置文件
jpy cloud config init-configs
```

### `cloud stress` — 改机压力测试

```bash
jpy cloud stress [--config <配置文件>] [-o json]
```

---

## 系统管理 (`admin`)

```bash
# 生成授权码
jpy admin middleware generate

# 列出授权码
jpy admin middleware list

# 获取中间件 Root 密码
jpy admin middleware get-root-password
```

---

## 本地配置 (`config`)

```bash
# 列出所有配置
jpy config list

# 获取配置项
jpy config get <key>

# 设置配置项
jpy config set <key> <value>
```

常用配置项：
| 配置键 | 默认值 | 说明 |
|--------|--------|------|
| `log_level` | `info` | 日志级别 |
| `log_output` | `file` | 日志输出：`console`/`file`/`both` |
| `max_concurrency` | `5` | 最大并发数 |
| `connect_timeout` | `3` | 连接超时（秒） |

---

## 其他命令

### `proxy` — 透明代理

```bash
jpy proxy [-p <端口>]
```

### `server ssh` — SSH 跳板机

```bash
jpy server ssh
```

### `server web` — Web 服务

```bash
jpy server web
```

### `tools completion-install` — 安装补全

```bash
jpy tools completion-install
```

### `log` — 查看 CLI 日志

```bash
jpy log [-f] [-n <行数>] [--grep <过滤词>]
```

---

## 实战场景

### 场景一：AI 远程批量操控（JSON 模式）

```bash
# 1. 查看服务器列表
jpy middleware list -o json

# 2. 查看设备状态
jpy middleware device status -o json

# 3. 批量重启无 IP 设备
jpy middleware device reboot --filter-has-ip false --all -o json

# 4. 查看执行结果
jpy middleware device status --authorized -o json
```

### 场景二：Win7 SSH 环境操作（Plain 模式）

```bash
# TUI 在 Win7 会崩溃，必须使用 plain 或 json
jpy middleware device status -o plain
jpy middleware device list -o plain
jpy middleware remove --has-error --all -o plain
```

### 场景三：设备初次上线（循环修复 IP）

```bash
# 1. 无 IP + OTG 的设备 → 切换到 USB
jpy middleware device usb --mode device --authorized --filter-has-ip false --filter-usb false --all -o json

# 2. 切回 OTG
jpy middleware device usb --mode host --authorized --filter-has-ip false --filter-usb true --all -o json

# 3. 检查统计
jpy middleware device status -o json

# 4. 所有 USB 切回 OTG
jpy middleware device usb --mode host --authorized --filter-usb true --all -o json

# 5. 循环 1-3，连续 3 次 IP 数量无增长则停止
# 6. 强制重启仍无 IP 的设备
jpy middleware device reboot --filter-has-ip false --all -o json
```

### 场景四：新服务器上线优化

```bash
# 1. 批量添加服务器
jpy middleware auth create --ip "192.168.1.201-230" -o json

# 2. 触发状态扫描（自动登录）
jpy middleware device status -o json

# 3. 隔离登录失败的服务器
jpy middleware remove --has-error --all -o json

# 4. 尝试恢复（建议循环 3 次）
jpy middleware relogin -o json
```

### 场景五：设备信息导出

```bash
# 导出所有在线设备 ID 和 UUID
jpy middleware device export devices.txt --export-id --export-uuid --filter-online true

# 智能导出（自动补齐 IP）
jpy middleware device export dhcp.txt --export-auto -s "192.168.1"

# JSON 格式导出
jpy middleware device export -o json > devices.json
```

### 场景六：服务器分组管理

```bash
# 查看分组列表
jpy middleware auth select -o json

# 切换分组
jpy middleware auth select production -o json

# 添加服务器到当前分组
jpy middleware auth login "192.168.0.102" -u admin -p admin -o json
```

### 场景七：排障抽查

```bash
# 列出无 IP 设备
jpy middleware device list --filter-has-ip false -o plain

# 抽查一台设备日志
jpy middleware device log -s 192.168.10.206 --seat 12
```

---

## 构建与发布

```bash
# 开发编译
make build

# 跨平台发布（Linux/macOS/Windows amd64+arm64）
make dist

# Win7 兼容编译（Go 1.19，排除 cloud 模块）
make dist-win7
```

配置文件位置：`~/.jpy/config.yaml`
服务器数据：`~/.jpy/servers.json`
日志文件：`~/.jpy/logs/jpy.log`
