package log

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func NewLogCmd() *cobra.Command {
	var follow bool
	var grep string
	var lines int

	cmd := &cobra.Command{
		Use:   "log",
		Short: "查看和实时跟踪日志",
		Long: `查看 jpy CLI 的运行日志。
支持实时跟踪 (-f) 和关键字过滤 (--grep)。
默认读取最后 50 行。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("无法获取用户主目录: %v", err)
			}
			logPath := filepath.Join(home, ".jpy", "logs", "jpy.log")

			if _, err := os.Stat(logPath); os.IsNotExist(err) {
				fmt.Println("日志文件不存在:", logPath)
				return nil
			}

			file, err := os.Open(logPath)
			if err != nil {
				return err
			}
			defer file.Close()

			// Initial read (tail logic)
			// Simplistic approach: Read whole file or seek near end.
			// For simplicity and "tail" behavior, let's just read all and print last N lines.
			// Optimization: use a ring buffer to store only last N lines
			ringBuffer := make([]string, 0, lines)
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if grep != "" && !strings.Contains(line, grep) {
					continue
				}
				if len(ringBuffer) < lines {
					ringBuffer = append(ringBuffer, line)
				} else {
					ringBuffer = append(ringBuffer[1:], line)
				}
			}

			for _, line := range ringBuffer {
				fmt.Println(line)
			}

			if !follow {
				return nil
			}

			// Follow logic
			// Seek to end of what we read (or just file end if we didn't filter?)
			// If we filtered, we are at EOF anyway.

			// We need to keep reading.
			// Since scanner stops at EOF, we need a loop.
			// Re-open or use Seek?
			// `tail -f` implementation usually involves:
			// 1. Read to EOF.
			// 2. Sleep.
			// 3. Read new data.

			// Reset position to current end (scanner consumed it)
			// Actually scanner might buffer.
			// Let's just use a simple reader loop from current offset.

			// To be robust, let's seek to current end (which we are at).

			reader := bufio.NewReader(file)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						time.Sleep(500 * time.Millisecond)
						continue
					}
					return err
				}

				line = strings.TrimRight(line, "\r\n")
				if grep != "" && !strings.Contains(line, grep) {
					continue
				}
				fmt.Println(line)
			}
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "实时跟踪日志更新")
	cmd.Flags().StringVar(&grep, "grep", "", "只输出包含指定关键字的行 (例如 'error')")
	cmd.Flags().IntVarP(&lines, "lines", "n", 50, "输出最后 N 行")

	return cmd
}
