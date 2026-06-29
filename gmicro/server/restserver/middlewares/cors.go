package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type CorsOptions struct {
	AllowOrigins     []string
	AllowCredentials bool
}

func Cors() gin.HandlerFunc {
	return CorsWithOptions(CorsOptions{
		AllowOrigins: []string{"*"},
	})
}

func CorsWithOptions(opts CorsOptions) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{}, len(opts.AllowOrigins))
	for _, origin := range opts.AllowOrigins {
		allowedOrigins[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		allowOrigin := "*"
		if _, ok := allowedOrigins["*"]; !ok {
			allowOrigin = ""
			if _, ok := allowedOrigins[origin]; ok {
				allowOrigin = origin
			}
		}

		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
		}
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token, x-token")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE, PATCH, PUT")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		if opts.AllowCredentials && allowOrigin != "*" && allowOrigin != "" {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
	}
}
