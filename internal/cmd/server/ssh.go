package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gliderlabs/ssh"
	"github.com/spf13/cobra"
)

func NewSSHServerCmd() *cobra.Command {
	var port int
	var password string

	cmd := &cobra.Command{
		Use:   "ssh",
		Short: "启动内置 SSH 服务端 (跳板机模式)",
		Long: `启动一个内置的 SSH 服务端，允许远程通过 SSH 连接并在本地 shell 中操作 JPY CLI。
这对于在内网机器上运行 CLI，并通过 frp 等工具暴露给远程使用非常有用。

连接方式:
  ssh -p <port> jpy@<ip>

交互体验:
  该服务会为每个连接分配一个伪终端 (PTY)，因此支持所有的 TUI 交互效果 (如表格、选择列表等)。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Password Handling
			if password == "" {
				randBytes := make([]byte, 4)
				if _, err := rand.Read(randBytes); err != nil {
					return fmt.Errorf("生成随机密码失败: %v", err)
				}
				password = hex.EncodeToString(randBytes)
			}

			fmt.Printf("Starting SSH Server on port %d...\n", port)
			fmt.Printf("Use the following command to connect:\n")
			fmt.Printf("  ssh -p %d jpy@<this-machine-ip>\n", port)
			fmt.Printf("Password: %s\n", password)

			// 2. Configure SSH Server
			s := &ssh.Server{
				Addr: fmt.Sprintf(":%d", port),
				PasswordHandler: func(ctx ssh.Context, pass string) bool {
					return pass == password
				},
				Handler: func(s ssh.Session) {
					// 3. PTY Handling
					cmd := exec.Command(getUserShell())

					// Set environment variables
					cmd.Env = append(os.Environ(), "TERM=xterm-256color")

					// Ensure jpy executable is in PATH
					if exe, err := os.Executable(); err == nil {
						exeDir := filepath.Dir(exe)
						currentPath := os.Getenv("PATH")
						newPath := fmt.Sprintf("PATH=%s%c%s", exeDir, os.PathListSeparator, currentPath)
						cmd.Env = append(cmd.Env, newPath)
					}

					ptyReq, winCh, isPty := s.Pty()

					var ptyIO io.ReadWriteCloser
					var err error

					if isPty {
						cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
						ptyIO, err = StartPTY(cmd)
						if err == nil {
							// Initialize PTY size immediately!
							w := ptyReq.Window.Width
							h := ptyReq.Window.Height
							if w == 0 {
								w = 80
							}
							if h == 0 {
								h = 24
							}
							ResizePTY(ptyIO, w, h)
						}
					}

					if !isPty || err != nil {
						// Fallback: No PTY requested OR PTY start failed
						if err != nil {
							io.WriteString(s, fmt.Sprintf("Warning: PTY start failed (%v).\r\n", err))
							io.WriteString(s, "Falling back to basic shell. Please try 'ssh -T ...' for better compatibility if you see no output.\r\n")
						}

						cmd.Stdin = s
						cmd.Stdout = s
						cmd.Stderr = s

						if err := cmd.Run(); err != nil {
							io.WriteString(s, fmt.Sprintf("Shell exited with error: %v\r\n", err))
						}
						s.Exit(0)
						return
					}

					// Handle Window Resize
					go func() {
						for win := range winCh {
							ResizePTY(ptyIO, win.Width, win.Height)
						}
					}()

					// Setup Stdin/Stdout/Stderr

					// 1. Copy from SSH Session to PTY (Input)
					// Run in background so we don't block here.
					// We want the session to close when the PROCESS exits, not when the USER disconnects.
					go func() {
						io.Copy(ptyIO, s)
					}()

					// 2. Copy from PTY to SSH Session (Output)
					// Block here! This waits for the PTY to close (which happens when the process exits).
					io.Copy(s, ptyIO)

					// Clean up
					if cmd.Process != nil {
						cmd.Process.Kill()
					}
					ptyIO.Close()
					s.Exit(0)
				},
			}

			// 4. Start Server
			return s.ListenAndServe()
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 2222, "SSH 监听端口")
	cmd.Flags().StringVarP(&password, "password", "P", "", "SSH 连接密码 (留空则随机生成)")

	return cmd
}
