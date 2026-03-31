package middleware

import (
	infrasentry "learning-go/internal/infrastructure/sentry"
	"learning-go/internal/shared/logger"
	"learning-go/internal/shared/response"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Error(c.Request.Context(), "[SERVER] panic recovered",
			zap.Any("panic", recovered),
			zap.String("stack", string(debug.Stack())),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)

		// Report to Sentry (no-op if Sentry DSN is not configured)
		infrasentry.RecoverWithSentry(recovered)

		response.InternalServerError(c, "common.internal_error")
		c.Abort()
	})
}
