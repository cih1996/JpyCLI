package ws

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/admin-middleware/api"
	apiModel "jpy-cli/pkg/admin-middleware/model"
	"jpy-cli/pkg/admin-middleware/service"
	httpclient "jpy-cli/pkg/client/http"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/model"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

func handleServerList(conn *SafeConn, req Request) {
	cfg, err := config.Load()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to load server config: %v", err))
		return
	}

	// Limit servers in each group to 100 to avoid performance issues
	// limitedGroups := make(map[string][]config.LocalServerConfig)
	// for groupName, servers := range cfg.Groups {
	// 	if len(servers) > 100 {
	// 		limitedGroups[groupName] = servers[:100]
	// 	} else {
	// 		limitedGroups[groupName] = servers
	// 	}
	// }
	// cfg.Groups = limitedGroups

	sendSuccess(conn, req.ID, cfg)
}

func handleServerSave(conn *SafeConn, req Request) {
	var server config.LocalServerConfig
	if err := json.Unmarshal(req.Params, &server); err != nil {
		sendError(conn, req.ID, "Invalid server config params")
		return
	}

	if server.URL == "" {
		sendError(conn, req.ID, "Server URL is required")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	if err := config.UpdateServer(cfg, server); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to save server: %v", err))
		return
	}

	sendSuccess(conn, req.ID, "Server saved successfully")
}

type ServerRemoveParams struct {
	URL   string `json:"url"`
	Group string `json:"group"`
	Soft  bool   `json:"soft"`
}

func handleServerRemove(conn *SafeConn, req Request) {
	var params ServerRemoveParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(conn, req.ID, "Invalid params")
		return
	}

	if params.URL == "" {
		sendError(conn, req.ID, "URL is required")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	if params.Soft {
		if err := config.SetServerDisabled(cfg, params.URL, params.Group, true); err != nil {
			sendError(conn, req.ID, fmt.Sprintf("Failed to soft remove server: %v", err))
			return
		}
	} else {
		if err := config.RemoveServer(cfg, params.URL, params.Group); err != nil {
			sendError(conn, req.ID, fmt.Sprintf("Failed to remove server: %v", err))
			return
		}
	}

	sendSuccess(conn, req.ID, "Server removed")
}

type ServerRemoveBatchParams struct {
	Items []ServerRemoveParams `json:"items"`
}

func handleServerRemoveBatch(conn *SafeConn, req Request) {
	var params ServerRemoveBatchParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(conn, req.ID, "Invalid params")
		return
	}

	if len(params.Items) == 0 {
		sendSuccess(conn, req.ID, "No items to remove")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	// Group by operation type
	var softRemoveItems []config.BatchItem
	var hardRemoveItems []ServerRemoveParams

	for _, item := range params.Items {
		if item.Soft {
			softRemoveItems = append(softRemoveItems, config.BatchItem{
				URL:   item.URL,
				Group: item.Group,
			})
		} else {
			hardRemoveItems = append(hardRemoveItems, item)
		}
	}

	// Handle Soft Removes
	if len(softRemoveItems) > 0 {
		if err := config.SetServerDisabledBatch(cfg, softRemoveItems, true); err != nil {
			sendError(conn, req.ID, fmt.Sprintf("Failed to soft remove batch: %v", err))
			return
		}
	}

	// Handle Hard Removes (one by one for now as store.go doesn't have batch remove yet,
	// but usually soft remove is the main use case for batch)
	for _, item := range hardRemoveItems {
		if err := config.RemoveServer(cfg, item.URL, item.Group); err != nil {
			// Log error but continue? Or fail?
			// For now, let's stop on error
			sendError(conn, req.ID, fmt.Sprintf("Failed to remove server %s: %v", item.URL, err))
			return
		}
	}

	sendSuccess(conn, req.ID, fmt.Sprintf("Batch remove complete: %d items", len(params.Items)))
}

func handleServerRelogin(conn *SafeConn, req Request) {
	var params struct {
		Group string `json:"group"`
		URL   string `json:"url"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(conn, req.ID, "Invalid params")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	targetGroup := params.Group
	if targetGroup == "" {
		targetGroup = cfg.ActiveGroup
	}
	if targetGroup == "" {
		targetGroup = "default"
	}

	servers := config.GetGroupServers(cfg, targetGroup)
	var targetIndices []int

	if params.URL != "" {
		// Find specific server
		for i, s := range servers {
			if s.URL == params.URL {
				targetIndices = append(targetIndices, i)
				break
			}
		}
		if len(targetIndices) == 0 {
			sendError(conn, req.ID, fmt.Sprintf("Server not found: %s", params.URL))
			return
		}
	} else {
		// Find all disabled servers
		for i, s := range servers {
			if s.Disabled {
				targetIndices = append(targetIndices, i)
			}
		}
		if len(targetIndices) == 0 {
			sendSuccess(conn, req.ID, "No disabled servers found to relogin")
			return
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 20)
	successCount := 0
	failCount := 0

	// Use original slice to modify
	groupServers := cfg.Groups[targetGroup]

	for _, idx := range targetIndices {
		idx := idx // capture loop variable
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			s := groupServers[idx]

			// 1. Try Login
			client := httpclient.NewClient(s.URL, s.Token)
			client.SetTimeout(3 * time.Second) // Fast timeout for relogin check

			// Try GetLicense first to check if token is valid
			_, err := client.GetLicense()
			if err != nil {
				// Try login
				newToken, loginErr := client.Login(s.Username, s.Password)
				if loginErr != nil {
					mu.Lock()
					failCount++
					groupServers[idx].LastLoginError = loginErr.Error()
					mu.Unlock()
					return
				}
				s.Token = newToken
				client.Token = newToken
			}

			// 2. Enable Server
			mu.Lock()
			groupServers[idx].Disabled = false
			groupServers[idx].LastLoginError = ""
			groupServers[idx].Token = s.Token
			successCount++
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Update config with modified servers
	cfg.Groups[targetGroup] = groupServers
	if err := config.Save(cfg); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Relogin completed but save failed: %v", err))
		return
	}

	sendSuccess(conn, req.ID, fmt.Sprintf("Relogin complete. Success: %d, Fail: %d", successCount, failCount))
}

func handleServerAutoAuth(conn *SafeConn, req Request) {
	var params struct {
		Prefix string `json:"prefix"`
	}
	json.Unmarshal(req.Params, &params)

	cfg, err := config.Load()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	sendProgress(conn, req.ID, 0, 100, "Scanning server authorization status...")
	unauthorized := scanServersForWS(cfg, func(current, total int, msg string) {
		sendProgress(conn, req.ID, current, total, msg)
	})

	if len(unauthorized) == 0 {
		sendSuccess(conn, req.ID, "All servers are authorized.")
		return
	}

	adminCfg, err := service.EnsureLoggedIn()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Admin login required: %v", err))
		return
	}
	adminClient := api.NewClient(adminCfg.Token)

	sendProgress(conn, req.ID, 10, 100, "Fetching recent auth records...")
	recentRecords, _ := adminClient.GetRecentAuthRecords(100)

	prefix := params.Prefix
	if prefix == "" {
		prefix = "CS-JPY-"
	}

	processAuthorizationForWS(conn, req.ID, unauthorized, prefix, adminClient, recentRecords)
}

func handleServerUpdateCluster(conn *SafeConn, req Request) {
	var params struct {
		TargetAddr    string `json:"target_addr"`
		Group         string `json:"group"`
		ServerPattern string `json:"server_pattern"`
		Authorized    string `json:"authorized"`
		Force         bool   `json:"force"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(conn, req.ID, "Invalid params")
		return
	}

	if params.TargetAddr == "" {
		sendError(conn, req.ID, "Target address is required")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	targetGroup := params.Group
	if targetGroup == "" {
		targetGroup = cfg.ActiveGroup
	}
	if targetGroup == "" {
		targetGroup = "default"
	}

	sendProgress(conn, req.ID, 0, 100, "Scanning servers...")

	servers := config.GetGroupServers(cfg, targetGroup)
	type candidateServer struct {
		Server  config.LocalServerConfig
		License *model.LicenseData
	}
	var candidates []candidateServer
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	totalServers := len(servers)
	processedCount := 0

	for _, s := range servers {
		if s.Disabled {
			processedCount++
			continue
		}

		// Filter by pattern
		if params.ServerPattern != "" {
			if !strings.Contains(s.URL, params.ServerPattern) {
				processedCount++
				continue
			}
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(s config.LocalServerConfig) {
			defer wg.Done()
			defer func() { <-sem }()

			// Use shorter timeout for check
			apiClient := httpclient.NewClient(s.URL, s.Token)
			apiClient.SetTimeout(2 * time.Second)
			lic, err := apiClient.GetLicense()
			if err == nil {
				// Filter by Authorized status
				match := true
				if params.Authorized == "yes" && !lic.S {
					match = false
				} else if params.Authorized == "no" && lic.S {
					match = false
				}

				if match {
					mu.Lock()
					candidates = append(candidates, candidateServer{Server: s, License: lic})
					mu.Unlock()
				}
			}

			mu.Lock()
			processedCount++
			// Only send progress occasionally to avoid spamming
			if processedCount%5 == 0 || processedCount == totalServers {
				sendProgress(conn, req.ID, processedCount, totalServers, fmt.Sprintf("Scanning %d/%d...", processedCount, totalServers))
			}
			mu.Unlock()
		}(s)
	}
	wg.Wait()

	if len(candidates) == 0 {
		sendSuccess(conn, req.ID, "No matching servers found.")
		return
	}

	sendProgress(conn, req.ID, 0, len(candidates), fmt.Sprintf("Updating %d servers...", len(candidates)))

	adminCfg, err := service.EnsureLoggedIn()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Admin login required: %v", err))
		return
	}
	adminClient := api.NewClient(adminCfg.Token)

	// Pre-filter auth codes to find target serial number record
	// var targetRecord *apiModel.AuthCodeItem
	if params.TargetAddr != "" {
		// Need to find which auth code corresponds to this target address if possible
		// Or just list all codes.
		// Actually updateServerCluster logic seems to use existing code.
		// If we want to verify target address validity or get more info.
	}

	successCount := 0
	failCount := 0
	processedCount = 0

	for _, cand := range candidates {
		s := cand.Server
		lic := cand.License

		wg.Add(1)
		sem <- struct{}{}
		go func(s config.LocalServerConfig, lic *model.LicenseData) {
			defer wg.Done()
			defer func() { <-sem }()

			err := updateServerCluster(s, lic.Sn, params.TargetAddr, adminClient, params.Force)
			mu.Lock()
			if err != nil {
				failCount++
				logger.Errorf("Failed to update cluster for %s: %v", s.URL, err)
			} else {
				successCount++
			}
			processedCount++
			sendProgress(conn, req.ID, processedCount, len(candidates), fmt.Sprintf("Updating %d/%d (Success: %d, Fail: %d)", processedCount, len(candidates), successCount, failCount))
			mu.Unlock()
		}(s, lic)
	}
	wg.Wait()

	sendSuccess(conn, req.ID, fmt.Sprintf("Cluster update complete. Success: %d, Fail: %d", successCount, failCount))
}

func scanServersForWS(cfg *config.Config, progressCb func(int, int, string)) []config.LocalServerConfig {
	allServers := config.GetAllServers(cfg)
	var activeServers []config.LocalServerConfig
	for _, s := range allServers {
		if !s.Disabled {
			activeServers = append(activeServers, s)
		}
	}
	total := len(activeServers)
	var unauthorized []config.LocalServerConfig
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // concurrency limit

	processed := 0

	for _, s := range activeServers {
		wg.Add(1)
		sem <- struct{}{}
		go func(server config.LocalServerConfig) {
			defer wg.Done()
			defer func() { <-sem }()

			client := httpclient.NewClient(server.URL, server.Token)
			client.SetTimeout(2 * time.Second) // Fast timeout
			lic, licErr := client.GetLicense()
			if licErr != nil {
				// Try login if license fetch fails (token might be expired)
				newToken, loginErr := client.Login(server.Username, server.Password)
				if loginErr == nil {
					server.Token = newToken
					config.UpdateServer(cfg, server)
					client.Token = newToken
					lic, licErr = client.GetLicense()
				}
			}

			if licErr == nil && !lic.S {
				mu.Lock()
				unauthorized = append(unauthorized, server)
				mu.Unlock()
			}

			mu.Lock()
			processed++
			if processed%5 == 0 || processed == total {
				progressCb(processed, total, fmt.Sprintf("Scanning %d/%d...", processed, total))
			}
			mu.Unlock()
		}(s)
	}
	wg.Wait()
	return unauthorized
}

func processAuthorizationForWS(conn *SafeConn, reqID string, servers []config.LocalServerConfig, prefix string, adminClient *api.Client, recentRecords []apiModel.AuthCodeItem) {
	successCount := 0
	failCount := 0
	total := len(servers)

	for i, s := range servers {
		sendProgress(conn, reqID, i+1, total, fmt.Sprintf("Authorizing %s...", s.URL))

		// Logic to reuse or create auth code
		// Simplified for brevity, assume we use existing logic from service/auth.go or similar
		// But since we can't import main packages easily due to cycles, we reimplement or move logic.
		// Here we just try to find by name match or create new.

		// 1. Check if name exists in recent records
		// ... (Implementation depends on complexity required)

		// For now, let's just count them as processed
		// In real impl, we would call adminClient.CreateAuthCode or UpdateAuth

		// Placeholder for actual logic
		name := fmt.Sprintf("%s%s", prefix, generateSuffix(s.URL))

		// Check if exists
		var key string
		// Try to find in recent
		for _, r := range recentRecords {
			if r.Name == name {
				key = r.SerialNumber
				break
			}
		}

		if key == "" {
			// Search
			var err error
			key, err = adminClient.SearchAuthCode(name)
			if err != nil {
				failCount++
				continue
			}
		}

		apiBase := s.URL
		if !strings.HasPrefix(apiBase, "http") {
			apiBase = "http://" + apiBase
		}

		cfg, _ := config.Load()
		token := findToken(cfg, s.URL)

		serverClient := httpclient.NewClient(apiBase, token)
		err := serverClient.Reauthorize(key)
		if err != nil {
			failCount++
		} else {
			successCount++
		}
	}

	sendSuccess(conn, reqID, fmt.Sprintf("Auth complete. Success: %d, Fail: %d", successCount, failCount))
}

func updateServerCluster(server config.LocalServerConfig, sn string, targetAddr string, adminClient *api.Client, force bool) error {
	authItem, err := adminClient.GetAuthBySN(sn)
	if err != nil {
		return fmt.Errorf("get auth failed: %v", err)
	}

	if authItem.MgtCenter == targetAddr && !force {
		// Already matches
	} else {
		payload := apiModel.AuthCodePayload{
			ID:           authItem.ID,
			Supervise:    authItem.Supervise,
			Type:         authItem.Type,
			Name:         authItem.Name,
			SerialNumber: authItem.SerialNumber,
			Title:        authItem.Title,
			MgtCenter:    targetAddr,
			Limit:        authItem.Limit,
			Day:          authItem.Day,
			Desc:         authItem.Desc,
		}

		if err := adminClient.UpdateAuth(payload); err != nil {
			return fmt.Errorf("update auth failed: %v", err)
		}
	}

	middlewareClient := httpclient.NewClient(server.URL, server.Token)
	if err := middlewareClient.Reauthorize(sn); err != nil {
		return fmt.Errorf("middleware reauth failed: %v", err)
	}

	return nil
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
	u, err := url.Parse(serverURL)
	if err != nil {
		u, err = url.Parse("http://" + serverURL)
	}

	if err == nil {
		host := u.Hostname()
		port := u.Port()

		if port != "" && port != "80" && port != "443" {
			return port
		}

		ip := net.ParseIP(host)
		if ip != nil {
			v4 := ip.To4()
			if v4 != nil {
				return fmt.Sprintf("%d%d", v4[2], v4[3])
			}
		}
	}
	return "00000"
}
