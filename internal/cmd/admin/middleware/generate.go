package middleware

import (
	"bufio"
	"fmt"
	"jpy-cli/pkg/admin-middleware/api"
	"jpy-cli/pkg/admin-middleware/service"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "批量生成授权码",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Ensure Login
			adminCfg, err := service.EnsureLoggedIn()
			if err != nil {
				return err
			}

			client := api.NewClient(adminCfg.Token)
			reader := bufio.NewReader(os.Stdin)

			// 2. Interactive Input
			fmt.Println("=== 批量生成授权码 ===")

			// Prefix
			fmt.Print("输入前缀 (默认 'CS-JPY-'): ")
			prefix, _ := reader.ReadString('\n')
			prefix = strings.TrimSpace(prefix)
			if prefix == "" {
				prefix = "CS-JPY-"
			}

			// Start Number
			fmt.Print("输入起始编号 (例如 23201): ")
			startStr, _ := reader.ReadString('\n')
			startStr = strings.TrimSpace(startStr)
			startNum, err := strconv.Atoi(startStr)
			if err != nil {
				return fmt.Errorf("无效的起始编号")
			}

			// Count
			fmt.Print("输入生成数量 (例如 10): ")
			countStr, _ := reader.ReadString('\n')
			countStr = strings.TrimSpace(countStr)
			count, err := strconv.Atoi(countStr)
			if err != nil {
				return fmt.Errorf("无效的数量")
			}

			// 3. Confirm
			fmt.Printf("\n即将生成 %d 个授权码，从 '%s%d' 到 '%s%d'。是否继续? [y/N]: ",
				count, prefix, startNum, prefix, startNum+count-1)
			confirm, _ := reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))
			if confirm != "y" && confirm != "yes" {
				fmt.Println("操作已取消。")
				return nil
			}

			// 4. Execute
			successCount := 0
			failCount := 0

			for i := 0; i < count; i++ {
				currentNum := startNum + i
				name := fmt.Sprintf("%s%d", prefix, currentNum)

				fmt.Printf("[%d/%d] 正在生成 %s... ", i+1, count, name)
				err := client.GenerateAuthCode(name)
				if err != nil {
					fmt.Printf("失败: %v\n", err)
					failCount++
				} else {
					// Fetch the SerialNumber to display it
					serial, searchErr := client.SearchAuthCode(name)
					if searchErr == nil {
						fmt.Printf("成功 -> %s\n", serial)
					} else {
						fmt.Printf("成功 (但获取序列号失败: %v)\n", searchErr)
					}
					successCount++
				}
			}

			fmt.Printf("\n完成。成功: %d, 失败: %d\n", successCount, failCount)
			return nil
		},
	}

	return cmd
}
