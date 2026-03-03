# JPY CLI Reference Manual

> One command controls all devices across multiple middleware servers. Add filters to narrow down to a single device on a single seat.

---

## Table of Contents

1. [Core Concept: Cluster Control](#1-core-concept-cluster-control)
2. [Self-Help Discovery with --help](#2-self-help-discovery-with---help)
3. [Control Granularity: Global to Single Device](#3-control-granularity-global-to-single-device)
4. [Command Tree](#4-command-tree)
5. [Device Cluster Control (Core Feature)](#5-device-cluster-control-core-feature)
6. [Middleware Server Management](#6-middleware-server-management)
7. [Output Modes --output](#7-output-modes---output)
8. [Operational Scenarios](#8-operational-scenarios)
9. [Other Commands](#9-other-commands)

---

## 1. Core Concept: Cluster Control

JPY CLI is designed for **cross-server cluster batch control**. An active group can contain multiple middleware servers, each managing N device slots.

```
Group: production
├── 192.168.1.201:443  →  Seats 1~30  →  30 devices
├── 192.168.1.202:443  →  Seats 1~30  →  30 devices
└── ... N middleware servers
```

**Without filters** = operate on ALL devices across ALL middleware in the active group:

```bash
jpy middleware device list          # List all devices
jpy middleware device reboot --all  # Reboot all devices
```

**With filters** = precise control:

```bash
jpy middleware device list -s 192.168.1.201                   # One server
jpy middleware device reboot -s 192.168.1.201 --seat 3 --all  # One device
```

---

## 2. Self-Help Discovery with `--help`

**`--help` is your primary tool for exploring commands.** Every command supports it.

```bash
jpy --help                              # Top-level commands
jpy middleware --help                   # middleware subcommands
jpy middleware device --help            # device subcommands
jpy middleware device list --help       # Full flags for list
jpy middleware device reboot --help     # Full flags for reboot
jpy middleware device shell --help      # Full flags for shell
```

Example output of `jpy middleware device list --help`:

```
Flags:
      --all                          Execute on all matched devices (skip confirmation)
      --authorized string[="true"]   Filter by auth status (true/false)
      --filter-adb string            Filter by ADB status (true/false)
      --filter-has-ip string         Filter by IP presence (true/false)
      --filter-online string         Filter by online status (true/false)
      --filter-usb string            Filter by USB mode (true=USB, false=OTG)
      --filter-uuid string           Filter by UUID presence (true/false)
  -g, --group string                 Target server group
  -h, --help                         help for list
  -i, --interactive                  Interactive TUI selection mode
  -l, --limit int                    Limit display count (default 100)
  -o, --output string                Output mode: tui/plain/json (default "tui")
      --seat int                     Seat number (default -1)
  -s, --server string                Server address pattern (e.g. 192.168.1)
  -u, --uuid string                  Device UUID (fuzzy match)
```

> **Rule**: When unsure, run `--help` first.

---

## 3. Control Granularity: Global to Single Device

Using `device reboot` as example:

| Granularity | Command | Scope |
|-------------|---------|-------|
| **Global** | `jpy middleware device reboot --all` | All devices in active group |
| **By group** | `jpy middleware device reboot -g production --all` | All devices in named group |
| **By subnet** | `jpy middleware device reboot -s "192.168.1" --all` | All devices on matching servers |
| **By server** | `jpy middleware device reboot -s "192.168.1.201" --all` | All devices on one server |
| **By status** | `jpy middleware device reboot --filter-online true --all` | All online devices |
| **Single device** | `jpy middleware device reboot -s "192.168.1.201" --seat 5 --all` | Seat 5 on one server |
| **By UUID** | `jpy middleware device reboot -u "ABCD1234" --all` | Exactly one device |

The same filter logic applies to: `list`, `status`, `reboot`, `usb`, `adb`, `shell`, `export`.

---

## 4. Command Tree

```
jpy
├── middleware                     Middleware management (LAN cluster control)
│   ├── list                       List middleware servers in active group
│   ├── auth                       Auth & server management
│   │   ├── login                  Add/login to a middleware server
│   │   ├── create                 Batch generate and add server configs
│   │   ├── list                   List configured servers
│   │   ├── select                 Switch active group
│   │   ├── import                 Import from JSON file
│   │   ├── export                 Export current group config
│   │   └── template               Generate config template
│   ├── device                     ★ Device cluster control (core)
│   │   ├── list                   List device details (multi-filter)
│   │   ├── status                 Server health & device statistics
│   │   ├── reboot                 Batch/single device reboot
│   │   ├── usb                    Batch/single USB mode switch
│   │   ├── adb                    Batch/single ADB control
│   │   ├── shell                  Send shell command to a device
│   │   ├── export                 Export device info to file
│   │   └── log                    Real-time log for single device
│   ├── admin
│   │   ├── auto-auth              Auto-scan and authorize servers
│   │   └── update-cluster         Batch update control platform address
│   ├── remove                     Remove/soft-delete servers
│   ├── relogin                    Reconnect soft-deleted servers
│   ├── restart                    Restart boxCore service
│   └── ssh                        SSH to middleware (auto-password)
├── cloud                          Cloud Platform Remote API
│   ├── config / init-configs
│   └── stress
├── admin middleware generate/list/get-root-password
├── config list/get/set
├── server ssh/web
├── proxy
├── tools middleware create / completion-install
└── log
```

---

## 5. Device Cluster Control (Core Feature)

### 5.1 Common Filter Flags (All `device` Subcommands)

| Flag | Alias | Description | Example |
|------|-------|-------------|---------|
| `--group` | `-g` | Server group name | `-g production` |
| `--server` | `-s` | Server URL fuzzy match | `-s "192.168.1"` |
| `--uuid` | `-u` | Device UUID fuzzy match | `-u "ABCD1234"` |
| `--seat` | — | Exact seat number | `--seat 5` |
| `--authorized` | — | Authorized servers only | `--authorized` |
| `--filter-online` | — | Filter by biz online (true/false) | `--filter-online true` |
| `--filter-adb` | — | Filter by ADB state (true/false) | `--filter-adb true` |
| `--filter-usb` | — | Filter by USB mode (true=USB, false=OTG) | `--filter-usb false` |
| `--filter-has-ip` | — | Filter by IP presence (true/false) | `--filter-has-ip false` |
| `--filter-uuid` | — | Filter by UUID presence (true/false) | `--filter-uuid true` |
| `--all` | — | Execute on all matches, skip confirmation | `--all` |
| `--interactive` | `-i` | Interactive TUI selection | `-i` |
| `--output` | `-o` | tui / plain / json | `-o json` |

> See all flags for any command: `jpy middleware device <subcmd> --help`

### 5.2 `device list` — List Devices

```bash
jpy middleware device list                                    # All devices (TUI)
jpy middleware device list -s 192.168.1.201                  # One server
jpy middleware device list --filter-online true -o json      # Online devices, JSON
jpy middleware device list --filter-has-ip false -o plain    # No-IP devices
jpy middleware device list -s 192.168.1.201 --seat 5         # Single seat
```

**JSON output**:
```json
{
  "total": 120,
  "devices": [
    {
      "server": "192.168.1.201:443", "seat": 1, "uuid": "ABCD1234",
      "ip": "10.0.0.101", "online": true, "biz_online": true,
      "usb_mode": "otg", "adb": false, "model": "Redmi Note 12", "android": "13"
    }
  ]
}
```

### 5.3 `device status` — Server Statistics

```bash
jpy middleware device status                        # All servers
jpy middleware device status --detail               # With SN / cluster address
jpy middleware device status --ip-count-lt 25       # Servers with few IPs
jpy middleware device status --biz-online-lt 10     # Low-online servers
jpy middleware device status --help                 # All available flags
```

Additional status-only filters: `--auth-failed`, `--fw-has/--fw-not`, `--speed-gt/lt`, `--cluster-contains/--cluster-not-contains`, `--ip-count-gt/lt`, `--biz-online-gt/lt`, `--uuid-count-gt/lt`

### 5.4 `device reboot` — Reboot Devices

```bash
jpy middleware device reboot --all                              # All devices
jpy middleware device reboot -s 192.168.1.201 --all            # One server
jpy middleware device reboot --filter-online true --all        # Online only
jpy middleware device reboot -s 192.168.1.201 --seat 5 --all  # Single device
jpy middleware device reboot --filter-has-ip false --all -o json
```

Exit codes: `0`=all success, `1`=partial failure, `2`=all failed

### 5.5 `device usb` — Switch USB Mode

```bash
jpy middleware device usb --mode host --all                   # All → OTG
jpy middleware device usb --mode device -s 192.168.1.201 --all  # One server → USB
jpy middleware device usb --mode host --filter-usb true --all   # USB→OTG only
jpy middleware device usb --mode device --authorized --filter-has-ip false --filter-usb false --all
```

`--mode host` = OTG | `--mode device` = USB

### 5.6 `device adb` — Control ADB

```bash
jpy middleware device adb --set on --all                          # Enable all
jpy middleware device adb --set off --filter-online true --all    # Disable online
jpy middleware device adb --set off -s 192.168.1.201 --all       # One server
```

### 5.7 `device shell` — Send Shell Command

Send a shell command to a specific device and return its output. Works without ADB.

> **Parameter distinction**: `--server` (`-s`) matches the **middleware server** address; `--ip` matches the **target device IP** inside the middleware — these are completely different.

```bash
# Recommended: command as positional argument (concise)
jpy middleware device shell "ls -lh" --server 192.168.255.1 --ip 192.168.10.195

# Reboot to fastboot (pre-flash, no ADB required)
jpy middleware device shell "reboot bootloader" -s 192.168.255.1 --ip 192.168.10.195

# By seat number (no IP needed)
jpy middleware device shell "reboot bootloader" -s 192.168.255.1 --seat 3

# Query device info (JSON output)
jpy middleware device shell "getprop ro.product.model" -s 192.168.255.1 --ip 192.168.10.195 -o json

# Also supported: --command / -c flag (legacy style)
jpy middleware device shell -s 192.168.255.1 --ip 192.168.10.195 -c "getprop ro.product.model"
```

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--server` | `-s` | ✅ | **Middleware server** address (fuzzy match) |
| `--ip` | — | One of | **Target device IP** inside the middleware |
| `--seat` | — | One of | Device seat number (alternative to `--ip`) |
| `[command]` | — | ✅ | Shell command to run (positional, recommended) |
| `--command` | `-c` | ✅ | Shell command (alternative to positional arg) |
| `--output` | `-o` | — | `plain` (default) / `json` |

> **Execution channel**: auto-selects: f=14 (main Guard, compatible) → f=289 (new systems) → Terminal (fallback). No manual configuration needed.

**JSON output**:
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

### 5.8 `device export` — Export Device Info

```bash
jpy middleware device export output.txt                                           # All fields
jpy middleware device export uuids.txt --export-uuid --export-ip --filter-uuid true
jpy middleware device export dhcp.txt --export-auto                               # Smart: fill missing IPs
jpy middleware device export -o json > devices.json
```

### 5.9 `device log` — Real-time Device Log

```bash
jpy middleware device log -s 192.168.1.201 --seat 12   # Must specify single device
```

Press `Ctrl+C` to exit. Auto-restores USB/ADB state.

---

## 6. Middleware Server Management

```bash
jpy middleware auth select -o json                              # List groups
jpy middleware auth select production                           # Switch group
jpy middleware list -o json                                     # Servers in group
jpy middleware auth login "192.168.1.201" -u admin -p admin    # Add server
jpy middleware auth create --ip "192.168.1.201-230" -o json    # Batch add
jpy middleware remove --has-error --all -o json                 # Remove errored
jpy middleware relogin -o json                                  # Retry soft-deleted (x3)
jpy middleware ssh 192.168.1.201                                # SSH with auto-password
jpy middleware admin update-cluster "10.0.1.100" --authorized  # Update cluster addr
```

---

## 7. Output Modes `--output`

| Mode | Flag | Description | Use Case |
|------|------|-------------|----------|
| TUI | `tui` (default) | Interactive paged UI | Human operation |
| Plain | `plain` | Tab-separated text | Shell scripts, Win7 SSH |
| JSON | `json` | Structured JSON | AI/automation |

> **Win7 SSH**: TUI crashes due to raw mode. Always use `-o plain` or `-o json`.

---

## 8. Operational Scenarios

### A. New Server Batch Onboarding
```bash
jpy middleware auth create --ip "192.168.1.201-230" -o json
jpy middleware device status -o json
jpy middleware remove --has-error --all -o json
jpy middleware relogin -o json  # repeat 3x
```

### B. Device IP Recovery Loop
```bash
jpy middleware device usb --mode device --authorized --filter-has-ip false --filter-usb false --all -o json
jpy middleware device usb --mode host --authorized --filter-has-ip false --filter-usb true --all -o json
jpy middleware device status -o json  # check; repeat until stable
jpy middleware device reboot --filter-has-ip false --all -o json  # last resort
```

### C. Pre-Flash: Reboot to Fastboot
```bash
jpy middleware device list -s 192.168.255.1 --filter-online true -o json
jpy middleware device shell -s 192.168.255.1 --ip 192.168.10.195 -c "reboot bootloader"
```

### D. AI/Automation (JSON Mode)
```bash
jpy middleware device status -o json
jpy middleware device list --filter-online true -o json
jpy middleware device reboot --filter-has-ip false --all -o json  # exits 0/1/2
jpy middleware device shell -s ... --ip ... -c "getprop ro.build.version.release" -o json
```

### E. Security: Disable ADB
```bash
jpy middleware device adb --set off --filter-online true --all -o json
jpy middleware device list --filter-adb true -o plain  # verify
```

---

## 9. Other Commands

```bash
# Config
jpy config list
jpy config set max_concurrency 20
jpy config set connect_timeout 5

# Cloud Platform
jpy cloud config -o json
jpy cloud stress -o json

# Logs & Tools
jpy log -f --grep ERROR
jpy proxy -p 8888
jpy server ssh
jpy tools completion-install

# Admin
jpy admin middleware generate
jpy admin middleware list
```

---

## Build

```bash
make build        # Current platform
make dist         # All platforms (Linux/macOS/Windows amd64+arm64)
make dist-win7    # Win7 compatible (Go 1.19, no cloud module)
```

Config: `~/.jpy/` | Logs: `~/.jpy/logs/jpy.log`
