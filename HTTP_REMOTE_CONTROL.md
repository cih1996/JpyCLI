# HTTP 远程控制使用文档

本文档说明如何直接通过 HTTP 请求控制被控端 `jpy server`，不依赖 `jpy --remote` 命令。

## 1. 启动被控端服务

在被控端机器上启动：

```bash
jpy server --port 9090
```

说明：

- 默认端口：`9090`
- 通讯协议：HTTP，不是 WebSocket
- 监听地址：`:端口`
- 局域网访问地址：`http://被控端IP:9090`
- 服务进程会一直挂起监听

健康检查：

```bash
curl http://被控端IP:9090/health
```

## 2. HTTP 接口总览

| 接口 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/version` | GET | 查看被控端 jpy 版本 |
| `/exec` | POST | 同步执行一条 jpy CLI 命令 |
| `/exec/async` | POST | 异步执行一条 jpy CLI 命令 |
| `/shell` | POST | 同步执行系统 shell 命令 |
| `/shell/async` | POST | 异步执行系统 shell 命令 |
| `/shell/task?id=<task_id>` | GET | 查询异步任务 |
| `/shell/tasks` | GET | 列出异步任务 |
| `/shell/kill?id=<task_id>` | GET | 终止异步任务 |
| `/file/upload` | POST | 上传文件到被控端 |
| `/file/download` | POST | 让被控端从 URL 下载文件 |
| `/file/chunk/init` | POST | 初始化分片上传 |
| `/file/chunk/upload` | POST | 上传分片 |
| `/file/chunk/complete` | POST | 合并分片 |

## 3. 执行 jpy CLI 命令

### 3.1 同步执行：`POST /exec`

请求体：

```json
{
  "args": ["device", "list", "-s", "172.25.0.251", "-u", "admin", "-p", "123456", "-o", "json"]
}
```

curl：

```bash
curl -X POST http://被控端IP:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","list","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```

响应结构：

```json
{
  "exit_code": 0,
  "stdout": "命令标准输出",
  "stderr": "命令错误输出"
}
```

说明：

- `args` 就是原 CLI 命令去掉开头 `jpy` 后的参数数组。
- 业务是否成功看 `exit_code`，`0` 表示成功。
- 禁止在 `args` 里传 `server` 和 `--remote`。
- 请求体上限约 1MB，不适合直接塞文件内容。

### 3.2 异步执行：`POST /exec/async`

适合刷机、压力测试、批量操作等长任务。

请求体：

```json
{
  "args": ["flash", "run", "--com", "COM3", "--ch", "1-10", "--mw", "172.25.0.251", "--ip-start", "172.25.0.11", "--script", "C:/flash/002.cmd", "-y"],
  "timeout": 0
}
```

curl：

```bash
curl -X POST http://被控端IP:9090/exec/async \
  -H "Content-Type: application/json" \
  -d '{"args":["flash","run","--com","COM3","--ch","1-10","--mw","172.25.0.251","--ip-start","172.25.0.11","--script","C:/flash/002.cmd","-y"],"timeout":0}'
```

响应结构：

```json
{
  "task_id": "任务ID",
  "status": "running"
}
```

`timeout` 说明：

| 值 | 含义 |
|----|------|
| 不传或小于 0 | 默认 600 秒 |
| `0` | 不限制时长 |
| 大于 0 | 指定超时秒数 |

## 4. 中间件固件升级 HTTP 调用

### 4.1 固件文件已经在被控端

```bash
curl -X POST http://被控端IP:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","middleware","upgrade","--file","D:/jpy/firmware.bin","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```

### 4.2 固件文件在主控端，需要先上传

先上传到被控端：

```bash
curl -X POST http://被控端IP:9090/file/upload \
  -F "file=@./firmware.bin" \
  -F "dest=D:/jpy/firmware.bin"
```

再执行升级：

```bash
curl -X POST http://被控端IP:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","middleware","upgrade","--file","D:/jpy/firmware.bin","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```

注意：`--file` 路径是被控端机器上的路径。

## 5. 中间件 ROM 接口刷机 HTTP 调用

### 5.1 上传 ROM 包

如果 ROM 包在主控端，先上传到被控端：

```bash
curl -X POST http://被控端IP:9090/file/upload \
  -F "file=@./rom.zip" \
  -F "dest=D:/jpy/rom.zip"
```

再让被控端上传 ROM 到中间件：

```bash
curl -X POST http://被控端IP:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","middleware","rom","upload","--file","D:/jpy/rom.zip","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```

### 5.2 查看 ROM 包列表

```bash
curl -X POST http://被控端IP:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","middleware","rom","list","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```

### 5.3 发起 ROM 刷机

```bash
curl -X POST http://被控端IP:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","middleware","rom","flash","--seat","3","--sn","ABC123","--image","1767856234","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```

### 5.4 查询 ROM 刷机状态

```bash
curl -X POST http://被控端IP:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","middleware","rom","status","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```

按盘位过滤：

```bash
curl -X POST http://被控端IP:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","middleware","rom","status","--seat","3","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```

### 5.5 查询 ROM 刷机详情

```bash
curl -X POST http://被控端IP:9090/exec \
  -H "Content-Type: application/json" \
  -d '{"args":["device","middleware","rom","detail","--seat","3","--session","session-id","-s","172.25.0.251","-u","admin","-p","123456","-o","json"]}'
```

## 6. 查询和管理异步任务

### 6.1 查询单个任务

```bash
curl "http://被控端IP:9090/shell/task?id=<task_id>"
```

响应结构：

```json
{
  "task_id": "任务ID",
  "status": "running|done|failed",
  "exit_code": 0,
  "stdout": "任务输出",
  "stderr": "错误输出",
  "elapsed": "耗时",
  "command": "实际执行命令"
}
```

### 6.2 列出所有任务

```bash
curl http://被控端IP:9090/shell/tasks
```

### 6.3 终止任务

```bash
curl "http://被控端IP:9090/shell/kill?id=<task_id>"
```

状态判断：

| status | exit_code | 含义 |
|--------|-----------|------|
| `running` | - | 进行中 |
| `done` | `0` | 成功完成 |
| `done` | 非 0 | 命令执行失败 |
| `failed` | `124` | 超时 |
| `failed` | `137` | 被终止 |

## 7. 执行系统 Shell 命令

### 7.1 同步 Shell：`POST /shell`

```bash
curl -X POST http://被控端IP:9090/shell \
  -H "Content-Type: application/json" \
  -d '{"cmd":"fastboot devices","timeout":60}'
```

### 7.2 异步 Shell：`POST /shell/async`

```bash
curl -X POST http://被控端IP:9090/shell/async \
  -H "Content-Type: application/json" \
  -d '{"cmd":"cd D:/flash && flash.bat","timeout":900}'
```

## 8. 文件传输接口

### 8.1 上传文件：`POST /file/upload`

```bash
curl -X POST http://被控端IP:9090/file/upload \
  -F "file=@./local.zip" \
  -F "dest=D:/jpy/local.zip"
```

说明：

- `file`：要上传的文件。
- `dest`：被控端保存路径；不传则使用原文件名。
- 非绝对路径会保存到被控端系统临时目录。
- 单次上传上限 5GB。

### 8.2 让被控端下载文件：`POST /file/download`

```bash
curl -X POST http://被控端IP:9090/file/download \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/rom.zip","dest":"D:/jpy/rom.zip"}'
```

## 9. 分片上传接口

适合大文件或不稳定网络。

流程：

```text
1. POST /file/chunk/init      初始化，拿 session_id
2. POST /file/chunk/upload    循环上传每个分片
3. POST /file/chunk/complete  合并分片
```

### 9.1 初始化

```bash
curl -X POST http://被控端IP:9090/file/chunk/init \
  -H "Content-Type: application/json" \
  -d '{"filename":"rom.zip","dest":"D:/jpy/rom.zip","total_size":104857600,"chunk_size":1048576,"total_chunk":100}'
```

### 9.2 上传分片

```bash
curl -X POST http://被控端IP:9090/file/chunk/upload \
  -F "session_id=<session_id>" \
  -F "chunk_index=0" \
  -F "chunk=@./chunk_0"
```

### 9.3 完成合并

```bash
curl -X POST http://被控端IP:9090/file/chunk/complete \
  -H "Content-Type: application/json" \
  -d '{"session_id":"<session_id>"}'
```

## 10. 版本信息

```bash
curl http://被控端IP:9090/version
```

## 11. 常见注意事项

1. HTTP 远程控制执行的是被控端上的 `jpy` 程序。
2. `/exec` 的 `args` 不要包含 `jpy` 本身，只写后面的参数。
3. `/exec` 和 `/exec/async` 禁止执行 `server`，也禁止包含 `--remote`。
4. 所有 `--file` 路径都是被控端路径。
5. 主控端本地文件要先通过 `/file/upload` 上传到被控端，再用被控端路径执行命令。
6. 推荐给 UI 调用的 CLI 命令都加 `-o json`，这样结果会在 `stdout` 中以 JSON 字符串返回。
7. 远程控制通道是 HTTP；部分业务命令内部可能再用 HTTP 或 WebSocket 连接中间件。
