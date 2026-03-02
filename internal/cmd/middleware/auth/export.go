package auth

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/config"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "导出当前分组的服务器配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFile, _ := cmd.Flags().GetString("output")
			return runExport(outputFile)
		},
	}

	cmd.Flags().StringP("output", "o", "servers_export.json", "导出文件路径")
	return cmd
}

func runExport(outputFile string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	currentGroup := cfg.ActiveGroup
	if currentGroup == "" {
		currentGroup = "default"
	}

	servers, ok := cfg.Groups[currentGroup]
	if !ok || len(servers) == 0 {
		return fmt.Errorf("当前分组 '%s' 没有服务器配置", currentGroup)
	}

	// Format Output Path
	// If output file is just a filename, put it in current dir.
	// If it's a path, use it.
	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return fmt.Errorf("无效的文件路径: %v", err)
	}

	// Prepare data for export
	// We might want to export exactly what LocalServerConfig is, 
	// or maybe strip internal fields like Token/LastLoginTime if they are sensitive or irrelevant for import.
	// For "template" purposes (import.go compatible), we need URL, Group, Username, Password.
	// We can just marshal the struct as it has json tags.

	jsonData, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败: %v", err)
	}

	if err := os.WriteFile(absPath, jsonData, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("成功导出 %d 台服务器配置到: %s\n", len(servers), absPath)
	return nil
}
