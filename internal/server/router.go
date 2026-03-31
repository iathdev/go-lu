package server

import (
	"net/http"

	"learning-go/internal/auth"
	"learning-go/internal/infrastructure/config"
	"learning-go/internal/shared/middleware"
	"learning-go/internal/shared/response"
	"learning-go/internal/vocabulary"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"gorm.io/gorm"
)

func NewRouter(
	authModule *auth.Module,
	vocabularyModule *vocabulary.Module,
	db *gorm.DB,
	redisClient *redis.Client,
	cfg *config.Config,
) *gin.Engine {
	if cfg.GinMode != "" {
		gin.SetMode(cfg.GinMode)
	}
	r := gin.New()
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RequestIDMiddleware())

	// OpenTelemetry tracing (no-op if OTLP_ENDPOINT is not configured)
	if cfg.OTLPEndpoint != "" {
		r.Use(otelgin.Middleware(cfg.GetServiceName()))
	}

	r.Use(middleware.RequestLoggerMiddleware())
	r.Use(middleware.LanguageMiddleware())
	r.Use(middleware.RecoveryMiddleware())

	r.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "common.route_not_found")
	})

	r.NoMethod(func(c *gin.Context) {
		response.MethodNotAllowed(c, "common.method_not_allowed")
	})

	// Health check
	r.GET("/health", healthHandler(db))

	// API docs (Swagger UI)
	r.StaticFile("/docs", ".claude/docs/swagger.html")
	r.StaticFile("/docs/openapi.yaml", ".claude/docs/openapi.yaml")

	// Public routes with rate limiting
	public := r.Group("/api")
	public.Use(middleware.GlobalRateLimitMiddleware(redisClient, cfg.GetRateLimitRPS(), cfg.GetRateLimitBurst()))

	// Protected routes
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware(authModule.PrepUserService, authModule.AuthUseCase))

	// Register modules
	authModule.RegisterRoutes(public, protected)
	vocabularyModule.RegisterRoutes(public, protected)

	return r
}

func healthHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "cannot get database connection",
			})
			return
		}

		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "database ping failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	}
}
