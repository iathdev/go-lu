package middleware

import (
	"learning-go/internal/auth/application/port"
	"learning-go/internal/shared/common"
	"learning-go/internal/shared/ctxlog"
	"learning-go/internal/shared/logger"
	"learning-go/internal/shared/response"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AuthMiddleware(prepService port.PrepUserServicePort, authUseCase port.AuthUseCasePort) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			logger.Debug(c.Request.Context(), "[AUTH] auth rejected",
				zap.String("reason", "missing or invalid authorization header"),
				zap.String("client_ip", common.ResolveClientIP(c.Request)),
			)
			response.Unauthorized(c, "common.unauthorized")
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		ctx := c.Request.Context()

		prepUser, err := prepService.ValidateToken(ctx, token)
		if err != nil {
			response.HandleError(c, err)
			c.Abort()
			return
		}

		user, err := authUseCase.UpsertFromPrepUser(ctx, prepUser)
		if err != nil {
			response.HandleError(c, err)
			c.Abort()
			return
		}

		c.Set("user_id", user.ID.String())
		c.Set("prep_user", prepUser)

		ctx = ctxlog.WithFields(ctx, zap.String("user_id", user.ID.String()))
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
