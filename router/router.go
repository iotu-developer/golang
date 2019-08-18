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
	router.Use(fliter.Authorize()) //使用Authorize()中间件身份验证
	// 静态资源返回
	router.Static("/static", "./static")
	code := router.Group("/code")
	{
		code.GET("/Create", views.CreateCode)
		code.GET("/Check", views.CheckCode)
		code.GET("/Consume", views.ConsumeCode)
	}

	_ = router.Run(":7777")
}
