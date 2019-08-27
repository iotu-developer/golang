package views

import (
	"backend/common/clog"
	"backend/common/httputil"
	"third/gin"
	"web/basic"
	"web/controller"
)

func AccountRegister(c *gin.Context) {
	clog.Warnf("正在注册")
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
	clog.Warnf("正在登录")
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

func CheckUserName(c *gin.Context) {
	req := basic.CheckUserNameReq{}
	if !c.Bind(&req) {
		clog.Errorf("绑定参数失败")
		httputil.GinErrRsp(c, "", "入参错误")
		return
	}
	state := controller.CheckUserName(req.UserName)
	httputil.GinOKRsp(c, state, "查询成功")
}
