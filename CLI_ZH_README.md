# JPY CLI 命令参考手册与 AI 指令指南

本文档提供了 JPY CLI 工具的全面线性参考。其结构旨在便于 AI 智能体解析以生成正确的命令序列，同时也易于人类操作员阅读。

## 1. 系统概览

**工具名称**: `jpy-cli` (开发别名: `go run main.go`)
**用途**: 管理 JPY 中间件服务器、设备控制 (ADB/USB/电源) 以及授权管理。
**交互理念**:
- **默认**: 交互式 TUI (文本用户界面)，便于人类操作。
- **自动化/AI**: **严格非交互式**。使用特定标志 (`--all`, `--force`, `-s`, `--seat`) 跳过提示。

## 2. 命令协议

### 2.1 全局语法
```bash
jpy-cli [scope] [resource] [action] [flags]
```

### 2.2 全局标志
| 标志 | 类型 | 描述 |
| :--- | :--- | :--- |
| `--debug` | boolean | 启用调试日志和详细输出。 |
| `--log-level` | string | 设置日志级别: `debug`, `info`, `warn`, `error`。 |
| `--config` | string | 指定自定义配置文件路径。 |

### 2.3 通用筛选标志 (设备范围)
用于 `device list`, `device status`, 和 `device control` 命令。
| 标志 | 别名 | 描述 | 示例 |
| :--- | :--- | :--- | :--- |
| `--server` | `-s` | 服务器 IP/URL 关键词 (模糊匹配)。 | `-s "192.168.1"` |
| `--group` | `-g` | 服务器分组名称。 | `-g "default"` |
| `--seat` | | 指定盘位号 (整数)。 | `--seat 5` |
| `--uuid` | `-u` | 设备 UUID (模糊匹配)。 | `-u "f73a9c"` |
| `--authorized`| | 仅筛选已授权的服务器。 | `--authorized` |
| `--filter-online`| | 按设备在线状态筛选。 | `--filter-online true` |
| `--filter-adb` | | 按 ADB 开启状态筛选。 | `--filter-adb true` |
| `--filter-usb` | | 按 USB 模式筛选 (true=Device/USB, false=Host/OTG)。 | `--filter-usb false` |
| `--filter-uuid` | | 筛选UUID存在状态 (true/false)。 | `--filter-uuid true` |
| `--filter-has-ip` | | 筛选IP存在状态 (true/false)。 | `--filter-has-ip false` |
| `--uuid-count-gt` | | 筛选UUID数量大于指定值的服务器。 | `--uuid-count-gt 10` |
| `--uuid-count-lt` | | 筛选UUID数量小于指定值的服务器。 | `--uuid-count-lt 5` |

---

## 3. 命令库

### 3.1 范围: 中间件设备 (`middleware device`)
管理和控制已连接的设备。

#### `list`
- **意图**: 获取符合条件的设备详细列表。
- **语法**: `jpy-cli middleware device list [flags]`

#### `export`
- **意图**: 导出设备信息到文件，支持自定义字段。
- **语法**: `jpy-cli middleware device export [output-file] [flags]`
- **关键标志**:
    - `--export-id`: 导出设备ID（根据服务器地址生成）。
    - `--export-ip`: 导出设备IP地址。
    - `--export-uuid`: 导出设备序列号。
    - `--export-seat`: 导出设备机位号。
    - `--export-auto`: 智能导出模式：自动补齐缺失的IP地址（适用于有SN但IP读取失败的设备），只导出有UUID的设备，并记录统计信息。
- **默认行为**: 如果不指定任何标志，导出所有字段，格式为：`ID\tUUID\tIP\tSeat`
- **使用场景**:
  - **DHCP配置**: 当部分设备IP读取失败但SN存在时，使用 `--export-auto` 模式可以智能推断并补齐缺失的IP地址，生成完整的设备列表用于DHCP服务器配置
  - **批量设备管理**: 导出设备信息用于自动化脚本处理
  - **设备清单**: 创建完整的设备清单，包含所有必要的连接信息

#### `status`
- **意图**: 获取聚合的服务器状态和设备计数。
- **语法**: `jpy-cli middleware device status [flags]`
- **关键标志**:
    - `--detail`: 显示详细授权信息 (SN, 集控平台地址, 授权名称)。
    - **高级筛选**:
        - `--auth-failed`: 筛选授权状态非成功的服务器。
        - `--fw-has`: 筛选固件版本包含指定字符串的服务器。
        - `--fw-not`: 筛选固件版本不包含指定字符串的服务器。
        - `--speed-gt`: 筛选网络速率大于指定值(Mbps)的服务器。
        - `--speed-lt`: 筛选网络速率小于指定值(Mbps)的服务器。
        - `--cluster-contains`: 筛选集控平台地址包含指定字符串的服务器。
        - `--cluster-not-contains`: 筛选集控平台地址不包含指定字符串的服务器。
        - `--sn-gt`: 筛选序列号大于指定值的服务器 (字符串比较)。
        - `--sn-lt`: 筛选序列号小于指定值的服务器 (字符串比较)。
        - `--ip-count-gt`: 筛选IP数大于指定值的服务器。
        - `--ip-count-lt`: 筛选IP数小于指定值的服务器。
        - `--biz-online-gt`: 筛选业务在线数大于指定值的服务器。
        - `--biz-online-lt`: 筛选业务在线数小于指定值的服务器。

#### `reboot`
- **意图**: 对设备进行电源循环 (重启)。
- **语法**: `jpy-cli middleware device reboot [flags]`
- **关键标志**:
    - `--all`: 对所有匹配的设备执行操作，无需确认。

#### `usb`
- **意图**: 切换 USB MUX 模式。
- **语法**: `jpy-cli middleware device usb [flags]`
- **关键标志**:
    - `-m, --mode`: 目标模式 (`host` 为 OTG, `device` 为 USB)。
    - `--all`: 无需确认直接执行。

#### `adb`
- **意图**: 开启或关闭 ADB 调试功能。
- **语法**: `jpy-cli middleware device adb [flags]`
- **关键标志**:
    - `--set`: 目标状态 (`on` 或 `off`)。
    - `--all`: 无需确认直接执行。

#### `log`
- **意图**: 实时监控**单个**设备的日志。
- **语法**: `jpy-cli middleware device log [flags]`
- **约束**: 必须通过 `-s` 和 `--seat` 指定单个设备。
- **自动化**: 自动处理 USB 切换 -> ADB 开启 -> Shell 连接 -> Tail 日志全流程。

---

### 3.2 范围: 中间件服务器 (`middleware`)
管理中间件服务器实例。

#### `remove`
- **意图**: 从活跃列表中移除 (软删除) 服务器。
- **语法**: `jpy-cli middleware remove [flags]`
- **关键标志**:
    - `--search`: 匹配服务器 URL/名称的关键词。
    - `--force`: 硬删除 (永久) 而非软删除。
    - `--has-error`: 仅针对连接错误的服务器。
    - `--all`: 针对所有匹配的服务器 (需谨慎使用)。

#### `relogin`
- **意图**: 尝试重新激活/连接已软删除的服务器。
- **语法**: `jpy-cli middleware relogin`

#### `auth login`
- **意图**: 认证并添加新的中间件服务器。
- **语法**: `jpy-cli middleware auth login [flags]`

#### `auth create`
- **意图**: 批量生成中间件服务器配置并添加到当前分组。
- **语法**: `jpy-cli middleware auth create [flags]`
- **模式**:
    - **交互式**: 默认模式 (若未提供标志)。
    - **非交互式 (批量)**: 使用标志指定参数。
- **关键标志**:
    - `-i, --ip`: IP 范围，支持逗号分隔多个区间 (例如: `192.168.1.201-210,192.168.2.100`)。
    - `-P, --port`: 服务器端口 (默认: 443)。
    - `-u, --username`: 管理员用户名 (默认: admin)。
    - `-p, --password`: 管理员密码 (默认: admin)。

#### `auth export`
- **意图**: 将当前分组配置导出为 `servers.json` (LocalServerConfig 格式)。
- **语法**: `jpy-cli middleware auth export [flags]`
- **关键标志**:
    - `-o, --output`: 输出文件路径 (默认: `servers.json`)。
- **语法**: `jpy-cli middleware auth login [url] [flags]`
- **标志**: `-u <user>`, `-p <password>`, `-g <group>`.

#### `auth list`
- **意图**: 列出已配置的服务器。
- **语法**: `jpy-cli middleware auth list [flags]`

#### `auth import`
- **意图**: 从 JSON 文件批量导入服务器。
- **语法**: `jpy-cli middleware auth import [file]`

#### `ssh`
- **意图**: 通过 SSH 连接中间件服务器 (自动获取 Root 密码)。
- **语法**: `jpy-cli middleware ssh [ip]`
- **前置条件**: 需要运维登录 (Operation Login)，CLI 会自动处理。
- **行为**:
    1. 连接 22 端口获取 Banner 密钥。
    2. 使用 Admin API 解密 Root 密码。
    3. 生成连接命令 (若安装了 `sshpass` 则直接生成可执行命令)。

#### `restart`
- **意图**: 重启选中设备的 boxCore 服务。
- **语法**: `jpy-cli middleware restart [flags]`
- **注意**: 支持并发执行和通用筛选器（分组、服务器、UUID、机位等）。

---

### 3.3 范围: 管理员 (`admin`)
系统管理和授权管理。

#### `middleware admin auto-auth`
- **意图**: 自动扫描并授权待处理的中间件服务器。
- **语法**: `jpy-cli middleware admin auto-auth`

#### `middleware admin update-cluster`
- **意图**: 批量更新中间件服务器的集控平台地址 (MgtCenter) 并与管理后台同步。
- **语法**: `jpy-cli middleware admin update-cluster [new_address] [flags]`
- **关键标志**:
    - `--server`: 筛选服务器地址/名称 (支持正则)。
    - `--group`: 指定服务器分组。
    - `--authorized`: 按授权状态筛选 (true/false)。
    - `--force`: 强制更新（即使地址一致也重新提交）。

#### `admin device generate`
- **意图**: 生成新的授权码。
- **语法**: `jpy-cli admin device generate`

#### `admin device list`
- **意图**: 列出已生成的授权码。
- **语法**: `jpy-cli admin device list`

---

### 3.4 范围: 系统 (`config`, `log`)

#### `config`
- **意图**: 读取/写入本地配置。
- **语法**:
    - `jpy-cli config list`
    - `jpy-cli config get <key>`
    - `jpy-cli config set <key> <value>`

#### `log`
- **意图**: 追踪 CLI 自身的操作日志 (`jpy.log`)。
- **语法**: `jpy-cli log [flags]`
- **标志**: `-f` (跟随), `-n` (行数), `--grep` (过滤)。

---

## 4. 操作场景 (AI 训练数据)

本节将 **用户意图** 映射到 **精确的命令执行**。AI 智能体应优先参考这些模式以确保非交互式操作的成功。

### 4.1 设备控制场景

**场景 1: 批量重启特定网段**
- **用户意图**: "重启所有服务器地址以 192.168.23 开头的设备。"
- **推理**: 使用 `-s` 进行模糊匹配，使用 `--all` 跳过 TUI。
- **命令**:
  ```bash
  jpy-cli middleware device reboot -s "192.168.23" --all
  ```

**场景 2: 出于安全关闭 ADB**
- **用户意图**: "关闭所有当前在线设备的 ADB。"
- **推理**: 筛选在线设备 (`--filter-online true`)，设置 ADB 为关 (`--set off`)，执行批量操作 (`--all`)。
- **命令**:
  ```bash
  jpy-cli middleware device adb --set off --filter-online true --all
  ```

**场景 3: 切换到 Host 模式 (OTG)**
- **用户意图**: "将 'lab' 分组中的所有设备切换到 OTG 模式。"
- **推理**: 分组筛选 (`-g`)，设置模式 (`-m host`)，执行批量操作 (`--all`)。
- **命令**:
  ```bash
  jpy-cli middleware device usb -m host -g "lab" --all
  ```

**场景 4: 查看设备日志 (单个目标)**
- **用户意图**: "显示服务器 192.168.1.100 上 5 号盘位的日志。"
- **推理**: 日志流式传输需要指定唯一目标。
- **命令**:
  ```bash
  jpy-cli middleware device log -s "192.168.1.100" --seat 5
  ```

**场景 5: 按UUID状态筛选设备**
- **用户意图**: "只列出有UUID的设备。"
- **推理**: 使用 `--filter-uuid true` 筛选有UUID的设备。
- **命令**:
  ```bash
  jpy-cli middleware device list --filter-uuid true
  ```

**场景 6: 按UUID数量筛选服务器**
- **用户意图**: "显示UUID数量大于10的服务器。"
- **推理**: 使用 `--uuid-count-gt 10` 按UUID数量筛选服务器。
- **命令**:
  ```bash
  jpy-cli middleware device status --uuid-count-gt 10
  ```

**场景 7: 导出设备信息到文件**
- **用户意图**: "将所有在线设备的ID和UUID导出到文件。"
- **推理**: 使用 `--filter-online true` 筛选在线设备，使用 `--export-id` 和 `--export-uuid` 导出指定字段。
- **命令**:
  ```bash
  jpy-cli middleware device export devices.txt --export-id --export-uuid --filter-online true
  ```

**场景 8: 导出特定服务器设备信息**
- **用户意图**: "导出192.168.1网段所有设备的完整信息。"
- **推理**: 使用 `-s "192.168.1"` 筛选特定服务器，不指定导出字段时默认导出所有字段。
- **命令**:
  ```bash
  jpy-cli middleware device export 192_168_1_devices.txt -s "192.168.1"
  ```

### 4.2 服务器维护场景

**场景 5: 清理死链服务器**
- **用户意图**: "移除所有当前报错的服务器。"
- **推理**: 使用 `--has-error` 筛选，使用 `--all` 确认删除匹配集合。
- **命令**:
  ```bash
  jpy-cli middleware remove --has-error --all
  ```

**场景 6: 硬删除特定服务器**
- **用户意图**: "永久删除 IP 为 10.0.0.5 的服务器。"
- **推理**: 按 IP 搜索 (`--search`)，强制删除 (`--force`)，确认 (`--all` 通常在搜索有匹配时需要，对于 AI，如果确信匹配则使用 `--all`)。
- **命令**:
  ```bash
  jpy-cli middleware remove --search "10.0.0.5" --force --all
  ```

**场景 7: 尝试恢复**
- **用户意图**: "尝试重新连接所有已禁用的服务器。"
- **命令**:
  ```bash
  jpy-cli middleware relogin
  ```

### 4.3 配置场景

**场景 8: 增加并发数**
- **用户意图**: "将最大并发数设置为 20 以加快扫描速度。"
- **命令**:
  ```bash
  jpy-cli config set max_concurrency 20
  ```

**场景 9: 调试模式**
- **用户意图**: "启用调试日志。"
- **命令**:
  ```bash
  jpy-cli config set log_level debug
  ```

---

## 5. 开发参考

供扩展此工具的开发者参考：
- **构建**: `make build` (本地), `make dist` (跨平台)。
- **运行源码**: 在所有示例中将 `jpy-cli` 替换为 `go run main.go`。
- **架构**:
    - `pkg/api`: 业务逻辑接口。
    - `pkg/service`: 连接和状态管理。
    - `pkg/client/ws`: WebSocket 传输层。
    - `cmd`: Cobra 命令定义。



## 6. 实战场景

# 添加、切换中间件服务器分组
1. 查看当前活动/选择分组
.\jpy.exe middleware auth select

2. 选择使用指定分组内的服务器
.\jpy.exe middleware auth select [group]

3. 添加服务器到当前活动分组
.\jpy.exe middleware auth login "192.168.0.102" -u admin -p admin


# 设备初次上线没有获取到IP，需尝试切换USB和OTG使得设备成功获取到IP
1. 将已授权的服务器，没有IP及当前处于OTG的设备切换到USB模式
.\jpy.exe middleware device usb --mode usb --authorized --filter-has-ip false --filter-usb false

2. 重新切换回 OTG模式
.\jpy.exe middleware device usb --mode host --authorized --filter-has-ip false --filter-usb true

3. 统计设备情况
 .\jpy.exe middleware device status

4. 把所有处于USB模式全部切换到OTG
.\jpy.exe middleware device usb --mode host --authorized --filter-usb true

5. 第1到第3一直循环执行，直到连续循环3次IP缺失数量还是保持没有减少，说明可能是设备本身原因，结束

6. 尝试强制重启（断电在通电）经过多次切换模式依然不上线IP的设备
.\jpy.exe middleware device reboot --filter-has-ip false

7. 约等待3分钟后，重新执行前面的步骤（1-5）最后最终实在无力解决

# 初次添加服务器到分组后，扫描服务器状态并暂时隔离登录失败服务器提高后续使用效率
1. 统计设备，将会自动登录
.\jpy.exe middleware device status

2. 软删除登录失败的服务器
.\jpy.exe middleware remove --has-error

3. 单独尝试重新登录软失败的服务器，成功的会自动恢复删除状态（建议循环3次）
.\jpy.exe middleware relogin


# 抽查上线失败的设备（IP无法获取到的）执行日志
1. 只列出无IP的设备
.\jpy.exe middleware device list --filter-has-ip false

2. 根据列出的设备随意抽查1个去获取设备内的日志
.\jpy.exe middleware device log --server 192.168.10.206 --seat 12


# 添加、切换中间件服务器分组
1. 查看当前活动/选择分组
.\jpy.exe middleware auth select

2. 选择使用指定分组内的服务器
.\jpy.exe middleware auth select [group]

3. 添加服务器到当前活动分组
.\jpy.exe middleware auth login "192.168.0.102" -u admin -p admin

# 获取中间件root密码（连接到中间件shell需要）
.\jpy.exe middleware ssh "192.168.0.102"

# 查看某个设备的日志
.\jpy.exe middleware device log --server 192.168.0.102 --seat 12
