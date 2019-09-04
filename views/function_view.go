package views

import (
	"backend/common/clog"
	"backend/common/httputil"
	"third/gin"
	"web/model"
)

type GetMenuRep struct {
	classify int
}

type GetMenuResp struct {
	Resp struct {
		Status string       `json:"status"`
		Data   []model.Menu `json:"data"`
		Desc   string       `json:"description"`
	}
}

//获取菜单信息
func GetMenu(c *gin.Context) {
	req := GetMenuRep{}
	if !c.Bind(&req) {
		clog.Errorf("parse http body err:%+v", req)
		httputil.GinErrRsp(c, nil, "参数错误")
		return
	}
	menu := model.Menu{}
	menus, err := menu.GetByClassify(req.classify)
	if err != nil {
		clog.Errorf("获取menu失败 err = %s", err)
	}
	httputil.GinOKRsp(c, menus, "查询成功")
	return
}

type GetMenuClassifyRep struct {
	classify int
}

//获取菜单详情
type GetMenuClassifyResp struct {
	Resp struct {
		Status string               `json:"status"`
		Data   []model.MenuClassify `json:"data"`
		Desc   string               `json:"description"`
	}
}

func GetMenuClassify(c *gin.Context) {
	req := GetMenuClassifyRep{}
	if !c.Bind(&req) {
		clog.Errorf("parse http body err:%+v", req)
		httputil.GinErrRsp(c, nil, "参数错误")
		return
	}
	menuClassify := model.MenuClassify{}
	menuClassifies, err := menuClassify.GetByClassify(req.classify)
	if err != nil {
		clog.Errorf("获取menu失败 err = %s", err)
	}
	httputil.GinOKRsp(c, menuClassifies, "查询成功")
	return
}
