package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func NewCompletionInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion-install",
		Short: "自动安装 Shell 补全脚本 (bash/zsh/powershell)",
		Long: `将 jpy 命令的自动补全脚本安装到当前用户的 Shell 配置文件中。
支持 bash (.bashrc), zsh (.zshrc) 和 Windows PowerShell ($PROFILE)。`,
		Run: func(cmd *cobra.Command, args []string) {
			shell := filepath.Base(os.Getenv("SHELL"))
			if runtime.GOOS == "windows" {
				shell = "powershell"
			} else if shell == "" || shell == "." {
				shell = "zsh" // Default to zsh on Mac/Linux if detection fails
			}

			var configFile string
			var targetBlock string

			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Printf("错误: 无法获取用户主目录: %v\n", err)
				return
			}

			if runtime.GOOS == "windows" {
				// PowerShell Profile logic
				// We can't easily get $PROFILE from Go without executing powershell.
				// But standard location is usually:
				// Documents\WindowsPowerShell\Microsoft.PowerShell_profile.ps1 (Windows PowerShell)
				// Documents\PowerShell\Microsoft.PowerShell_profile.ps1 (PowerShell Core)

				// Let's try to find Documents folder.
				// On Windows, UserHomeDir is usually C:\Users\Name
				docs := filepath.Join(home, "Documents")

				// Check for PowerShell Core first (pwsh)
				// pwshDir := filepath.Join(docs, "PowerShell")
				winPsDir := filepath.Join(docs, "WindowsPowerShell")

				// We will try to install to BOTH if directories exist, or create WindowsPowerShell one as default.
				// For simplicity, let's target the Windows PowerShell one as it's most common for built-in usage,
				// unless we detect Core.

				// Actually, let's just use the standard Windows PowerShell path for now as it's safer.
				configFile = filepath.Join(winPsDir, "Microsoft.PowerShell_profile.ps1")

				// Ensure directory exists
				_ = os.MkdirAll(winPsDir, 0755)

				targetBlock = `
# JPY Completion
if (Get-Command jpy -ErrorAction SilentlyContinue) {
    jpy completion powershell | Out-String | Invoke-Expression
}
`
				shell = "powershell"
			} else if shell == "zsh" {
				configFile = filepath.Join(home, ".zshrc")
				targetBlock = `
# JPY Completion
if ! command -v compdef >/dev/null 2>&1; then
    autoload -U compinit && compinit
fi
source <(jpy completion zsh)
`
			} else if shell == "bash" {
				configFile = filepath.Join(home, ".bashrc")
				targetBlock = `
# JPY Completion
source <(jpy completion bash)
`
			} else {
				fmt.Printf("不支持的 Shell: %s。请手动安装补全脚本。\n", shell)
				return
			}

			// Read file
			contentBytes, err := os.ReadFile(configFile)
			var content string
			if err == nil {
				content = string(contentBytes)
			}

			// Remove legacy installation if exists
			legacyCmd := fmt.Sprintf("\n# JPY Completion\nsource <(jpy completion %s)\n", shell)
			if strings.Contains(content, legacyCmd) {
				content = strings.ReplaceAll(content, legacyCmd, "")
			}

			// Check if already installed (exact match of new block)
			if strings.Contains(content, strings.TrimSpace(targetBlock)) {
				fmt.Printf("自动补全脚本已安装在 %s 中。\n请执行 'source %s' 使其立即生效。\n", configFile, configFile)
				return
			}

			// Write updated content
			// If we replaced legacy content, we need to overwrite the file.
			// If it's a new install, we append.

			// Simplest approach: Write full content + new block
			// But wait, if we stripped legacyCmd, 'content' is modified.
			// We should just write 'content' + 'targetBlock' if we modified it?
			// Or just Append if we didn't modify it?

			// Let's rewrite the file if we modified it, otherwise append.

			needRewrite := strings.Contains(string(contentBytes), legacyCmd)

			if needRewrite {
				// Append new block to cleaned content
				newFullContent := strings.TrimRight(content, "\n") + "\n" + strings.TrimSpace(targetBlock) + "\n"
				if err := os.WriteFile(configFile, []byte(newFullContent), 0644); err != nil {
					fmt.Printf("错误: 写入配置文件失败: %v\n", err)
					return
				}
			} else {
				// Append mode
				f, err := os.OpenFile(configFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Printf("错误: 无法打开配置文件 %s: %v\n", configFile, err)
					return
				}
				defer f.Close()

				if _, err := f.WriteString("\n" + strings.TrimSpace(targetBlock) + "\n"); err != nil {
					fmt.Printf("错误: 写入配置文件失败: %v\n", err)
					return
				}
			}

			fmt.Printf("成功! 自动补全脚本已添加到 %s\n", configFile)
			if runtime.GOOS == "windows" {
				fmt.Printf("注意: 如果遇到 '在此系统上禁止运行脚本' 的错误，请以管理员身份运行 PowerShell 并执行:\n\n  Set-ExecutionPolicy RemoteSigned -Scope CurrentUser\n\n")
				fmt.Printf("然后执行以下命令使其生效:\n\n  . \"%s\"\n\n", configFile)
			} else {
				fmt.Printf("请执行以下命令使其立即生效:\n\n  source %s\n\n", configFile)
			}
		},
	}
	return cmd
}
