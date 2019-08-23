package views

import (
	"backend/common/clog"
	"backend/common/httputil"
	"third/gin"
	"web/model"
)

type GetUrlListReq struct {
}

type GetUrlListResp struct {
	Resp struct {
		Status string              `json:"status"`
		Data   []model.RotaryChart `json:"data"`
		Desc   string              `json:"description"`
	}
}

//获取图片的url地址
func GetUrlList(c *gin.Context) {
	rotaryChart := model.RotaryChart{}
	if data, err := rotaryChart.GetAll(); err != nil {
		httputil.GinErrRsp(c, "", "查询失败")
	} else {
		httputil.GinOKRsp(c, data, "查询成功")
	}
}

//获取成员信息请求参数
type GetIotuMembersReq struct {
	Limit  int `json:"limit" form:"limit"`   //每页容量
	Offset int `json:"offset" form:"offset"` //起始行
}

//获取成员信息响应参数
type GetIotuMembersResp struct {
	Resp struct {
		Status string             `json:"status"`
		Data   []model.IotuMember `json:"data"`
		Desc   string             `json:"description"`
	}
}

func GetIotuMembers(c *gin.Context) {
	req := GetIotuMembersReq{}
	if !c.Bind(&req) {
		clog.Errorf("绑定参数失败")
		httputil.GinErrRsp(c, "", "入参错误")
		return
	}
	iotuMember := model.IotuMember{}
	datalist, err := iotuMember.GetAll(req.Limit, req.Offset)
	if err != nil {
		clog.Errorf("查询成员信息失败 [err = %s]", err)
		httputil.GinErrRsp(c, "", "查询失败")
		return
	} else {
		httputil.GinOKRsp(c, datalist, "查询成功")
		return
	}
}

//获取栏目信息请求参数
type GetColumnDataReq struct {
	Limit  int `json:"limit" form:"limit"`   //每页容量
	Offset int `json:"offset" form:"offset"` //起始行
}

//获取栏目信息响应参数
type GetColumnDataResp struct {
	Resp struct {
		Status string             `json:"status"`
		Data   []model.ColumnData `json:"data"`
		Desc   string             `json:"description"`
	}
}

func GetColumnData(c *gin.Context) {
	req := GetColumnDataReq{}
	if !c.Bind(&req) {
		clog.Errorf("绑定参数失败")
		httputil.GinErrRsp(c, "", "入参错误")
		return
	}
	columnData := model.ColumnData{}
	datalist, err := columnData.GetAll(req.Limit, req.Offset)
	if err != nil {
		clog.Errorf("查询栏目失败 [err = %s]", err)
		httputil.GinErrRsp(c, "", "查询失败")
		return
	} else {
		httputil.GinOKRsp(c, datalist, "查询成功")
		return
	}
}
