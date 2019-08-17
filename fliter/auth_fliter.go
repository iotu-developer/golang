package fliter

import (
	"code/util"
	"net/http"
	"third/gin"
)

func Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("token") // 访问令牌
		if token == "" {
			c.Abort()
			c.JSON(http.StatusUnauthorized, gin.H{"state": -1, "message": "访问未授权"})
			return
		} else if util.CheckToken(token) {
			// 验证通过，会继续访问下一个中间件
			c.Next()
		} else {
			// 验证不通过，不再调用后续的函数处理
			c.Abort()
			c.JSON(http.StatusUnauthorized, gin.H{"state": -1, "message": "访问未授权"})
			return
		}
	}
}
