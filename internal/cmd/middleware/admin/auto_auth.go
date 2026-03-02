package admin

import (
	"bufio"
	"context"
	"fmt"
	"jpy-cli/pkg/admin-middleware/api"
	apiModel "jpy-cli/pkg/admin-middleware/model"
	"jpy-cli/pkg/admin-middleware/service"
	httpclient "jpy-cli/pkg/client/http"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/device/fetcher"
	"jpy-cli/pkg/middleware/model"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

func NewAutoAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-auth",
		Short: "自动扫描并重新授权中间件服务器",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Load Config
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// 2. Scan for Unauthorized Servers
			fmt.Println("正在扫描服务器授权状态...")
			unauthorized := scanServers(cfg)

			if len(unauthorized) == 0 {
				fmt.Println("所有服务器均已授权。")
				return nil
			}

			// 3. Display Stats
			fmt.Printf("\n发现 %d 台未授权服务器。\n", len(unauthorized))
			fmt.Println("未授权服务器 Top 10 (按设备数):")
			for i, s := range unauthorized {
				if i >= 10 {
					break
				}
				fmt.Printf(" - %s (设备数: %d, 在线: %v)\n", s.Address, s.DeviceCount, s.Online)
			}

			// 4. Admin Login
			adminCfg, err := service.EnsureLoggedIn()
			if err != nil {
				return err
			}
			adminClient := api.NewClient(adminCfg.Token)

			// 5. Interactive Prompt
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("\n请输入前缀 (默认 'CS-JPY-'): ")
			prefix, _ := reader.ReadString('\n')
			prefix = strings.TrimSpace(prefix)
			if prefix == "" {
				prefix = "CS-JPY-"
			}

			fmt.Printf("\n将尝试授权 %d 台服务器，前缀为 '%s'。是否继续? [y/N]: ", len(unauthorized), prefix)
			confirm, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
				fmt.Println("操作已取消。")
				return nil
			}

			// 6. Pre-fetch Recent Auth Records (Cache)
			fmt.Println("正在获取最近的授权记录缓存 (Top 100)...")
			recentRecords, err := adminClient.GetRecentAuthRecords(100)
			if err != nil {
				// Warn but continue, maybe SearchAuthCode fallback will work
				logger.Warnf("Failed to fetch recent records: %v", err)
				fmt.Printf("警告: 获取最近授权记录失败: %v (将使用逐个查询)\n", err)
			} else {
				fmt.Printf("成功缓存 %d 条最近授权记录。\n", len(recentRecords))
			}

			// 7. Process Authorization
			processAuthorization(unauthorized, prefix, adminClient, recentRecords)

			return nil
		},
	}
	return cmd
}

func scanServers(cfg *config.Config) []model.ServerStatus {
	var unauthorizedConfigs []config.LocalServerConfig
	var mu sync.Mutex
	var wg sync.WaitGroup

	allServers := config.GetAllServers(cfg)
	// client := serverApi.NewServerClient()
	sem := make(chan struct{}, 20) // Concurrency limit

	// Step 1: Check Licenses
	fmt.Printf("正在检查 %d 台服务器的授权状态...\n", len(allServers))
	for _, s := range allServers {
		wg.Add(1)
		go func(server config.LocalServerConfig) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			apiBase := server.URL
			if !strings.HasPrefix(apiBase, "http") {
				apiBase = "http://" + apiBase
			}

			client := httpclient.NewClient(apiBase, server.Token)
			info, err := client.GetLicense()
			if err != nil {
				// If we can't connect, skip
				return
			}

			// Debug logging for license status
			statusVal := fmt.Sprintf("%d", info.Status)

			// Check authorization status using logic from status.go (StatusTxt == "成功")
			isAuthorized := false
			statusTxt := info.StatusTxt

			if statusTxt == "成功" {
				isAuthorized = true
			}

			if !isAuthorized {
				logger.Infof("Server %s license status: Valid=%v, Status=%s, StatusTxt=%s (Unauthorized)", server.URL, info.S, statusVal, statusTxt)
				mu.Lock()
				unauthorizedConfigs = append(unauthorizedConfigs, server)
				mu.Unlock()
			}
		}(s)
	}
	wg.Wait()

	if len(unauthorizedConfigs) == 0 {
		return nil
	}

	fmt.Printf("正在获取 %d 台未授权服务器的设备统计信息...\n", len(unauthorizedConfigs))

	// Step 2: Fetch Device Stats for unauthorized servers
	resultsChan, _ := fetcher.FetchDevices(context.Background(), unauthorizedConfigs, cfg)

	var unauthorizedStats []model.ServerStatus
	for resRaw := range resultsChan {
		res := resRaw.(fetcher.ServerResult)

		status := model.ServerStatus{
			Address:     res.ServerURL,
			Online:      res.Error == nil,
			DeviceCount: len(res.Devices),
		}
		if res.Error != nil {
			status.Error = res.Error.Error()
		}
		unauthorizedStats = append(unauthorizedStats, status)
	}

	// Sort by DeviceCount descending, then Address ascending
	sort.Slice(unauthorizedStats, func(i, j int) bool {
		if unauthorizedStats[i].DeviceCount != unauthorizedStats[j].DeviceCount {
			return unauthorizedStats[i].DeviceCount > unauthorizedStats[j].DeviceCount
		}
		return unauthorizedStats[i].Address < unauthorizedStats[j].Address
	})

	return unauthorizedStats
}

func processAuthorization(servers []model.ServerStatus, prefix string, adminClient *api.Client, recentRecords []apiModel.AuthCodeItem) {
	// serverClient := serverApi.NewServerClient()
	successCount := 0
	failCount := 0

	for i, s := range servers {
		// Generate Name
		suffix := generateSuffix(s.Address)
		name := prefix + suffix

		fmt.Printf("[%d/%d] 正在处理 %s (名称: %s)... ", i+1, len(servers), s.Address, name)
		logger.Infof("[AUDIT] START: Authorization for server=%s, name=%s", s.Address, name)

		// 1. Check Cache First
		var key string
		var foundInCache bool
		for _, record := range recentRecords {
			if record.Name == name {
				key = record.SerialNumber
				foundInCache = true
				fmt.Print("(缓存命中) ")
				logger.Infof("[AUDIT] Found existing auth in cache: name=%s, key=%s", name, key)
				break
			}
		}

		if !foundInCache {
			// 2. Generate Auth Code if not found
			err := adminClient.GenerateAuthCode(name)
			if err != nil {
				// If error (e.g. duplicate name not in cache), we still try to search
				logger.Warnf("[AUDIT] GenerateAuthCode warning for name=%s: %v (will try search)", name, err)
			} else {
				logger.Infof("[AUDIT] GenerateAuthCode success: name=%s", name)
			}

			// 3. Get Serial Number (Key) from API
			key, err = adminClient.SearchAuthCode(name)
			if err != nil {
				fmt.Printf("失败 (查找Key: %v)\n", err)
				logger.Errorf("[AUDIT] FAILED: SearchAuthCode failed for name=%s: %v", name, err)
				failCount++
				continue
			}
			logger.Infof("[AUDIT] SearchAuthCode success: name=%s, key=%s", name, key)
		} else {
			// Found in cache, skip generation and search
		}

		// 4. Reauthorize Server
		// Ensure URL format
		apiBase := s.Address
		if !strings.HasPrefix(apiBase, "http") {
			apiBase = "http://" + apiBase
		}

		cfg, _ := config.Load() // Reload to be safe
		token := findToken(cfg, s.Address)

		serverClient := httpclient.NewClient(apiBase, token)
		logger.Infof("[AUDIT] Reauthorizing server=%s with key=%s", s.Address, key)
		err := serverClient.Reauthorize(key)
		if err != nil {
			fmt.Printf("失败 (重授权: %v)\n", err)
			logger.Errorf("[AUDIT] FAILED: Reauthorize failed for server=%s: %v", s.Address, err)
			failCount++
		} else {
			fmt.Printf("成功\n")
			logger.Infof("[AUDIT] SUCCESS: Server authorized. server=%s, name=%s, key=%s", s.Address, name, key)
			successCount++
		}
	}

	fmt.Printf("\n完成。成功: %d, 失败: %d\n", successCount, failCount)
}

func findToken(cfg *config.Config, url string) string {
	servers := config.GetAllServers(cfg)
	for _, s := range servers {
		if s.URL == url {
			return s.Token
		}
	}
	return ""
}

func generateSuffix(serverURL string) string {
	// Logic:
	// 1. Parse URL
	// 2. If Port exists and is "special" (e.g. > 10000 or user logic), use Port.
	// 3. Else use IP last 2 octets.

	u, err := url.Parse(serverURL)
	if err != nil {
		// Try adding scheme if missing
		u, err = url.Parse("http://" + serverURL)
	}

	if err == nil {
		host := u.Hostname()
		port := u.Port()

		// Heuristic: If port is present and not standard (80/443), prefer it?
		// User example: 129.204.22.176:31203 -> 31203.
		// User example: 192.168.31.203 -> 31203.
		// It seems 31203 is preferred.
		if port != "" && port != "80" && port != "443" {
			return port
		}

		// Fallback to IP octets
		// 192.168.31.203 -> 31203
		ip := net.ParseIP(host)
		if ip != nil {
			v4 := ip.To4()
			if v4 != nil {
				return fmt.Sprintf("%d%d", v4[2], v4[3])
			}
		}
	}

	// Fallback if parsing fails or not IP
	return "00000"
}
