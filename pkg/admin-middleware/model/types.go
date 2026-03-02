package model

// Admin API Models

type AdminCaptchaData struct {
	CaptchaID  string `json:"captchaId"`
	CaptchaPic string `json:"captchaPic"`
}

type AdminLoginPayload struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	CaptchaID  string `json:"captchaId"`
	CaptchaKey string `json:"captchaKey"`
}

type AdminLoginResponse struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   struct {
		Token    string      `json:"token"`
		UserInfo interface{} `json:"userInfo"`
	} `json:"data"`
}

type AuthCodePayload struct {
	ID           int    `json:"id"`
	Supervise    bool   `json:"supervise"`
	Type         int    `json:"type"`
	Name         string `json:"name"`
	SerialNumber string `json:"SerialNumber"`
	Title        string `json:"title"`
	MgtCenter    string `json:"mgtCenter"`
	Limit        int    `json:"limit"`
	Day          int    `json:"day"`
	Desc         string `json:"desc"`
}

type AuthCodeResponse struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
}

type AuthCodeItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Title        string `json:"title"`
	SerialNumber string `json:"SerialNumber"`
	Limit        int    `json:"limit"`
	Used         int    `json:"used"`
	Day          int    `json:"day"`
	Supervise    bool   `json:"supervise"`
	Type         int    `json:"type"`
	Desc         string `json:"desc"`
	MgtCenter    string `json:"mgtCenter"`
	Online       bool   `json:"online"`
}

type AuthSearchResult struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   struct {
		Total    int            `json:"total"`
		DataList []AuthCodeItem `json:"dataList"`
	} `json:"data"`
}

type DecryptPasswordPayload struct {
	Code string `json:"code"`
}

type DecryptPasswordResponse struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   string `json:"data"`
}
