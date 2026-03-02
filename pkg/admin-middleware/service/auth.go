package service

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"jpy-cli/pkg/admin-middleware/api"
	"jpy-cli/pkg/config"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"golang.org/x/term"
)

type AdminRole string

const (
	RoleAuth      AdminRole = "auth"
	RoleOperation AdminRole = "operation"
)

func EnsureLoggedIn() (*config.AdminConfig, error) {
	return EnsureAuthLoggedIn()
}

func EnsureAuthLoggedIn() (*config.AdminConfig, error) {
	return ensureRoleLoggedIn(RoleAuth)
}

func EnsureOperationLoggedIn() (*config.AdminConfig, error) {
	return ensureRoleLoggedIn(RoleOperation)
}

func ensureRoleLoggedIn(role AdminRole) (*config.AdminConfig, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	var adminCfg *config.AdminConfig
	if role == RoleAuth {
		adminCfg = cfg.AdminAuth
		// Fallback to legacy Admin if AdminAuth is missing
		if adminCfg == nil && cfg.Admin != nil {
			adminCfg = cfg.Admin
		}
	} else if role == RoleOperation {
		adminCfg = cfg.AdminOperation
	}

	// Check if token exists
	if adminCfg != nil && adminCfg.Token != "" {
		// Validate token with a lightweight request (e.g. Search with dummy)
		client := api.NewClient(adminCfg.Token)
		_, err := client.SearchAuthCode("CHECK_TOKEN_VALIDITY")
		if err == nil || strings.Contains(err.Error(), "not found") {
			return adminCfg, nil
		}
		if err.Error() != "unauthorized" && err.Error() != "权限不足，请重新登录" {
			// Network error or other issue, but let's assume valid if not explicitly unauthorized
			// actually, if "not found" it means token worked.
			// If "unauthorized", we need to relogin.
			fmt.Printf("Token expired or invalid: %v\n", err)
		}
	}

	return PerformLogin(cfg, role)
}

func PerformLogin(cfg *config.Config, role AdminRole) (*config.AdminConfig, error) {
	roleName := "Admin"
	if role == RoleAuth {
		roleName = "Authorization"
	} else if role == RoleOperation {
		roleName = "Operation"
	}
	fmt.Printf("=== %s 登录 ===\n", roleName)
	client := api.NewClient("")

	// 1. Get Captcha
	captcha, err := client.GetCaptcha()
	if err != nil {
		return nil, fmt.Errorf("获取验证码失败: %v", err)
	}

	// 2. Display Captcha
	captchaPath := filepath.Join(os.TempDir(), "jpy_captcha.png")
	if err := saveBase64Image(captcha.CaptchaPic, captchaPath); err != nil {
		return nil, fmt.Errorf("保存验证码失败: %v", err)
	}

	fmt.Printf("验证码已保存到: %s\n", captchaPath)
	if err := openFile(captchaPath); err != nil {
		fmt.Printf("请手动打开图片查看验证码。\n")
	}

	// 3. Prompt for Input
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("输入验证码: ")
	captchaCode, _ := reader.ReadString('\n')
	captchaCode = strings.TrimSpace(captchaCode)

	fmt.Print("用户名: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("密码: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, err
	}
	password := string(bytePassword)
	fmt.Println() // Newline after password input

	// 4. Login
	resp, err := client.Login(username, password, captcha.CaptchaID, captchaCode)
	if err != nil {
		return nil, err
	}

	if resp.Status != 200 {
		return nil, fmt.Errorf("%s", resp.Msg)
	}

	// 5. Save Token
	if role == RoleAuth {
		if cfg.AdminAuth == nil {
			cfg.AdminAuth = &config.AdminConfig{}
		}
		cfg.AdminAuth.Token = resp.Data.Token
		cfg.AdminAuth.Username = username
		// Also update legacy Admin for compatibility if needed, or maybe just migrate
		// For now, let's keep AdminAuth as primary.
	} else if role == RoleOperation {
		if cfg.AdminOperation == nil {
			cfg.AdminOperation = &config.AdminConfig{}
		}
		cfg.AdminOperation.Token = resp.Data.Token
		cfg.AdminOperation.Username = username
	}

	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("保存配置失败: %v", err)
	}

	fmt.Println("登录成功!")
	if role == RoleAuth {
		return cfg.AdminAuth, nil
	}
	return cfg.AdminOperation, nil
}

func saveBase64Image(base64Str, outputPath string) error {
	idx := strings.Index(base64Str, ";base64,")
	if idx != -1 {
		base64Str = base64Str[idx+8:]
	}
	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}

func openFile(path string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, path)
	return exec.Command(cmd, args...).Start()
}
