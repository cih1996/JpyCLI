package device

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/terminal"
	"jpy-cli/pkg/middleware/model"
	"os"
	"strings"
	"time"
)

// shellExecViaF14 通过 f=14 向指定设备发送 shell 命令
func shellExecViaF14(server connector.ServerInfo, seat int, command string) (*model.ShellResult, error) {
	ws, err := connector.ConnectGuard(server)
	if err != nil {
		return nil, fmt.Errorf("f=14 Guard 连接失败: %v", err)
	}
	defer ws.Close()

	resp, err := ws.SendRequestToDevice(model.FuncBatchCommand, command, uint64(seat))
	if err != nil {
		return nil, fmt.Errorf("f=14 发送失败: %v", err)
	}
	if resp.Code != nil && *resp.Code != 0 {
		msg := "unknown"
		if resp.Msg != nil {
			msg = *resp.Msg
		}
		return nil, fmt.Errorf("f=14 执行失败 (code %d): %s", *resp.Code, msg)
	}

	output := ""
	switch v := resp.Data.(type) {
	case string:
		output = v
	case []byte:
		output = string(v)
	}

	return &model.ShellResult{Output: output}, nil
}

// shellExecViaTerminal 通过 Terminal 通道执行 shell 命令（老系统兼容）
func shellExecViaTerminal(server connector.ServerInfo, seat int, command, outputMode string) error {
	ws, err := connector.ConnectDeviceTerminal(server, int64(seat))
	if err != nil {
		return fmt.Errorf("Terminal 通道连接失败: %v", err)
	}
	defer ws.Close()

	term := terminal.NewTerminalSession(ws, int64(seat))
	if err := term.Init(); err != nil {
		return fmt.Errorf("Terminal 初始化失败: %v", err)
	}

	if err := term.WaitForReady(8 * time.Second); err != nil {
		// 超时仍尝试
	}

	// 排空缓冲
	for {
		select {
		case <-term.Output:
		default:
			goto Flushed
		}
	}
Flushed:

	if err := term.Exec(command); err != nil {
		return fmt.Errorf("Terminal 发送命令失败: %v", err)
	}

	var buf strings.Builder
	deadline := time.After(15 * time.Second)
	lastChunk := time.Now()
	for {
		select {
		case chunk := <-term.Output:
			buf.WriteString(chunk)
			lastChunk = time.Now()
			cleaned := termStripANSI(buf.String())
			if termHasPrompt(cleaned) {
				goto CollectDone
			}
		case <-deadline:
			goto CollectDone
		default:
			if time.Since(lastChunk) > 1500*time.Millisecond && buf.Len() > 0 {
				goto CollectDone
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
CollectDone:

	rawOutput := termStripANSI(buf.String())
	rawOutput = termStripEcho(rawOutput, command)

	switch outputMode {
	case "json":
		b, _ := json.Marshal(map[string]interface{}{
			"server": server.URL, "seat": seat, "command": command,
			"output": rawOutput, "success": true,
		})
		fmt.Println(string(b))
	default:
		if rawOutput != "" {
			fmt.Print(rawOutput)
			if rawOutput[len(rawOutput)-1] != '\n' {
				fmt.Println()
			}
		}
	}
	return nil
}

// shellOutputResult 输出 shell 执行结果
func shellOutputResult(result *model.ShellResult, server string, seat int, command string, outputMode string) error {
	switch outputMode {
	case "json":
		out := map[string]interface{}{
			"server": server, "seat": seat, "command": command,
			"output": result.Output, "success": true,
		}
		if result.ExitCode != nil {
			out["exit_code"] = *result.ExitCode
		}
		if result.Error != nil && *result.Error != "" {
			out["error"] = *result.Error
			out["success"] = false
		}
		b, _ := json.Marshal(out)
		fmt.Println(string(b))
		if result.ExitCode != nil && *result.ExitCode != 0 {
			os.Exit(*result.ExitCode)
		}
	default:
		if result.Error != nil && *result.Error != "" {
			fmt.Fprintf(os.Stderr, "执行错误: %s\n", *result.Error)
		}
		if result.Output != "" {
			fmt.Print(result.Output)
			if result.Output[len(result.Output)-1] != '\n' {
				fmt.Println()
			}
		}
		if result.ExitCode != nil && *result.ExitCode != 0 {
			os.Exit(*result.ExitCode)
		}
	}
	return nil
}

// termStripANSI 去除 VT100/ANSI 转义序列
func termStripANSI(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && (s[i] < 'A' || s[i] > 'Z') && (s[i] < 'a' || s[i] > 'z') {
				i++
			}
			if i < len(s) {
				i++
			}
		} else if s[i] == '\r' {
			i++
		} else {
			out.WriteByte(s[i])
			i++
		}
	}
	return out.String()
}

func termHasPrompt(s string) bool {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, "$") || strings.HasSuffix(line, "#") ||
			strings.Contains(line, "$ ") || strings.Contains(line, "# ") {
			return true
		}
		break
	}
	return false
}

func termStripEcho(output, command string) string {
	lines := strings.Split(output, "\n")
	var result []string
	echoSkipped := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !echoSkipped && strings.Contains(line, command) {
			echoSkipped = true
			continue
		}
		if termHasPrompt(trimmed) && trimmed != "" {
			continue
		}
		result = append(result, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}
