package device

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	wsclient "jpy-cli/pkg/client/ws"
	"jpy-cli/pkg/middleware/model"

	"github.com/spf13/cobra"
)

type romUploadResult struct {
	Success bool   `json:"success"`
	Server  string `json:"server"`
	File    string `json:"file"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type romListResult struct {
	Server   string             `json:"server"`
	Total    int                `json:"total"`
	Packages []model.ROMPackage `json:"packages"`
}

type romFlashResult struct {
	Success bool   `json:"success"`
	Server  string `json:"server"`
	Seat    int    `json:"seat"`
	SN      string `json:"sn"`
	Image   string `json:"image"`
	Mode    int    `json:"mode"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type romStatusResult struct {
	Server string              `json:"server"`
	Total  int                 `json:"total"`
	Items  []romStatusItemView `json:"items"`
}

type romStatusItemView struct {
	Seat      int    `json:"seat"`
	SN        string `json:"sn"`
	Mode      int    `json:"mode"`
	Status    int    `json:"status"`
	Session   string `json:"session"`
	Image     string `json:"image"`
	QueueTime int64  `json:"queue_time"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	LastError string `json:"last_error,omitempty"`
}

type romDetailResult struct {
	Success bool   `json:"success"`
	Server  string `json:"server"`
	Seat    int    `json:"seat"`
	Session string `json:"session"`
	Detail  string `json:"detail,omitempty"`
	Error   string `json:"error,omitempty"`
}

func newROMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rom",
		Short: "中间件 ROM 包管理与刷机命令",
	}

	cmd.AddCommand(newROMUploadCmd())
	cmd.AddCommand(newROMListCmd())
	cmd.AddCommand(newROMFlashCmd())
	cmd.AddCommand(newROMStatusCmd())
	cmd.AddCommand(newROMDetailCmd())
	return cmd
}

func newROMUploadCmd() *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "upload --file <rom-file>",
		Short: "上传 ROM 包到中间件",
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("必须指定 --file 参数")
			}
			creds, err := resolveCredentials()
			if err != nil {
				return err
			}
			if err := uploadROMPackage(creds.ServerURL, creds.Username, creds.Token, filePath); err != nil {
				return err
			}
			result := romUploadResult{Success: true, Server: creds.ServerURL, File: filepath.Base(filePath), Message: "upload success"}
			return printROMUploadResult(result)
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "本地 ROM 文件路径（必填）")
	return cmd
}

func newROMListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出中间件内已上传的 ROM 包",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := resolveCredentials()
			if err != nil {
				return err
			}
			client, err := newGuardClient(creds.ServerURL, creds.Token)
			if err != nil {
				return err
			}
			defer client.Close()

			packages, err := getROMPackages(client)
			if err != nil {
				return err
			}
			return printROMListResult(romListResult{Server: creds.ServerURL, Total: len(packages), Packages: packages})
		},
	}
	return cmd
}

func newROMFlashCmd() *cobra.Command {
	var (
		seat  int
		sn    string
		image string
		mode  int
	)

	cmd := &cobra.Command{
		Use:   "flash --seat <seat> --sn <sn> --image <image>",
		Short: "发起中间件 ROM 刷机",
		RunE: func(cmd *cobra.Command, args []string) error {
			if seat <= 0 {
				return fmt.Errorf("必须指定 --seat，且必须大于 0")
			}
			if sn == "" {
				return fmt.Errorf("必须指定 --sn 参数")
			}
			if image == "" {
				return fmt.Errorf("必须指定 --image 参数")
			}
			creds, err := resolveCredentials()
			if err != nil {
				return err
			}
			client, err := newGuardClient(creds.ServerURL, creds.Token)
			if err != nil {
				return err
			}
			defer client.Close()

			msg, err := flashROM(client, seat, sn, image, mode)
			if err != nil {
				return err
			}
			result := romFlashResult{Success: true, Server: creds.ServerURL, Seat: seat, SN: sn, Image: image, Mode: mode, Message: msg}
			return printROMFlashResult(result)
		},
	}

	cmd.Flags().IntVar(&seat, "seat", 0, "盘位号（必填）")
	cmd.Flags().StringVar(&sn, "sn", "", "设备序列号（必填）")
	cmd.Flags().StringVar(&image, "image", "", "ROM 镜像 ID（必填，通常为 rom list 返回的 name）")
	cmd.Flags().IntVar(&mode, "mode", 2, "刷机模式（默认 2）")
	return cmd
}

func newROMStatusCmd() *cobra.Command {
	var (
		seat int
		sn   string
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "查询中间件 ROM 刷机状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := resolveCredentials()
			if err != nil {
				return err
			}
			client, err := newGuardClient(creds.ServerURL, creds.Token)
			if err != nil {
				return err
			}
			defer client.Close()

			items, err := queryFlashStatus(client)
			if err != nil {
				return err
			}
			filtered := make([]romStatusItemView, 0, len(items))
			for _, item := range items {
				if seat > 0 && item.Seat != seat {
					continue
				}
				if sn != "" && item.SN != sn {
					continue
				}
				filtered = append(filtered, item)
			}
			return printROMStatusResult(romStatusResult{Server: creds.ServerURL, Total: len(filtered), Items: filtered})
		},
	}

	cmd.Flags().IntVar(&seat, "seat", 0, "按盘位号过滤")
	cmd.Flags().StringVar(&sn, "sn", "", "按设备序列号过滤")
	return cmd
}

func newROMDetailCmd() *cobra.Command {
	var (
		seat    int
		session string
	)

	cmd := &cobra.Command{
		Use:   "detail --seat <seat> --session <session>",
		Short: "获取 ROM 刷机详情日志",
		RunE: func(cmd *cobra.Command, args []string) error {
			if seat <= 0 {
				return fmt.Errorf("必须指定 --seat，且必须大于 0")
			}
			if session == "" {
				return fmt.Errorf("必须指定 --session 参数")
			}
			creds, err := resolveCredentials()
			if err != nil {
				return err
			}
			detail, err := getROMDetail(creds.ServerURL, creds.Token, seat, session)
			if err != nil {
				return err
			}
			return printROMDetailResult(romDetailResult{Success: true, Server: creds.ServerURL, Seat: seat, Session: session, Detail: detail})
		},
	}

	cmd.Flags().IntVar(&seat, "seat", 0, "盘位号（必填）")
	cmd.Flags().StringVar(&session, "session", "", "刷机会话 ID（必填）")
	return cmd
}

func uploadROMPackage(serverURL, username, token, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开 ROM 文件失败: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("获取 ROM 文件信息失败: %v", err)
	}
	if stat.IsDir() {
		return fmt.Errorf("--file 不能是目录")
	}

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

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(serverURL, "/")+"/box/upload", pr)
	if err != nil {
		return fmt.Errorf("创建上传请求失败: %v", err)
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Cookie", fmt.Sprintf("username=%s;token=%s", username, token))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 2 * time.Hour}
	resp, err := client.Do(req)
	writeErr := <-errCh
	if err != nil {
		if writeErr != nil {
			return fmt.Errorf("上传 ROM 失败: %v", writeErr)
		}
		return fmt.Errorf("上传 ROM 失败: %v", err)
	}
	defer resp.Body.Close()
	if writeErr != nil {
		return fmt.Errorf("写入 ROM 数据失败: %v", writeErr)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取上传响应失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("上传 ROM 失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析上传响应失败: %s", string(body))
	}
	if result.Code != 200 {
		if result.Msg == "" {
			result.Msg = "上传 ROM 失败"
		}
		return fmt.Errorf(result.Msg)
	}
	return nil
}

func newGuardClient(serverURL, token string) (*wsclient.Client, error) {
	client := wsclient.NewClient(serverURL, token)
	client.Endpoint = "/box/guard"
	client.Params = map[string]string{"id": "0"}
	client.Timeout = 15 * time.Second
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("连接 Guard WebSocket 失败: %v", err)
	}
	return client, nil
}

func getROMPackages(client *wsclient.Client) ([]model.ROMPackage, error) {
	resp, err := client.SendRequest(113, nil)
	if err != nil {
		return nil, err
	}
	if resp.Code != nil && *resp.Code != 200 {
		return nil, fmt.Errorf(extractWSMessage(resp, "获取 ROM 包列表失败"))
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, err
	}
	var packages []model.ROMPackage
	if err := json.Unmarshal(data, &packages); err != nil {
		return nil, fmt.Errorf("解析 ROM 包列表失败: %v", err)
	}
	return packages, nil
}

func flashROM(client *wsclient.Client, seat int, sn, image string, mode int) (string, error) {
	resp, err := client.SendRequest(119, map[string]interface{}{
		"seat":  seat,
		"sn":    sn,
		"image": image,
		"mode":  mode,
	})
	if err != nil {
		return "", err
	}
	if resp.Code != nil && *resp.Code != 200 {
		return "", fmt.Errorf(extractWSMessage(resp, "发起 ROM 刷机失败"))
	}
	return extractWSMessage(resp, "flash request submitted"), nil
}

func queryFlashStatus(client *wsclient.Client) ([]romStatusItemView, error) {
	resp, err := client.SendRequest(117, nil)
	if err != nil {
		return nil, err
	}
	if resp.Code != nil && *resp.Code != 200 {
		return nil, fmt.Errorf(extractWSMessage(resp, "查询刷机状态失败"))
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, err
	}
	var rawItems []map[string]interface{}
	if err := json.Unmarshal(data, &rawItems); err != nil {
		return nil, fmt.Errorf("解析刷机状态失败: %v", err)
	}
	items := make([]romStatusItemView, 0, len(rawItems))
	for _, item := range rawItems {
		items = append(items, romStatusItemView{
			Seat:      intValue(item["seat"]),
			SN:        stringValue(item["sn"]),
			Mode:      intValue(item["mode"]),
			Status:    intValue(item["status"]),
			Session:   stringValue(item["session"]),
			Image:     stringValue(item["image"]),
			QueueTime: int64Value(item["queueTime"]),
			StartTime: int64Value(item["startTime"]),
			EndTime:   int64Value(item["endTime"]),
			LastError: stringValue(item["lastError"]),
		})
	}
	return items, nil
}

func getROMDetail(serverURL, token string, seat int, session string) (string, error) {
	u, err := url.Parse(strings.TrimRight(serverURL, "/") + "/box/detail")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("id", fmt.Sprintf("%d", seat))
	q.Set("session", session)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", token)

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("获取刷机详情失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取刷机详情失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取刷机详情失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	text := string(body)
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		text = parseSSEDetail(text)
	}
	return strings.TrimSpace(text), nil
}

func parseSSEDetail(text string) string {
	lines := strings.Split(text, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			content := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if content != "" {
				parts = append(parts, content)
			}
			continue
		}
		if line != "" && !strings.HasPrefix(line, ":") && !strings.HasPrefix(line, "event:") && !strings.HasPrefix(line, "id:") {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, "\n")
}

func extractWSMessage(resp *model.WSResponse, fallback string) string {
	if resp != nil && resp.Msg != nil && *resp.Msg != "" {
		return *resp.Msg
	}
	if resp != nil {
		if dataMap, ok := resp.Data.(map[string]interface{}); ok {
			if msg, ok := dataMap["message"].(string); ok && msg != "" {
				return msg
			}
		}
	}
	return fallback
}

func intValue(v interface{}) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case float32:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	case json.Number:
		n, _ := x.Int64()
		return int(n)
	default:
		return 0
	}
}

func int64Value(v interface{}) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case float32:
		return int64(x)
	case int:
		return int64(x)
	case int64:
		return x
	case json.Number:
		n, _ := x.Int64()
		return n
	case map[string]interface{}:
		if value, ok := x["Value"]; ok {
			return int64Value(value)
		}
		if value, ok := x["value"]; ok {
			return int64Value(value)
		}
		if number, ok := x["number"]; ok {
			return int64Value(number)
		}
		return 0
	default:
		return 0
	}
}

func stringValue(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", x)
	}
}

func printROMUploadResult(result romUploadResult) error {
	if flagOutput == "json" {
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("SERVER\t%s\n", strings.TrimPrefix(strings.TrimPrefix(result.Server, "https://"), "http://"))
	fmt.Printf("FILE\t%s\n", result.File)
	fmt.Printf("MESSAGE\t%s\n", result.Message)
	fmt.Println("STATUS\tsuccess")
	return nil
}

func printROMListResult(result romListResult) error {
	if flagOutput == "json" {
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
		return nil
	}
	fmt.Println("NAME\tMODEL\tVERSION\tDESC")
	for _, pkg := range result.Packages {
		fmt.Printf("%s\t%s\t%s\t%s\n", pkg.Name, pkg.Model, pkg.Version, pkg.Desc)
	}
	fmt.Printf("--- total: %d\n", result.Total)
	return nil
}

func printROMFlashResult(result romFlashResult) error {
	if flagOutput == "json" {
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("SERVER\t%s\n", strings.TrimPrefix(strings.TrimPrefix(result.Server, "https://"), "http://"))
	fmt.Printf("SEAT\t%d\n", result.Seat)
	fmt.Printf("SN\t%s\n", result.SN)
	fmt.Printf("IMAGE\t%s\n", result.Image)
	fmt.Printf("MODE\t%d\n", result.Mode)
	fmt.Printf("MESSAGE\t%s\n", result.Message)
	fmt.Println("STATUS\tsuccess")
	return nil
}

func printROMStatusResult(result romStatusResult) error {
	if flagOutput == "json" {
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
		return nil
	}
	fmt.Println("SEAT\tSN\tMODE\tSTATUS\tSESSION\tIMAGE\tQUEUE\tSTART\tEND\tERROR")
	for _, item := range result.Items {
		fmt.Printf("%d\t%s\t%d\t%d\t%s\t%s\t%d\t%d\t%d\t%s\n", item.Seat, item.SN, item.Mode, item.Status, item.Session, item.Image, item.QueueTime, item.StartTime, item.EndTime, item.LastError)
	}
	fmt.Printf("--- total: %d\n", result.Total)
	return nil
}

func printROMDetailResult(result romDetailResult) error {
	if flagOutput == "json" {
		data, _ := json.Marshal(result)
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("SERVER\t%s\n", strings.TrimPrefix(strings.TrimPrefix(result.Server, "https://"), "http://"))
	fmt.Printf("SEAT\t%d\n", result.Seat)
	fmt.Printf("SESSION\t%s\n", result.Session)
	fmt.Println("DETAIL")
	fmt.Println(result.Detail)
	return nil
}
