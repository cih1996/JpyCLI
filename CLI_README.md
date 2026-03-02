# JPY CLI Command Reference & AI Instruction Manual

This document provides a comprehensive, linear reference for the JPY CLI tool. It is structured to be easily parsed by AI agents to generate correct command sequences, while remaining readable for human operators.

## 1. System Overview

**Tool Name**: `jpy-cli` (Development alias: `go run main.go`)
**Purpose**: Management of JPY middleware servers, device control (ADB/USB/Power), and authorization administration.
**Interaction Philosophy**:
- **Default**: Interactive TUI (Text User Interface) for human convenience.
- **Automation/AI**: **Strictly Non-Interactive**. Use specific flags (`--all`, `--force`, `-s`, `--seat`) to bypass prompts.

## 2. Command Protocol

### 2.1 Global Syntax
```bash
jpy-cli [scope] [resource] [action] [flags]
```

### 2.2 Global Flags
| Flag | Type | Description |
| :--- | :--- | :--- |
| `--debug` | boolean | Enable debug logging and verbose output. |
| `--log-level` | string | Set log level: `debug`, `info`, `warn`, `error`. |
| `--config` | string | Specify custom config file path. |

### 2.3 Common Filter Flags (Device Scope)
Used for `device list`, `device status`, and `device control` commands.
| Flag | Alias | Description | Example |
| :--- | :--- | :--- | :--- |
| `--server` | `-s` | Server IP/URL keyword (fuzzy match). | `-s "192.168.1"` |
| `--group` | `-g` | Server group name. | `-g "default"` |
| `--seat` | | Specific seat number (integer). | `--seat 5` |
| `--uuid` | `-u` | Device UUID (fuzzy match). | `-u "f73a9c"` |
| `--authorized`| | Filter by authorized servers only. | `--authorized` |
| `--filter-online`| | Filter by device online status. | `--filter-online true` |
| `--filter-adb` | | Filter by ADB enabled status. | `--filter-adb true` |
| `--filter-usb` | | Filter by USB mode (true=Device/USB, false=Host/OTG). | `--filter-usb false` |
| `--filter-uuid` | | Filter by UUID presence status (true/false). | `--filter-uuid true` |
| `--filter-has-ip` | | Filter by IP presence status (true/false). | `--filter-has-ip false` |
| `--uuid-count-gt` | | Filter servers with UUID count greater than specified value. | `--uuid-count-gt 10` |
| `--uuid-count-lt` | | Filter servers with UUID count less than specified value. | `--uuid-count-lt 5` |

---

## 3. Command Library

### 3.1 Scope: Middleware Device (`middleware device`)
Manage and control connected devices.

#### `list`
- **Intent**: Retrieve detailed list of devices matching criteria.
- **Syntax**: `jpy-cli middleware device list [flags]`

#### `export`
- **Intent**: Export device information to a file with customizable fields.
- **Syntax**: `jpy-cli middleware device export [output-file] [flags]`
- **Key Flags**:
        - `--export-id`: Export device ID (generated from server address).
    - `--export-ip`: Export device IP address.
    - `--export-uuid`: Export device UUID.
    - `--export-seat`: Export device seat number.
    - `--export-auto`: Intelligent export mode: automatically complete missing IP addresses for devices with UUID but no IP, skip devices without UUID, and log statistics.
- **Default Behavior**: If no flags specified, exports all fields in format: `ID\tUUID\tIP\tSeat`
- **Use Cases**:
  - **DHCP Configuration**: When some devices fail to read IP but have SN (UUID), use `--export-auto` mode to intelligently infer and complete missing IP addresses, generating a complete device list for DHCP server configuration
  - **Batch Device Management**: Export device information for automated script processing
  - **Device Inventory**: Create complete device inventory with all necessary connection information

#### `status`
- **Intent**: Get aggregated server status and device counts.
- **Syntax**: `jpy-cli middleware device status [flags]`
- **Key Flags**:
    - `--detail`: Show detailed authorization info (SN, Control Platform, License Name).
    - **Advanced Filters**:
        - `--auth-failed`: Filter servers with authorization issues.
        - `--fw-has`: Filter servers where firmware version contains string.
        - `--fw-not`: Filter servers where firmware version does not contain string.
        - `--speed-gt`: Filter servers with network speed > value (Mbps).
        - `--speed-lt`: Filter servers with network speed < value (Mbps).
        - `--cluster-contains`: Filter servers where control platform address contains string.
        - `--cluster-not-contains`: Filter servers where control platform address does not contain string.
        - `--sn-gt`: Filter servers with SN > value (string compare).
        - `--sn-lt`: Filter servers with SN < value (string compare).
        - `--ip-count-gt`: Filter servers with IP count > value.
        - `--ip-count-lt`: Filter servers with IP count < value.
        - `--biz-online-gt`: Filter servers with Business Online count > value.
        - `--biz-online-lt`: Filter servers with Business Online count < value.

#### `reboot`
- **Intent**: Power cycle the device.
- **Syntax**: `jpy-cli middleware device reboot [flags]`
- **Critical Flags**:
    - `--all`: Execute on all matching devices without confirmation.

#### `usb`
- **Intent**: Switch USB MUX mode.
- **Syntax**: `jpy-cli middleware device usb [flags]`
- **Critical Flags**:
    - `-m, --mode`: Target mode (`host` for OTG, `device` for USB).
    - `--all`: Execute without confirmation.

#### `adb`
- **Intent**: Enable or disable ADB debugging.
- **Syntax**: `jpy-cli middleware device adb [flags]`
- **Critical Flags**:
    - `--set`: Target state (`on` or `off`).
    - `--all`: Execute without confirmation.

#### `log`
- **Intent**: Real-time log monitoring for a **single** device.
- **Syntax**: `jpy-cli middleware device log [flags]`
- **Constraint**: Must target a single device via `-s` and `--seat`.
- **Automation**: Automatically handles USB switch -> ADB enable -> Shell connection -> Tail log.

---

### 3.2 Scope: Middleware Server (`middleware`)
Manage middleware server instances.

#### `remove`
- **Intent**: Remove (soft delete) a server from the active list.
- **Syntax**: `jpy-cli middleware remove [flags]`
- **Critical Flags**:
    - `--search`: Keyword to match server URL/Name.
    - `--force`: Hard delete (permanent) instead of soft delete.
    - `--has-error`: Target only servers with connection errors.
    - `--all`: Target all matching servers (requires careful use).

#### `relogin`
- **Intent**: Attempt to reactivate/reconnect soft-deleted servers.
- **Syntax**: `jpy-cli middleware relogin`

#### `auth login`
- **Intent**: Authenticate and add a new middleware server.
- **Syntax**: `jpy-cli middleware auth login [flags]`

#### `auth create`
- **Intent**: Batch generate middleware server configurations and add them to the current group.
- **Syntax**: `jpy-cli middleware auth create [flags]`
- **Modes**:
    - **Interactive**: Default mode if no flags provided.
    - **Non-Interactive (Batch)**: Use flags to specify parameters.
- **Key Flags**:
    - `-i, --ip`: IP range(s), supports comma-separated list (e.g., `192.168.1.201-210,192.168.2.100`).
    - `-P, --port`: Server port (default: 443).
    - `-u, --username`: Admin username (default: admin).
    - `-p, --password`: Admin password (default: admin).

#### `auth export`
- **Intent**: Export current group configuration to `servers.json` (LocalServerConfig format).
- **Syntax**: `jpy-cli middleware auth export [flags]`
- **Key Flags**:
    - `-o, --output`: Output file path (default: `servers.json`).

#### `auth list`
- **Intent**: List configured servers.
- **Syntax**: `jpy-cli middleware auth list [flags]`

#### `auth import`
- **Intent**: Batch import servers from JSON.
- **Syntax**: `jpy-cli middleware auth import [file]`

#### `ssh`
- **Intent**: Connect to a middleware server via SSH (Automatically retrieves Root password).
- **Syntax**: `jpy-cli middleware ssh [ip]`
- **Prerequisites**: Requires Operation Login (automatically handled).
- **Behavior**:
    1. Connects to port 22 to retrieve the banner key.
    2. Decrypts the root password using the Admin API.
    3. Generates a connection command (supports `sshpass` if installed).

#### `restart`
- **Intent**: Restart the boxCore service on selected devices.
- **Syntax**: `jpy-cli middleware restart [flags]`
- **Note**: Supports concurrent execution and common filters (Group, Server, UUID, Seat, etc.).

---

### 3.3 Scope: Admin (`admin`)
System administration and license management.

#### `middleware admin auto-auth`
- **Intent**: Auto-scan and authorize pending middleware servers.
- **Syntax**: `jpy-cli middleware admin auto-auth`

#### `middleware admin update-cluster`
- **Intent**: Batch update the Control Platform address (MgtCenter) for middleware servers and sync with Admin Backend.
- **Syntax**: `jpy-cli middleware admin update-cluster [new_address] [flags]`
- **Key Flags**:
    - `--server`: Filter server URL/Name (regex supported).
    - `--group`: Target specific server group.
    - `--authorized`: Filter by authorization status (true/false).
    - `--force`: Force update even if the address already matches.

#### `admin device generate`
- **Intent**: Generate new license codes.
- **Syntax**: `jpy-cli admin device generate`

#### `admin device list`
- **Intent**: List generated licenses.
- **Syntax**: `jpy-cli admin device list`

---

### 3.4 Scope: System (`config`, `log`)

#### `config`
- **Intent**: Read/Write local configuration.
- **Syntax**:
    - `jpy-cli config list`
    - `jpy-cli config get <key>`
    - `jpy-cli config set <key> <value>`

#### `log`
- **Intent**: Tail the CLI's own operation log (`jpy.log`).
- **Syntax**: `jpy-cli log [flags]`
- **Flags**: `-f` (follow), `-n` (lines), `--grep` (filter).

---

## 4. Operational Scenarios (AI Training Data)

This section maps **User Intent** to **Precise Command Execution**. AI agents should prioritize these patterns to ensure non-interactive success.

### 4.1 Device Control Scenarios

**Scenario 1: Batch Reboot Specific Subnet**
- **User Intent**: "Reboot all devices on servers starting with 192.168.23."
- **Reasoning**: Use `-s` for fuzzy match, `--all` to skip TUI.
- **Command**:
  ```bash
  jpy-cli middleware device reboot -s "192.168.23" --all
  ```

**Scenario 2: Turn Off ADB for Security**
- **User Intent**: "Disable ADB on all currently online devices."
- **Reasoning**: Filter online devices (`--filter-online true`), set ADB off (`--set off`), execute batch (`--all`).
- **Command**:
  ```bash
  jpy-cli middleware device adb --set off --filter-online true --all
  ```

**Scenario 3: Switch to Host Mode (OTG)**
- **User Intent**: "Switch all devices in the 'lab' group to OTG mode."
- **Reasoning**: Group filter (`-g`), set mode (`-m host`), execute batch (`--all`).
- **Command**:
  ```bash
  jpy-cli middleware device usb -m host -g "lab" --all
  ```

**Scenario 4: View Device Logs (Single Target)**
- **User Intent**: "Show me the logs for seat 5 on server 192.168.1.100."
- **Reasoning**: Specific target required for log streaming.
- **Command**:
  ```bash
  jpy-cli middleware device log -s "192.168.1.100" --seat 5
  ```

**Scenario 5: Filter Devices by UUID Status**
- **User Intent**: "List only devices that have UUIDs."
- **Reasoning**: Use `--filter-uuid true` to filter devices with UUIDs.
- **Command**:
  ```bash
  jpy-cli middleware device list --filter-uuid true
  ```

**Scenario 6: Filter Servers by UUID Count**
- **User Intent**: "Show servers with more than 10 UUIDs."
- **Reasoning**: Use `--uuid-count-gt 10` to filter servers by UUID count.
- **Command**:
  ```bash
  jpy-cli middleware device status --uuid-count-gt 10
  ```

### 4.2 Server Maintenance Scenarios

**Scenario 5: Cleanup Dead Servers**
- **User Intent**: "Remove all servers that are currently reporting errors."
- **Reasoning**: Use `--has-error` filter, use `--all` to confirm deletion of matched set.
- **Command**:
  ```bash
  jpy-cli middleware remove --has-error --all
  ```

**Scenario 6: Hard Delete Specific Server**
- **User Intent**: "Permanently delete the server at 10.0.0.5."
- **Reasoning**: Search by IP (`--search`), force delete (`--force`), confirm (`--all` usually required if search returns matches, or interactive; for AI, assume `--all` if confident).
- **Command**:
  ```bash
  jpy-cli middleware remove --search "10.0.0.5" --force --all
  ```

**Scenario 7: Attempt Recovery**
- **User Intent**: "Try to reconnect all disabled servers."
- **Command**:
  ```bash
  jpy-cli middleware relogin
  ```

### 4.3 Configuration Scenarios

**Scenario 8: Increase Concurrency**
- **User Intent**: "Set maximum concurrency to 20 for faster scanning."
- **Command**:
  ```bash
  jpy-cli config set max_concurrency 20
  ```

**Scenario 9: Debug Mode**
- **User Intent**: "Enable debug logging."
- **Command**:
  ```bash
  jpy-cli config set log_level debug
  ```

---

## 5. Development Reference

For developers extending this tool:
- **Build**: `make build` (local), `make dist` (cross-platform).
- **Run Source**: Replace `jpy-cli` with `go run main.go` in all examples.
- **Architecture**:
    - `pkg/api`: Business logic interfaces.
    - `pkg/service`: Connection and state management.
    - `pkg/client/ws`: WebSocket transport.
    - `cmd`: Cobra command definitions.

---

## 6. Real-world Scenarios

### Device Initial Online Failure (No IP) - Cycle USB/OTG to fix
1. **Switch authorized servers, devices with no IP and currently in OTG mode to USB mode.**
   ```bash
   .\jpy.exe middleware device usb --mode usb --authorized --filter-has-ip false --filter-usb false
   ```

2. **Switch back to OTG mode.**
   ```bash
   .\jpy.exe middleware device usb --mode host --authorized --filter-has-ip false --filter-usb true
   ```

3. **Check device status statistics.**
   ```bash
   .\jpy.exe middleware device status
   ```

4. **Switch all devices currently in USB mode to OTG mode.**
   ```bash
   .\jpy.exe middleware device usb --mode host --authorized --filter-usb true
   ```

5. **Repeat Steps 1-3:** Loop this process. If the number of devices missing IP does not decrease after 3 consecutive loops, it indicates a likely hardware issue. Stop.

6. **Force Reboot:** Attempt forced reboot (power cycle) for devices that still fail to get IP after mode switching.
   ```bash
   .\jpy.exe middleware device reboot --filter-has-ip false
   ```

7. **Wait and Retry:** Wait approximately 3 minutes, then repeat steps 1-5. If still unresolved, manual intervention is required.

### Optimize Efficiency after Adding New Servers: Isolate Login Failures
1. **Check Status (Triggers Auto-login):**
   ```bash
   .\jpy.exe middleware device status
   ```

2. **Soft-delete Servers with Errors:**
   ```bash
   .\jpy.exe middleware remove --has-error
   ```

3. **Retry Login:** Attempt to re-login soft-deleted servers separately. Successful ones automatically recover. (Recommend looping 3 times).
   ```bash
   .\jpy.exe middleware relogin
   ```

### Export Device Information for Analysis
1. **Export all online devices with ID and UUID:**
   ```bash
   .\jpy.exe middleware device export devices.txt --export-id --export-uuid --filter-online true
   ```

2. **Export complete device information for specific server range:**
   ```bash
   .\jpy.exe middleware device export server_devices.txt -s "192.168.1"
   ```

### Investigate Logs for Devices Failed to Online (No IP)
1. **List Only Devices Without IP:**
   ```bash
   .\jpy.exe middleware device list --filter-has-ip false
   ```

2.3. Retrieve Log for a Sample Device: Pick one device from the list to inspect internal logs.
   ```bash
   .\jpy.exe middleware device log --server 192.168.10.206 --seat 12
   ```

### Add/Switch Middleware Server Groups
1. View current active/selected group
   ```bash
   .\jpy.exe middleware auth select
   ```

2. Select/Switch to specific group
   ```bash
   .\jpy.exe middleware auth select [group]
   ```

3. Add server to current active group
   ```bash
   .\jpy.exe middleware auth login "192.168.0.102" -u admin -p admin
   ```
