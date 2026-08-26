package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	UsernameKey = "username"
	KeyUserID   = "user_id"
	UserIP      = "ip"
)

// 为每个请求添加上下文
func Context() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil {
			return
		}
		if userIP := strings.TrimSpace(c.ClientIP()); userIP != "" {
			c.Set(UserIP, userIP)
		}
		if value := c.Request.Context().Value(UsernameKey); value != nil {
			if _, exists := c.Get(UsernameKey); !exists {
				c.Set(UsernameKey, value)
			}
		}
		if value := c.Request.Context().Value(KeyUserID); value != nil {
			if _, exists := c.Get(KeyUserID); !exists {
				c.Set(KeyUserID, value)
			}
		}

		c.Next()
	}
}
