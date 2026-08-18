package middlewares

import "github.com/gin-gonic/gin"

// SecurityHeadersOptions configures HTTP response hardening headers.
type SecurityHeadersOptions struct {
	ContentSecurityPolicy   string
	StrictTransportSecurity string
	FrameOptions            string
	ReferrerPolicy          string
	PermissionsPolicy       string
}

// SecurityHeaders returns a middleware with conservative security defaults.
func SecurityHeaders() gin.HandlerFunc {
	return SecurityHeadersWithOptions(SecurityHeadersOptions{})
}

// SecurityHeadersWithOptions returns a middleware that sets common browser
// security headers. Empty option fields fall back to production-safe defaults.
func SecurityHeadersWithOptions(opts SecurityHeadersOptions) gin.HandlerFunc {
	if opts.ContentSecurityPolicy == "" {
		opts.ContentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"
	}
	if opts.StrictTransportSecurity == "" {
		opts.StrictTransportSecurity = "max-age=31536000"
	}
	if opts.FrameOptions == "" {
		opts.FrameOptions = "DENY"
	}
	if opts.ReferrerPolicy == "" {
		opts.ReferrerPolicy = "no-referrer"
	}
	if opts.PermissionsPolicy == "" {
		opts.PermissionsPolicy = "camera=(), microphone=(), geolocation=()"
	}

	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", opts.FrameOptions)
		c.Header("Referrer-Policy", opts.ReferrerPolicy)
		c.Header("Permissions-Policy", opts.PermissionsPolicy)
		c.Header("Content-Security-Policy", opts.ContentSecurityPolicy)
		c.Header("Strict-Transport-Security", opts.StrictTransportSecurity)
		c.Next()
	}
}
