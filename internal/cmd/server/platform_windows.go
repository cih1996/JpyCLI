//go:build windows

package server

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/UserExistsError/conpty"
)

func getUserShell() string {
	// Prefer PowerShell for better interactive experience (autocompletion, history, etc.)
	// Check for pwsh (PowerShell Core) first, then powershell (Windows PowerShell)
	if _, err := exec.LookPath("pwsh.exe"); err == nil {
		return "pwsh.exe"
	}
	if _, err := exec.LookPath("powershell.exe"); err == nil {
		return "powershell.exe"
	}

	// Fallback to cmd.exe if PowerShell is not available
	shell := os.Getenv("COMSPEC")
	if shell == "" {
		return "cmd.exe"
	}
	return shell
}

func StartPTY(cmd *exec.Cmd) (io.ReadWriteCloser, error) {
	// Reconstruct command line
	cmdLine := cmd.Path
	if len(cmd.Args) > 1 {
		// Simple joining, might need more robust escaping for complex args
		cmdLine = strings.Join(cmd.Args, " ")
	}

	// Auto-inject completion for PowerShell
	if strings.Contains(strings.ToLower(cmd.Path), "powershell") || strings.Contains(strings.ToLower(cmd.Path), "pwsh") {
		// We use -NoExit and -Command to load completion and stay in the shell

		// Let's append the injection logic if it's not already there
		if !strings.Contains(cmdLine, "-NoExit") {
			// PowerShell expects: pwsh.exe -NoExit -Command "..."
			// We need to be careful not to break existing args if any.
			// But usually cmd.Args is just [shell_path] here.

			injection := `try { if (Get-Command jpy -ErrorAction SilentlyContinue) { jpy completion powershell | Out-String | Invoke-Expression; Write-Host ' [Jpy Completion Loaded]' -ForegroundColor Green } } catch {}`
			cmdLine = fmt.Sprintf("%s -NoExit -Command \"%s\"", cmdLine, injection)
		}
	}

	cpty, err := conpty.Start(cmdLine)
	if err != nil {
		return nil, err
	}

	// conpty.Start spawns the process, but we need to ensure we don't leak it.
	// We can't easily attach the *exec.Cmd to the conpty process handle,
	// so we rely on conpty's Close() to kill the process.
	// Note: cmd.Process will be nil here because we didn't use cmd.Start().

	return cpty, nil
}

func ResizePTY(ptyIO io.ReadWriteCloser, w, h int) {
	if cpty, ok := ptyIO.(*conpty.ConPty); ok {
		cpty.Resize(w, h)
	}
}
