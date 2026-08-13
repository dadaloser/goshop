package middlewares

import "github.com/gin-gonic/gin"

// Lookup creates a configured middleware by name. The registry is intentionally
// read-only to callers; framework middleware such as recovery and request
// logging is installed by restserver itself, while CORS requires explicit
// options and is handled separately.
func Lookup(name string) (gin.HandlerFunc, bool) {
	switch name {
	case "context":
		return Context(), true
	case "security-headers":
		return SecurityHeaders(), true
	default:
		return nil, false
	}
}
