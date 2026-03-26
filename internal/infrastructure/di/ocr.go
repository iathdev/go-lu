package di

import (
	"learning-go/internal/infrastructure/circuitbreaker"
	"learning-go/internal/infrastructure/config"
	"learning-go/internal/ocr"
	ocrservice "learning-go/internal/ocr/adapter/service"
	ocrport "learning-go/internal/ocr/application/port"
	sharederror "learning-go/internal/shared/error"
	"learning-go/internal/shared/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type ocrResult struct {
	module   *ocr.Module
	cleanups []func()
}

func (result *ocrResult) cleanup() {
	for i := len(result.cleanups) - 1; i >= 0; i-- {
		result.cleanups[i]()
	}
}

func initOCR(cfg *config.Config, redisClient *redis.Client) ocrResult {
	engines := ocrport.OCREngineRegistry{}
	var cleanups []func()

	withRetry := func(engine ocrport.OCRServicePort) ocrport.OCRServicePort {
		return ocrservice.NewOCRRetryDecorator(engine, cfg.GetOCRRetryMax(), cfg.GetOCRRetryDelay())
	}

	register := func(key ocrport.OCREngineKey, engine ocrport.OCRServicePort, cleanupFn ...func()) {
		engines[key] = engine
		if len(cleanupFn) > 0 {
			cleanups = append(cleanups, cleanupFn[0])
		}
	}

	if cfg.OCRServiceURL != "" {
		paddleAdapter := ocrservice.NewPaddleOCRService(cfg.OCRServiceURL, newOCRBreaker("paddle-ocr"))
		register(ocrport.OCREnginePaddleOCR, withRetry(paddleAdapter))

		tessAdapter := ocrservice.NewTesseractOCRService(cfg.OCRServiceURL, newOCRBreaker("tesseract"))
		register(ocrport.OCREngineTesseract, withRetry(tessAdapter))
	}

	if cfg.GoogleApplicationCredentials != "" {
		adapter, cleanup, err := ocrservice.NewGoogleVisionService(cfg.GoogleApplicationCredentials, newOCRBreaker("google-vision"))
		if err != nil {
			logger.Warn("[DI] Google Vision init failed, skipping", zap.Error(err))
		} else {
			register(ocrport.OCREngineGoogleVision, withRetry(adapter), cleanup)
		}
	}

	if cfg.BaiduOCRAPIKey != "" && cfg.BaiduOCRSecretKey != "" {
		adapter := ocrservice.NewBaiduOCRService(cfg.BaiduOCRAPIKey, cfg.BaiduOCRSecretKey, newOCRBreaker("baidu-ocr"), redisClient)
		register(ocrport.OCREngineBaiduOCR, withRetry(adapter))
	}

	return ocrResult{
		module:   ocr.NewModule(engines),
		cleanups: cleanups,
	}
}

func newOCRBreaker(name string) *circuitbreaker.Breaker {
	return circuitbreaker.NewBreaker(circuitbreaker.BreakerConfig{
		Name: name,
	}, func(err error) bool {
		if err == nil {
			return true
		}
		if appErr, ok := sharederror.IsAppError(err); ok {
			return appErr.Code() == sharederror.CodeNotFound
		}
		return false
	})
}
