package device

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type upgradeResult struct {
	Success   bool   `json:"success"`
	Server    string `json:"server"`
	PackageID int    `json:"package_id,omitempty"`
	Required  bool   `json:"required"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

type sysUploadResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

type sysUpdateResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func NewMiddlewareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "middleware",
		Short: "中间件维护命令",
	}

	cmd.AddCommand(newUpgradeCmd())
	cmd.AddCommand(newROMCmd())
	return cmd
}

func newUpgradeCmd() *cobra.Command {
	var (
		filePath string
		required bool
	)

	cmd := &cobra.Command{
		Use:   "upgrade --file <firmware-file>",
		Short: "上传固件并执行中间件升级",
		Long: `上传本地固件到中间件，然后调用升级接口执行更新。

示例:
  jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456
  jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456 -o json
  jpy device middleware upgrade --file ./firmware.bin -s 172.25.0.251 -u admin -p 123456 --required=false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("必须指定 --file 参数")
			}

			creds, err := resolveCredentials()
			if err != nil {
				return err
			}

			result, err := runMiddlewareUpgrade(creds.ServerURL, creds.Username, creds.Token, filePath, required)
			if err != nil {
				return err
			}

			switch flagOutput {
			case "json":
				data, _ := json.Marshal(result)
				fmt.Println(string(data))
			default:
				fmt.Printf("SERVER\t%s\n", strings.TrimPrefix(strings.TrimPrefix(result.Server, "https://"), "http://"))
				fmt.Printf("PACKAGE_ID\t%d\n", result.PackageID)
				fmt.Printf("REQUIRED\t%v\n", result.Required)
				if result.Message != "" {
					fmt.Printf("MESSAGE\t%s\n", result.Message)
				}
				fmt.Println("STATUS\tsuccess")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "本地固件文件路径（必填）")
	cmd.Flags().BoolVar(&required, "required", true, "是否强制执行升级")
	return cmd
}

func runMiddlewareUpgrade(serverURL, username, token, filePath string, required bool) (*upgradeResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开固件文件失败: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取固件文件信息失败: %v", err)
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("--file 不能是目录")
	}

	packageID, err := uploadSystemPackage(serverURL, username, token, filePath, file)
	if err != nil {
		return nil, err
	}

	msg, err := updateSystemPackage(serverURL, token, packageID, required)
	if err != nil {
		return nil, err
	}

	return &upgradeResult{
		Success:   true,
		Server:    serverURL,
		PackageID: packageID,
		Required:  required,
		Message:   msg,
	}, nil
}

func uploadSystemPackage(serverURL, username, token, filePath string, file *os.File) (int, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	errCh := make(chan error, 1)
	filename := filepath.Base(filePath)

	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			errCh <- err
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(serverURL, "/")+"/sys/upload", pr)
	if err != nil {
		return 0, fmt.Errorf("创建上传请求失败: %v", err)
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Cookie", fmt.Sprintf("username=%s;token=%s", username, token))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	writeErr := <-errCh
	if err != nil {
		if writeErr != nil {
			return 0, fmt.Errorf("上传固件失败: %v", writeErr)
		}
		return 0, fmt.Errorf("上传固件失败: %v", err)
	}
	defer resp.Body.Close()

	if writeErr != nil {
		return 0, fmt.Errorf("写入固件数据失败: %v", writeErr)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("读取上传响应失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("上传固件失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result sysUploadResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("解析上传响应失败: %s", string(body))
	}
	if result.Code != 200 {
		if result.Msg == "" {
			result.Msg = "上传系统包失败"
		}
		return 0, fmt.Errorf(result.Msg)
	}

	var packageID int
	if err := json.Unmarshal(result.Data, &packageID); err != nil {
		var packageIDFloat float64
		if err2 := json.Unmarshal(result.Data, &packageIDFloat); err2 != nil {
			return 0, fmt.Errorf("解析 package_id 失败: %s", string(result.Data))
		}
		packageID = int(packageIDFloat)
	}
	return packageID, nil
}

func updateSystemPackage(serverURL, token string, packageID int, required bool) (string, error) {
	url := fmt.Sprintf("%s/sys/update?required=%t&id=%d", strings.TrimRight(serverURL, "/"), required, packageID)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("创建升级请求失败: %v", err)
	}
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("执行升级请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取升级响应失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("执行升级失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result sysUpdateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析升级响应失败: %s", string(body))
	}
	if result.Code != 200 {
		if result.Msg == "" {
			result.Msg = "执行系统升级失败"
		}
		return "", fmt.Errorf(result.Msg)
	}

	if result.Msg == "" {
		result.Msg = "upgrade submitted"
	}
	return result.Msg, nil
}
