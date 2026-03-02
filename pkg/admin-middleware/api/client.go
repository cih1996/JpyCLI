package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/admin-middleware/model"
	"net/http"
	"time"
)

const AdminAPIBase = "https://admin.htsystem.cn/api/v1"

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		BaseURL: AdminAPIBase,
		Token:   token,
		HTTP: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) GetCaptcha() (*model.AdminCaptchaData, error) {
	resp, err := c.HTTP.Get(c.BaseURL + "/admin/captcha")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result struct {
		Status int                    `json:"status"`
		Msg    string                 `json:"msg"`
		Data   model.AdminCaptchaData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status == 200 {
		return &result.Data, nil
	}
	return nil, fmt.Errorf("%s", result.Msg)
}

func (c *Client) Login(username, password, captchaID, captchaKey string) (*model.AdminLoginResponse, error) {
	payload := model.AdminLoginPayload{
		Username:   username,
		Password:   password,
		CaptchaID:  captchaID,
		CaptchaKey: captchaKey,
	}

	body, _ := json.Marshal(payload)
	resp, err := c.HTTP.Post(c.BaseURL+"/admin/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result model.AdminLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GenerateAuthCode(name string) error {
	payload := model.AuthCodePayload{
		ID:        0,
		Supervise: true,
		Type:      1,
		Name:      name,
		Title:     name,
		Limit:     20,
		Day:       365,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", c.BaseURL+"/partner/auth", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result model.AuthCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.Status != 200 {
		return fmt.Errorf("%s", result.Msg)
	}

	return nil
}

func (c *Client) DecryptPassword(encryptedCode string) (string, error) {
	payload := model.DecryptPasswordPayload{
		Code: encryptedCode,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", c.BaseURL+"/partner/password", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 402 || resp.StatusCode == 403 {
		return "", fmt.Errorf("权限不足，请重新登录")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result model.DecryptPasswordResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Status == 200 {
		return result.Data, nil
	}

	if result.Status == 401 || result.Status == 402 || result.Status == 403 {
		return "", fmt.Errorf("%s", result.Msg)
	}

	return "", fmt.Errorf("%s", result.Msg)
}

func (c *Client) GetAuthList(pageNum int) (*model.AuthSearchResult, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/partner/auth?did=0&sortOrder=id%%20desc&pageNum=%d", c.BaseURL, pageNum), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("unauthorized")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result model.AuthSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status == 200 {
		return &result, nil
	}

	if result.Status == 401 || result.Status == 403 {
		return nil, fmt.Errorf("unauthorized")
	}

	return nil, fmt.Errorf("%s", result.Msg)
}

func (c *Client) SearchAuthCode(name string) (string, error) {
	// 尝试在前 5 页中查找，每页 20 条，共 100 条
	for page := 1; page <= 5; page++ {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/partner/auth?did=0&sortOrder=id%%20desc&pageNum=%d", c.BaseURL, page), nil)
		if err != nil {
			return "", err
		}

		req.Header.Set("Authorization", c.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return "", fmt.Errorf("unauthorized")
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("HTTP %d", resp.StatusCode)
		}

		var result model.AuthSearchResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}

		if result.Status != 200 {
			if result.Status == 401 || result.Status == 403 {
				return "", fmt.Errorf("unauthorized")
			}
			return "", fmt.Errorf("%s", result.Msg)
		}

		for _, item := range result.Data.DataList {
			if item.Name == name {
				return item.SerialNumber, nil
			}
		}

		// 如果当前页没有填满，说明后面没有数据了，不用继续翻页
		if len(result.Data.DataList) < 20 {
			break
		}
	}

	return "", fmt.Errorf("not found in first 100 records")
}

func (c *Client) GetRecentAuthRecords(limit int) ([]model.AuthCodeItem, error) {
	var allRecords []model.AuthCodeItem
	pageSize := 20
	maxPages := (limit + pageSize - 1) / pageSize

	for page := 1; page <= maxPages; page++ {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/partner/auth?did=0&sortOrder=id%%20desc&pageNum=%d", c.BaseURL, page), nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", c.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}

		var result model.AuthSearchResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		if result.Status != 200 {
			return nil, fmt.Errorf("%s", result.Msg)
		}

		allRecords = append(allRecords, result.Data.DataList...)

		if len(result.Data.DataList) < pageSize {
			break
		}
	}

	if len(allRecords) > limit {
		allRecords = allRecords[:limit]
	}

	return allRecords, nil
}

func (c *Client) GetAuthBySN(sn string) (*model.AuthCodeItem, error) {
	url := fmt.Sprintf("%s/partner/auth?did=0&queryField=serial_number&query=%s&sortOrder=id%%20asc&pageNum=1&pageSize=30", c.BaseURL, sn)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result model.AuthSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != 200 {
		return nil, fmt.Errorf("%s", result.Msg)
	}

	for _, item := range result.Data.DataList {
		if item.SerialNumber == sn {
			return &item, nil
		}
	}

	return nil, fmt.Errorf("not found")
}

func (c *Client) UpdateAuth(payload model.AuthCodePayload) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("PUT", c.BaseURL+"/partner/auth", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.Status != 200 {
		return fmt.Errorf("%s", result.Msg)
	}

	return nil
}
