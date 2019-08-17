package views

import (
	"backend/common/clog"
	"backend/common/httputil"
	"code/controller"
	"third/gin"
)

//CreateCode请求参数
type CreateCodeReq struct {
	UserId string `json:"user_id" form:"user_id"`
}

//CreateCode返还参数
type CreateCodeResp struct {
	Resp struct {
		Status string `json:"status"`
		Data   string `json:"data"`
		Desc   string `json:"description"`
	}
}

func CreateCode(c *gin.Context) {
	req := CreateCodeReq{}
	resp := CreateCodeResp{}.Resp
	if !c.Bind(&req) {
		clog.Errorf("parse http body err:%+v", req)
		httputil.GinErrRsp(c, nil, "参数错误")
		return
	}
	data, state := controller.CreatCode(req.UserId)
	resp.Data = data.ActiveCode
	switch state {
	case -1:
		httputil.GinOKRsp(c, nil, "无此权限")
		return
	case 1:
		httputil.GinOKRsp(c, resp.Data, "生成成功")
		return
	case 0:
		httputil.GinOKRsp(c, nil, "生成失败")
		return
	}
}

//CheckCode请求参数
type CheckCodeReq struct {
	Code string `json:"code" form:"code"`
}

//CheckCode返还参数
type CheckCodeResp struct {
	Resp struct {
		Status string `json:"status"`
		Data   bool   `json:"data"`
		Desc   string `json:"description"`
	}
}

func CheckCode(c *gin.Context) {
	req := CheckCodeReq{}
	resp := CheckCodeResp{}.Resp
	if !c.Bind(&req) {
		clog.Errorf("parse http body err:%+v", req)
		httputil.GinErrRsp(c, nil, "参数错误")
		return
	}
	resp.Data = controller.CheckCode(req.Code)
	if resp.Data {
		httputil.GinOKRsp(c, resp.Data, "有效")
		return
	} else {
		httputil.GinOKRsp(c, resp.Data, "无效")
		return
	}
}

//ConsumeCode请求参数
type ConsumeCodeReq struct {
	ConsumeUserId string `json:"consume_user_id" form:"consume_user_id"`
	Code          string `json:"code" form:"code"`
}

//ConsumeCode返还参数
type ConsumeCodeResp struct {
	Resp struct {
		Status string `json:"status"`
		Data   bool   `json:"data"`
		Desc   string `json:"description"`
	}
}

//使用激活码
func ConsumeCode(c *gin.Context) {
	req := ConsumeCodeReq{}
	if !c.Bind(&req) {
		clog.Errorf("parse http body err:%+v", req)
		httputil.GinErrRsp(c, nil, "参数错误")
		return
	}
	consumeState := controller.ConsumeCode(req.ConsumeUserId, req.Code)
	if !consumeState {
		httputil.GinErrRsp(c, false, "激活失败")
	} else {
		httputil.GinOKRsp(c, true, "激活成功")
	}
}
