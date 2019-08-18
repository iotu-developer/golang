package basic

//账号注册请求参数
type AccountRegisterReq struct {
	UserName   string `json:"user_name" form:"user_name"`
	NickName   string `json:"nick_name" form:"nick_name"`
	Password   string `json:"password" form:"password"`
	Major      string `json:"major" form:"major"`
	College    string `json:"college" form:"college"`
	ActiveCode string `json:"active_code" form:"active_code"`
}

//账号注册返还参数
type AccountRegisterResp struct {
	Resp struct {
		Status string `json:"status"`
		Data   string `json:"data"`
		Desc   string `json:"description"`
	}
}

//账号登录请求参数
type AccountLoginReq struct {
	UserName string `json:"user_name" form:"user_name"`
	Password string `json:"password" form:"password"`
}

//账号登录返还参数
type AccountLoginResp struct {
	Resp struct {
		Status string `json:"status"`
		Data   string `json:"data"`
		Desc   string `json:"description"`
	}
}
