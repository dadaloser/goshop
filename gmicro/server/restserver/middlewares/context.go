package middlewares

import (
	"github.com/gin-gonic/gin"
)

const (
	UsernameKey = "username"
	KeyUserID   = "user_id"
	UserIP      = "ip"
)

// 为每个请求添加上下文, django
func Context() gin.HandlerFunc {
	return func(c *gin.Context) {
		//TODO 大家自己去扩展
		//c.Set(UsernameKey, c.Request.Context().Value(UsernameKey))
		//c.Set(KeyUserID, c.Request.Context().Value(KeyUserID))

		c.Next()
	}
}
