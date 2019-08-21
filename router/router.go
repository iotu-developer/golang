package router

import (
	"fmt"
	"golang/fliter"
	"golang/views"
	"third/gin"
)

func StartHttpServer() {
	fmt.Println("StartServer")
	router := gin.Default()

	Account := router.Group("/account")
	{
		Account.POST("/register", views.AccountRegister)
		Account.POST("/login", views.AccountLogin)
	}

	//首页
	t := router.Group("/home_page")
	{
		t.GET("/iotu_members", views.GetIotuMembers)
		t.GET("/url_list", views.GetUrlList)
		t.GET("/column_datas", views.GetColumnData)
	}

	// 静态资源返回
	router.Static("/static", "./static")

	//使用Authorize()中间件身份验证
	router.Use(fliter.Authorize())

	//激活码
	code := router.Group("/code")
	{
		code.GET("/Create", views.CreateCode)
		code.GET("/Check", views.CheckCode)
		code.GET("/Consume", views.ConsumeCode)
	}

	_ = router.Run(":7777")
}
