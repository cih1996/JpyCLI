package httpclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"jpy-cli/pkg/middleware/model"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewClient(baseURL, token string) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTP:    &http.Client{Timeout: 10 * time.Second, Transport: tr},
	}
}

func (c *Client) SetTimeout(d time.Duration) {
	if c.HTTP != nil {
		c.HTTP.Timeout = d
	}
}

type LoginResponse struct {
	Code int `json:"code"`
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
	Msg string `json:"msg"`
}

func (c *Client) Login(username, password string) (string, error) {
	payload := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	resp, err := c.HTTP.Post(c.BaseURL+"/login/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	var result LoginResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	if result.Code != 200 {
		return "", fmt.Errorf("登录失败: %s", result.Msg)
	}

	c.Token = result.Data.Token
	return c.Token, nil
}

type LicenseResponse struct {
	Code int               `json:"code"`
	Data model.LicenseData `json:"data"`
	Msg  string            `json:"msg"`
}

func (c *Client) GetLicense() (*model.LicenseData, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+"/box/license", nil)
	req.Header.Set("Authorization", c.Token)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	var result LicenseResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("获取许可失败: %s", result.Msg)
	}

	return &result.Data, nil
}

func (c *Client) Reauthorize(key string) error {
	url := fmt.Sprintf("%s/box/license?key=%s", c.BaseURL, key)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.Token)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("reauthorize failed with status: %d", resp.StatusCode)
	}
	return nil
}
