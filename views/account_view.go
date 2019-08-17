package views

import (
	"backend/common/clog"
	"backend/common/httputil"
	"code/basic"
	"code/controller"
	"third/gin"
)

func AccountRegister(c *gin.Context) {
	req := basic.AccountRegisterReq{}
	if !c.Bind(&req) {
		clog.Errorf("绑定参数失败")
		httputil.GinErrRsp(c, "", "入参错误")
		return
	}
	err := controller.AccountRegister(req)
	if err != nil {
		httputil.GinErrRsp(c, "", "注册失败")
		return
	} else {
		httputil.GinOKRsp(c, "", "注册成功")
		return
	}
}

func AccountLogin(c *gin.Context) {
	req := basic.AccountLoginReq{}
	//resp := basic.AccountLoginResp{}
	if !c.Bind(&req) {
		clog.Errorf("绑定参数失败")
		httputil.GinErrRsp(c, "", "入参错误")
		return
	}
	token, err := controller.AccountLogin(req)
	if err != nil {
		httputil.GinErrRsp(c, "", "登录失败")
	} else {
		c.Writer.Header().Add("token", token)
		httputil.GinOKRsp(c, "", "登陆成功")
	}
}
