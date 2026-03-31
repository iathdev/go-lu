package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		if strings.HasPrefix(c.Request.URL.Path, "/docs") {
			c.Header("Content-Security-Policy", "default-src 'self'; script-src 'unsafe-inline' https://unpkg.com; style-src 'unsafe-inline' https://unpkg.com; img-src 'self' data: https://unpkg.com")
		} else {
			c.Header("Content-Security-Policy", "default-src 'self'")
		}
		c.Next()
	}
}
