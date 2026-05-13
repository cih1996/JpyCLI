# JPY CLI User Guide

JPY middleware management command-line tool for AI agents, scripts, and UI automation. It is stateless, zero-config, and designed for one-command execution.

## Basic conventions

### Middleware connection flags

Commands that call middleware APIs use the following flags:

| Flag | Required | Description |
|------|----------|-------------|
| `-s, --server` | Yes | Middleware address, for example `172.25.0.251` or `http://172.25.0.251` |
| `-u, --user` | Yes | Middleware username |
| `-p, --password` | Yes | Middleware password |
| `-o, --output` | No | Output format: `plain` or `json`, default `plain` |

### Output formats

Most commands support both `plain` and `json` output:

```bash
jpy device list -s 172.25.0.251 -u admin -p 123456
jpy device list -s 172.25.0.251 -u admin -p 123456 -o json
```

Use `json` when integrating with a UI or automation layer.

### Remote execution

Start the HTTP server on the controlled machine first:

```bash
jpy server --port 9090
```

The process keeps running and listens on the given port. Remote communication uses **HTTP, not WebSocket**.

Call from the controller machine:

```bash
jpy --remote 192.168.1.100:9090 device list -s 172.25.0.251 -u admin -p 123456 -o json
```

Remote support matrix:

| Command type | Remote support | Notes |
|--------------|----------------|-------|
| `device` | Yes | Use global `jpy --remote <host:port> device ...` |
| `device middleware` | Yes | Firmware upgrade and ROM upload/list/flash/status/detail are supported |
| `com` | Yes | Useful when COM ports are on the controlled machine |
| `flash` | Yes | Useful when COM ports, fastboot, and flash scripts are on the controlled machine |
| `stress` | Yes | Use async mode for long-running jobs |
| `shell` | Yes | Dedicated form: `jpy shell --remote <host:port> ...` |
| `file` | Yes | Has its own `--remote` flag for file transfer |
| `update` | Yes | Has its own `--remote` flag for updating the remote jpy executable |
| `version` | Yes | Has its own `--remote` flag for checking remote version |
| `server` | No | Disabled to avoid recursive server startup |

Note: `file`, `update`, and `version` use their own `--remote` flag. Other normal commands use `jpy --remote <host:port> <command>`.

Use async mode for long-running jobs:

```bash
jpy --remote 192.168.1.100:9090 flash run ... --async --async-timeout 0
jpy shell --remote 192.168.1.100:9090 --task <task_id>
jpy shell --remote 192.168.1.100:9090 --kill <task_id>
```

Remote HTTP endpoints:

| Scenario | HTTP endpoint |
|----------|---------------|
| Sync CLI command | `POST /exec` |
| Async CLI command | `POST /exec/async` |
| Sync system shell command | `POST /shell` |
| Async system shell command | `POST /shell/async` |
| Query task | `GET /shell/task?id=<task_id>` |
| Kill task | `GET /shell/kill?id=<task_id>` |
| File transfer | `/file/*` |
| Version / health check | `GET /version`, `GET /health` |

---

## Document index

- [`README.md`](./README.md): project overview
- [`HTTP_REMOTE_CONTROL.md`](./HTTP_REMOTE_CONTROL.md): direct HTTP remote control
- [`HANDOFF.md`](./HANDOFF.md): handoff guide

## Command overview

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

jpy flash run      --com COM3 --mw <server> --ip-start <start-ip> --script <flash-script> [--ch 1-10] [--dry] [-y]
jpy file push      <local-file> --remote <host:port> [--dest <path>] [--timeout N]
jpy file pull      <url> --remote <host:port> [--dest <path>] [--timeout N]
jpy update         <local-file|URL> --remote <host:port>
jpy stress user    -s <ws-server> -k <secret-key> -c <config.json> [--device 1,2,3] [--loop N] [--interval 3m] [--timeout 10m]
jpy shell          --remote <host:port> -c "<cmd>" [--timeout N] [--async] [--task ID] [--tasks] [--kill ID]
jpy server         [--port 9090]
jpy version        [--remote <host:port>] [-o plain|json]
jpy --remote <host:port> <any command> [--async] [--async-timeout N]
```

---

## 1. Device management: `device`

### 1.1 List devices

```bash
jpy device list -s 172.25.0.251 -u admin -p 123456
jpy device list -s 172.25.0.251 -u admin -p 123456 -o json
jpy device list -s 172.25.0.251 -u admin -p 123456 --seat 3
jpy device list -s 172.25.0.251 -u admin -p 123456 --ip 10.0.0.5
jpy device list -s 172.25.0.251 -u admin -p 123456 --uuid 7b9f2b7a
```

### 1.2 Run shell commands on devices

```bash
jpy device shell "ls /sdcard" -s 172.25.0.251 -u admin -p 123456 --seat 3
jpy device shell "getprop ro.product.model" -s 172.25.0.251 -u admin -p 123456 --ip 10.0.0.5 -o json
```

### 1.3 Reboot devices

```bash
jpy device reboot -s 172.25.0.251 -u admin -p 123456
jpy device reboot -s 172.25.0.251 -u admin -p 123456 --seat 3
jpy device reboot -s 172.25.0.251 -u admin -p 123456 --ip 10.0.0.5
jpy device reboot -s 172.25.0.251 -u admin -p 123456 --uuid 7b9f2b7a
```

### 1.4 USB / ADB control

```bash
jpy device usb -s 172.25.0.251 -u admin -p 123456 --mode host --seat 3
jpy device usb -s 172.25.0.251 -u admin -p 123456 --mode device --seat 3
jpy device adb -s 172.25.0.251 -u admin -p 123456 --set on --seat 3
jpy device adb -s 172.25.0.251 -u admin -p 123456 --set off --seat 3
```

### 1.5 Server status

```bash
jpy device status -s 172.25.0.251 -u admin -p 123456
jpy device status -s 172.25.0.251 -u admin -p 123456 --detail -o json
```

---

## 2. Middleware firmware upgrade: `device middleware upgrade`

This command uploads a local middleware firmware package and immediately triggers the middleware upgrade API.

### Syntax

```bash
jpy device middleware upgrade --file <firmware-file> -s <server> -u <user> -p <pass> [--required=true|false] [-o plain|json]
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--file` | Yes | Local firmware file path |
| `--required` | No | Whether to force the upgrade, default `true` |
| `-s, --server` | Yes | Middleware address |
| `-u, --user` | Yes | Username |
| `-p, --password` | Yes | Password |
| `-o, --output` | No | `plain` or `json` |

### Examples

```bash
jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456
jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456 -o json
jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456 --required=false
```

### Plain output example

```text
SERVER	172.25.0.251
PACKAGE_ID	123
REQUIRED	true
MESSAGE	upgrade submitted
STATUS	success
```

### JSON output example

```json
{"success":true,"server":"http://172.25.0.251","package_id":123,"required":true,"message":"upgrade submitted"}
```

### Internal flow

1. Log in to the middleware and obtain a token.
2. Upload the firmware to `/sys/upload`.
3. Read `package_id` from the upload response.
4. Call `/sys/update?required=<true|false>&id=<package_id>`.

---

## 3. Middleware ROM flashing: `device middleware rom`

These commands use middleware HTTP and Guard WebSocket APIs to manage ROM packages and trigger ROM flashing. This is different from the COM-based `flash run` flow.

The current implementation provides primitive commands: upload ROM package, list packages, start flashing, query status, and read detail logs. Automatic polling and keyword-based result judgement can be built on top of these commands later.

### 3.1 Upload ROM package

```bash
jpy device middleware rom upload --file ./rom.zip -s 172.25.0.251 -u admin -p 123456
jpy device middleware rom upload --file ./rom.zip -s 172.25.0.251 -u admin -p 123456 -o json
```

Flags:

| Flag | Required | Description |
|------|----------|-------------|
| `--file` | Yes | Local ROM package path |
| `-s, --server` | Yes | Middleware address |
| `-u, --user` | Yes | Username |
| `-p, --password` | Yes | Password |
| `-o, --output` | No | `plain` or `json` |

Plain output example:

```text
SERVER	172.25.0.251
FILE	rom.zip
MESSAGE	upload success
STATUS	success
```

JSON output example:

```json
{"success":true,"server":"http://172.25.0.251","file":"rom.zip","message":"upload success"}
```

Internal API: `POST /box/upload`.

### 3.2 List ROM packages

```bash
jpy device middleware rom list -s 172.25.0.251 -u admin -p 123456
jpy device middleware rom list -s 172.25.0.251 -u admin -p 123456 -o json
```

Plain output columns:

```text
NAME	MODEL	VERSION	DESC
```

JSON output shape:

```json
{"server":"http://172.25.0.251","total":1,"packages":[{"name":"1767856234","model":"xxx","version":"xxx","desc":"xxx"}]}
```

Use the returned `name` as `rom flash --image` in most cases.

Internal flow: Guard WebSocket `/box/guard`, request code `113`.

### 3.3 Start ROM flashing

```bash
jpy device middleware rom flash --seat 3 --sn ABC123 --image 1767856234 -s 172.25.0.251 -u admin -p 123456
jpy device middleware rom flash --seat 3 --sn ABC123 --image 1767856234 --mode 2 -s 172.25.0.251 -u admin -p 123456 -o json
```

Flags:

| Flag | Required | Description |
|------|----------|-------------|
| `--seat` | Yes | Seat number, must be greater than 0 |
| `--sn` | Yes | Device serial number |
| `--image` | Yes | ROM image ID, usually the `name` returned by `rom list` |
| `--mode` | No | Flash mode, default `2` |
| `-s, --server` | Yes | Middleware address |
| `-u, --user` | Yes | Username |
| `-p, --password` | Yes | Password |
| `-o, --output` | No | `plain` or `json` |

Plain output example:

```text
SERVER	172.25.0.251
SEAT	3
SN	ABC123
IMAGE	1767856234
MODE	2
MESSAGE	flash request submitted
STATUS	success
```

JSON output example:

```json
{"success":true,"server":"http://172.25.0.251","seat":3,"sn":"ABC123","image":"1767856234","mode":2,"message":"flash request submitted"}
```

Internal flow: Guard WebSocket `/box/guard`, request code `119`.

### 3.4 Query ROM flashing status

```bash
jpy device middleware rom status -s 172.25.0.251 -u admin -p 123456
jpy device middleware rom status -s 172.25.0.251 -u admin -p 123456 --seat 3
jpy device middleware rom status -s 172.25.0.251 -u admin -p 123456 --sn ABC123 -o json
```

Optional filters:

| Flag | Description |
|------|-------------|
| `--seat` | Only show one seat |
| `--sn` | Only show one device serial number |

Plain output columns:

```text
SEAT	SN	MODE	STATUS	SESSION	IMAGE	QUEUE	START	END	ERROR
```

JSON output shape:

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

`status` is the raw middleware status code. The CLI does not translate it so UI code can make decisions based on the original value.

Internal flow: Guard WebSocket `/box/guard`, request code `117`.

### 3.5 Read ROM flashing detail log

```bash
jpy device middleware rom detail --seat 3 --session session-id -s 172.25.0.251 -u admin -p 123456
jpy device middleware rom detail --seat 3 --session session-id -s 172.25.0.251 -u admin -p 123456 -o json
```

Flags:

| Flag | Required | Description |
|------|----------|-------------|
| `--seat` | Yes | Seat number, must be greater than 0 |
| `--session` | Yes | Flashing session ID, usually returned by `rom status` |
| `-s, --server` | Yes | Middleware address |
| `-u, --user` | Yes | Username |
| `-p, --password` | Yes | Password |
| `-o, --output` | No | `plain` or `json` |

Plain output example:

```text
SERVER	172.25.0.251
SEAT	3
SESSION	session-id
DETAIL
flash detail log...
```

JSON output example:

```json
{"success":true,"server":"http://172.25.0.251","seat":3,"session":"session-id","detail":"flash detail log..."}
```

Internal API: `GET /box/detail?id=<seat>&session=<session>`.

---

## 4. COM control: `com`

COM commands control the USB HUB controller board. This flow is separate from middleware ROM flashing.

```bash
jpy com list -o json
jpy com devices --port COM3 -o json
jpy com set-mode --port COM3 --mode hub --channel 5
jpy com set-mode --port COM3 --mode otg --channel 2-20
jpy com set-mode --port COM3 --mode hub --channel 1,2,3
jpy com restart --port COM3 --channel 3
jpy com restart --port COM3 --channel 2-20
```

Channel syntax:

| Syntax | Description |
|--------|-------------|
| `1` | Single channel |
| `1,2,3` | Multiple channels |
| `2-20` | Channel range |
| `0` | All channels |

---

## 5. COM-based flashing: `flash run`

This command is the traditional COM-based flashing flow. It combines middleware device operations, COM channel switching, and local flash script execution.

```bash
jpy flash run --com COM3 --ch 1 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y
jpy flash run --com COM3 --ch 1-10 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y
jpy flash run --com COM3 --ch 1,3,5,7 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y
jpy flash run --com COM3,COM4 --ch 1-20 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y
jpy flash run --com all --ch 1-20 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" -y
jpy flash run --com COM3 --ch 1-5 --mw 172.25.0.251 --ip-start 172.25.0.11 --script "C:/ai-services/rom/8se-20260309/002.cmd" --dry
```

Flow:

1. Check device status.
2. Send `reboot bootloader`.
3. Switch COM channel to HUB mode.
4. Wait for fastboot device.
5. Run the flash script.
6. Switch back to OTG mode.

---

## 6. Remote file transfer: `file`

```bash
jpy file push ./rom.zip --remote 192.168.1.100:9090 --dest D:\flash\rom.zip
jpy file pull "https://example.com/rom.zip" --remote 192.168.1.100:9090 --dest D:\flash\rom.zip
jpy file push ./large.zip --remote 192.168.1.100:9090 --timeout 3600
```

---

## 7. Remote JPY CLI update: `update`

```bash
jpy update ./jpy-windows-amd64.exe --remote 192.168.1.100:9090
jpy update https://example.com/jpy.exe --remote 192.168.1.100:9090
```

This command updates the remote `jpy` executable, not middleware firmware. To upgrade middleware firmware, use:

```bash
jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456
```

---

## 8. Stress test: `stress user`

```bash
jpy stress user -s wss://home.accjs.cn/ws -k YOUR_SECRET_KEY -c config.json
jpy stress user -k YOUR_SECRET_KEY -c config.json --device 123,456,789 --loop 3 --interval 5m
jpy stress user -k YOUR_SECRET_KEY -c config.json --loop 0 --interval 3m
jpy stress user -k YOUR_SECRET_KEY -c config.json --loop 0 --debug
jpy stress user -k YOUR_SECRET_KEY -c config.json --timeout 15m --log-dir /var/log/stress
```

---

## 9. Remote shell: `shell`

```bash
jpy shell --remote 192.168.1.100:9090 -c "dir C:\Users"
jpy shell --remote 192.168.1.100:9090 -c "fastboot devices" --timeout 60
jpy shell --remote 192.168.1.100:9090 -c "long-running-command" --async --timeout 900
jpy shell --remote 192.168.1.100:9090 --task <task_id>
jpy shell --remote 192.168.1.100:9090 --tasks
jpy shell --remote 192.168.1.100:9090 --kill <task_id>
```

---

## 10. Server mode: `server`

Start on the controlled machine:

```bash
jpy server --port 9090
```

Notes:

- Default port: `9090`
- The process keeps running and listens on the port
- Protocol: HTTP
- Listen address: `:port`, reachable as `http://controlled-host:port` from LAN machines
- `server` itself cannot be started through remote execution

Common HTTP APIs. See [`HTTP_REMOTE_CONTROL.md`](./HTTP_REMOTE_CONTROL.md) for direct HTTP usage without `jpy --remote`:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/version` | GET | Version information |
| `/exec` | POST | Execute a CLI command synchronously |
| `/exec/async` | POST | Execute a CLI command asynchronously |
| `/shell` | POST | Execute a system command synchronously |
| `/shell/async` | POST | Execute a system command asynchronously |
| `/shell/task` | GET | Query async task |
| `/shell/tasks` | GET | List async tasks |
| `/shell/kill` | GET | Kill async task |
| `/file/upload` | POST | Upload file |
| `/file/download` | POST | Download file |
| `/file/chunk/init` | POST | Initialize chunk upload |
| `/file/chunk/upload` | POST | Upload chunk |
| `/file/chunk/complete` | POST | Complete chunk upload |

---

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Failure |
| `124` | Timeout |
| `137` | Killed |
