# JPY CLI Command Reference

A CLI tool for JPY middleware server management and device control, supporting both local LAN middleware and remote Cloud Platform APIs.

## Command Tree

```
jpy
├── middleware                # Middleware management (LAN)
│   ├── list                  # List servers in current group
│   ├── auth                  # Authentication & server management
│   │   ├── login             # Login to server
│   │   ├── create            # Batch generate server configs
│   │   ├── list              # List configured servers
│   │   ├── select            # Select/switch active group
│   │   ├── import            # Import configs from JSON file
│   │   ├── export            # Export current group config
│   │   └── template          # Generate config template
│   ├── device                # Device management
│   │   ├── list              # List device details
│   │   ├── status            # Server status & device statistics
│   │   ├── export            # Export device info to file
│   │   ├── reboot            # Reboot devices (power cycle)
│   │   ├── usb               # Switch USB mode
│   │   ├── adb               # Control ADB state
│   │   └── log               # View single device log
│   ├── admin                 # Admin commands
│   │   ├── auto-auth         # Auto-scan and authorize servers
│   │   └── update-cluster    # Batch update control platform address
│   ├── remove                # Remove/soft-delete servers
│   ├── relogin               # Reconnect soft-deleted servers
│   ├── restart               # Restart boxCore service
│   └── ssh                   # SSH to middleware (auto-password)
├── cloud                     # Cloud Platform Remote API
│   ├── config                # View/modify cloud config
│   │   └── init-configs      # Create sample config files
│   └── stress                # Device modification stress test
├── admin                     # System admin commands
│   └── middleware
│       ├── generate          # Generate license codes
│       ├── list              # List license codes
│       └── get-root-password # Get root password
├── config                    # Local config management
│   ├── list                  # List all configs
│   ├── get                   # Get config value
│   └── set                   # Set config value
├── server                    # Backend services
│   ├── ssh                   # Start SSH jump server
│   └── web                   # Start web server
├── proxy                     # Transparent proxy
├── tools                     # Utilities
│   ├── middleware create     # Batch generate middleware configs
│   └── completion-install    # Install shell completion
└── log                       # View CLI operation logs
```

---

## Global Flags

| Flag | Type | Description |
|------|------|-------------|
| `--debug` | bool | Enable debug logging (also outputs to console) |
| `--log-level` | string | Log level: `debug`/`info`/`warn`/`error` |

---

## Output Modes `--output` / `-o`

Most commands support three output modes:

| Mode | Description | Use Case |
|------|-------------|----------|
| `tui` | Default interactive TUI | Human operation |
| `plain` | Plain text, tab-separated | Script parsing, Win7 SSH |
| `json` | Structured JSON | AI/programmatic parsing |

### Commands Supporting `--output`

| Command | tui | plain | json |
|---------|-----|-------|------|
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

> **Win7 Note**: Win7 SSH does not support TUI raw mode. Use `-o plain` or `-o json` to avoid crashes.

---

## Middleware Management (`middleware`)

### `middleware list` — List Servers

```bash
jpy middleware list [-o json|plain]
```

**JSON Output**:
```json
{
  "group": "default",
  "total": 3,
  "servers": [
    { "url": "192.168.1.201:443", "username": "admin", "status": "ok" }
  ]
}
```

### `middleware auth login` — Login to Server

```bash
jpy middleware auth login <url> -u <username> -p <password> [-g <group>] [-o json]
```

### `middleware auth create` — Batch Generate Server Configs

```bash
jpy middleware auth create --ip <range> [-P <port>] [-u <user>] [-p <pass>] [-o json]
```

> Non-interactive mode (`-o json/plain`) requires `--ip`.

### `middleware auth list` — List Configured Servers

```bash
jpy middleware auth list [--details] [-o json]
```

### `middleware auth select` — Select/Switch Group

```bash
# List groups
jpy middleware auth select -o json

# Switch group
jpy middleware auth select <group_name> -o json
```

### `middleware auth import/export/template`

```bash
jpy middleware auth import <file>
jpy middleware auth export [-o <output_file>]
jpy middleware auth template
```

### `middleware remove` — Remove Servers

```bash
jpy middleware remove [--search <keyword>] [--has-error] [--force] [--all] [-o json]
```

> Non-interactive mode requires `--all`, `--has-error`, or `--search`.

### `middleware relogin` — Reconnect Soft-Deleted Servers

```bash
jpy middleware relogin [-o json]
```

**JSON Output**:
```json
{
  "total": 5, "restored": 3, "failed": 2,
  "results": [
    { "url": "192.168.1.201:443", "status": "restored" },
    { "url": "192.168.1.202:443", "status": "failed", "error": "connection refused" }
  ]
}
```

### `middleware ssh` — SSH to Middleware

```bash
jpy middleware ssh <IP>
```

### `middleware restart` — Restart boxCore Service

```bash
jpy middleware restart [filter flags]
```

---

## Device Management (`middleware device`)

### Common Filter Flags

| Flag | Alias | Description |
|------|-------|-------------|
| `--group` | `-g` | Server group name |
| `--server` | `-s` | Server address fuzzy match |
| `--uuid` | `-u` | Device UUID fuzzy match |
| `--seat` | — | Seat number |
| `--authorized` | — | Authorized servers only |
| `--filter-online` | — | Filter by online status |
| `--filter-adb` | — | Filter by ADB status |
| `--filter-usb` | — | Filter by USB mode (true=USB, false=OTG) |
| `--filter-has-ip` | — | Filter by IP presence |
| `--filter-uuid` | — | Filter by UUID presence |
| `--interactive` | `-i` | Interactive selection mode |
| `--all` | — | Execute on all matched devices |
| `--output` | `-o` | Output mode (tui/plain/json) |

### `device list` — List Device Details

```bash
jpy middleware device list [filters] [-o json]
```

**JSON**: `{ "total": N, "devices": [{ "server", "seat", "uuid", "ip", "online", "biz_online", "usb_mode", "adb", "model", "android" }] }`

### `device status` — Server Status & Statistics

```bash
jpy middleware device status [filters] [--detail] [-o json]
```

Additional status-specific filters: `--auth-failed`, `--fw-has/--fw-not`, `--speed-gt/--speed-lt`, `--cluster-contains/--cluster-not-contains`, `--sn-gt/--sn-lt`, `--ip-count-gt/--ip-count-lt`, `--biz-online-gt/--biz-online-lt`, `--uuid-count-gt/--uuid-count-lt`

**JSON**: `{ "summary": { totals... }, "servers": [{ per-server stats... }] }`

### `device export` — Export Device Info

```bash
jpy middleware device export [file] [filters] [--export-id|--export-ip|--export-uuid|--export-seat|--export-auto] [-o json]
```

### `device reboot` — Reboot Devices

```bash
jpy middleware device reboot [filters] [--all] [-o json]
```

Exit codes: 0=all success, 1=partial failure, 2=all failed

### `device usb` — Switch USB Mode

```bash
jpy middleware device usb --mode <host|device> [filters] [--all] [-o json]
```

### `device adb` — Control ADB State

```bash
jpy middleware device adb --set <on|off> [filters] [--all] [-o json]
```

### `device log` — View Device Log

```bash
jpy middleware device log -s <server_ip> --seat <seat_number>
```

---

## Middleware Admin (`middleware admin`)

```bash
jpy middleware admin auto-auth
jpy middleware admin update-cluster <new_address> [--server <match>] [--group <group>] [--authorized] [--force]
```

---

## Cloud Platform (`cloud`)

```bash
jpy cloud config [-o json]
jpy cloud config init-configs
jpy cloud stress [--config <file>] [-o json]
```

---

## System Admin (`admin`)

```bash
jpy admin middleware generate
jpy admin middleware list
jpy admin middleware get-root-password
```

---

## Local Config (`config`)

```bash
jpy config list
jpy config get <key>
jpy config set <key> <value>
```

| Config Key | Default | Description |
|------------|---------|-------------|
| `log_level` | `info` | Log level |
| `log_output` | `file` | Output: `console`/`file`/`both` |
| `max_concurrency` | `5` | Max concurrency |
| `connect_timeout` | `3` | Connection timeout (seconds) |

---

## Other Commands

```bash
jpy proxy [-p <port>]            # Transparent proxy
jpy server ssh                   # SSH jump server
jpy server web                   # Web server
jpy tools completion-install     # Install shell completion
jpy log [-f] [-n <lines>] [--grep <filter>]  # CLI operation logs
```

---

## Operational Scenarios

### AI Remote Batch Control (JSON Mode)

```bash
jpy middleware list -o json
jpy middleware device status -o json
jpy middleware device reboot --filter-has-ip false --all -o json
```

### Win7 SSH Environment (Plain Mode)

```bash
jpy middleware device status -o plain
jpy middleware device list -o plain
jpy middleware remove --has-error --all -o plain
```

### Device Initial Online Fix (IP Recovery Loop)

```bash
# Switch no-IP OTG devices to USB
jpy middleware device usb --mode device --authorized --filter-has-ip false --filter-usb false --all -o json
# Switch back to OTG
jpy middleware device usb --mode host --authorized --filter-has-ip false --filter-usb true --all -o json
# Check stats
jpy middleware device status -o json
# Repeat until IP count stabilizes (3 loops)
# Force reboot remaining no-IP devices
jpy middleware device reboot --filter-has-ip false --all -o json
```

### New Server Onboarding

```bash
jpy middleware auth create --ip "192.168.1.201-230" -o json
jpy middleware device status -o json
jpy middleware remove --has-error --all -o json
jpy middleware relogin -o json  # repeat 3x
```

---

## Build & Release

```bash
make build          # Development build
make dist           # Cross-platform release (Linux/macOS/Windows amd64+arm64)
make dist-win7      # Win7 compatible build (Go 1.19, excludes cloud module)
```

Config: `~/.jpy/config.yaml` | Data: `~/.jpy/servers.json` | Logs: `~/.jpy/logs/jpy.log`
