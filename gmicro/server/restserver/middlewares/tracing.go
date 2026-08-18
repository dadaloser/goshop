package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func TracingHandler(service string) gin.HandlerFunc {
	return otelgin.Middleware(service, otelgin.WithFilter(func(req *http.Request) bool {
		return !isLowNoiseManagementPath(req.URL.Path)
	}))
}
