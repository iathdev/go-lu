package auth

import (
	"learning-go/internal/auth/adapter/handler"
	"learning-go/internal/auth/adapter/repository"
	"learning-go/internal/auth/adapter/service"
	"learning-go/internal/auth/application/port"
	"learning-go/internal/auth/application/usecase"
	"learning-go/internal/auth/domain"
	"learning-go/internal/infrastructure/cache"
	"learning-go/internal/infrastructure/circuitbreaker"
	"learning-go/internal/infrastructure/config"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Module struct {
	handler         *handler.AuthHandler
	PrepUserService port.PrepUserServicePort
	AuthUseCase     port.AuthUseCasePort
}

func NewModule(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *Module {
	userRepo := repository.NewUserRepository(db)

	tokenCache := cache.New[domain.PrepUser](redisClient, cache.Options{
		KeyPrefix: "auth:token:",
		TTL:       cfg.GetPrepTokenCacheTTL(),
	})

	breaker := circuitbreaker.NewBreaker(circuitbreaker.BreakerConfig{
		Name: "prep-user-service",
	}, nil)
	prepUserService := service.NewPrepUserService(
		service.PrepAuthConfig{
			BaseDomain:        cfg.PrepAPIGatewayDomain,
			AuthTokenEndpoint: cfg.GetPrepAuthTokenEndpoint(),
			AuthMeEndpoint:    cfg.GetPrepMeEndpoint(),
			Timeout:           cfg.GetPrepHTTPClientTimeout(),
		},
		breaker,
		tokenCache)

	authUseCase := usecase.NewAuthUseCase(userRepo)
	authHandler := handler.NewAuthHandler(authUseCase)

	return &Module{
		handler:         authHandler,
		PrepUserService: prepUserService,
		AuthUseCase:     authUseCase,
	}
}

func (module *Module) RegisterRoutes(public, protected *gin.RouterGroup) {
	v1 := protected.Group("/v1/auth")
	v1.GET("/me", module.handler.GetMe)
}
