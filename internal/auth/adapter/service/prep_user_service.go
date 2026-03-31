package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"learning-go/internal/auth/application/port"
	"learning-go/internal/auth/domain"
	"learning-go/internal/infrastructure/cache"
	"learning-go/internal/infrastructure/circuitbreaker"
	apperr "learning-go/internal/shared/error"
	"learning-go/internal/shared/logger"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type PrepAuthConfig struct {
	BaseDomain        string
	AuthTokenEndpoint string
	AuthMeEndpoint    string
	Timeout           time.Duration
}

type PrepUserService struct {
	cfg        PrepAuthConfig
	client     *http.Client
	breaker    *circuitbreaker.Breaker
	tokenCache cache.Cache[domain.PrepUser]
}

type prepAuthMeResponse struct {
	Data    prepAuthMeData `json:"data"`
	Message string     `json:"message"`
}

type prepAuthMeData struct {
	ID                  int64  `json:"id"`
	Email               string `json:"email"`
	Name                string `json:"name"`
	IsFirstLogin        bool   `json:"is_first_login"`
	ForceUpdatePassword bool   `json:"force_update_password"`
}

type prepAuthTokenResponse struct {
	Data    prepAuthTokenData `json:"data"`
	Message string            `json:"message"`
}

type prepAuthTokenData struct {
	ReturnURL string `json:"return_url"`
}

func NewPrepUserService(cfg PrepAuthConfig, breaker *circuitbreaker.Breaker, tokenCache cache.Cache[domain.PrepUser]) port.PrepUserServicePort {
	return &PrepUserService{
		cfg:        cfg,
		client:     &http.Client{Timeout: cfg.Timeout},
		breaker:    breaker,
		tokenCache: tokenCache,
	}
}

func (service *PrepUserService) ValidateToken(ctx context.Context, token string) (*domain.PrepUser, error) {
	cacheKey := hashToken(token)

	if cached := service.getFromCache(ctx, cacheKey); cached != nil {
		return cached, nil
	}

	prepUser, err := service.callPrepMe(ctx, token)
	if err != nil {
		return nil, err
	}

	service.setToCache(ctx, cacheKey, prepUser)

	return prepUser, nil
}

func (service *PrepUserService) getFromCache(ctx context.Context, key string) *domain.PrepUser {
	cached, err := service.tokenCache.Get(ctx, key)
	if err != nil {
		logger.Debug(ctx, "[AUTH] token cache read failed", zap.Error(err))
	}
	return cached
}

func (service *PrepUserService) setToCache(ctx context.Context, key string, prepUser *domain.PrepUser) {
	if err := service.tokenCache.Set(ctx, key, prepUser); err != nil {
		logger.Debug(ctx, "[AUTH] token cache write failed", zap.Error(err))
	}
}

func (service *PrepUserService) callPrepMe(ctx context.Context, token string) (*domain.PrepUser, error) {
	result, err := service.breaker.Execute(func() (any, error) {
		endpoint := service.cfg.BaseDomain + service.cfg.AuthMeEndpoint

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := service.client.Do(req)
		if err != nil {
			logger.Error(ctx, "[AUTH] Prep connection failed", zap.String("endpoint", endpoint), zap.Error(err))
			return nil, apperr.ServiceUnavailable("common.service_unavailable", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusUnauthorized {
			authErr := apperr.Unauthorized("common.unauthorized")
			if returnURL := service.fetchReturnURL(ctx, token); returnURL != "" {
				authErr = authErr.WithData(map[string]any{"return_url": returnURL})
			}
			return nil, authErr
		}

		if resp.StatusCode != http.StatusOK {
			logger.Error(ctx, "[AUTH] Prep unexpected status", zap.String("endpoint", endpoint), zap.Int("status", resp.StatusCode))
			return nil, apperr.ServiceUnavailable("common.service_unavailable", fmt.Errorf("unexpected status: %s", resp.Status))
		}

		var body prepAuthMeResponse
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			logger.Error(ctx, "[AUTH] Prep decode failed", zap.String("endpoint", endpoint), zap.Error(err))
			return nil, apperr.ServiceUnavailable("common.service_unavailable", err)
		}

		return domain.NewPrepUser(body.Data.ID, body.Data.Email, body.Data.Name, body.Data.IsFirstLogin, body.Data.ForceUpdatePassword), nil
	})
	if err != nil {
		return nil, err
	}

	return result.(*domain.PrepUser), nil
}

func hashToken(token string) string {
	hash := md5.Sum([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (service *PrepUserService) fetchReturnURL(ctx context.Context, token string) string {
	if service.cfg.BaseDomain == "" {
		return ""
	}

	endpoint := service.cfg.BaseDomain + service.cfg.AuthTokenEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := service.client.Do(req)
	if err != nil {
		logger.Warn(ctx, "[AUTH] failed to fetch return_url", zap.String("endpoint", endpoint), zap.Error(err))
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	var body prepAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return ""
	}

	return body.Data.ReturnURL
}
