package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"learning-go/internal/ocr/application/port"
	apperr "learning-go/internal/shared/error"
	"learning-go/internal/shared/logger"
)

const maxBackoff = 5 * time.Second

type OCRRetryDecorator struct {
	inner      port.OCRServicePort
	maxRetries int
	baseDelay  time.Duration
}

func NewOCRRetryDecorator(inner port.OCRServicePort, maxRetries int, baseDelay time.Duration) port.OCRServicePort {
	return &OCRRetryDecorator{
		inner:      inner,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

func (decorator *OCRRetryDecorator) Recognize(ctx context.Context, req port.OCRRequest) (*port.OCRResult, error) {
	var lastErr error
	for attempt := range decorator.maxRetries {
		result, err := decorator.inner.Recognize(ctx, req)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !isRetryable(err) || attempt == decorator.maxRetries-1 {
			break
		}

		backoff := decorator.baseDelay * time.Duration(1<<attempt)
		if backoff > maxBackoff {
			backoff = maxBackoff
		}

		logger.Warn(ctx, "[OCR] retrying",
			zap.Int("attempt", attempt+1),
			zap.Int("max", decorator.maxRetries),
			zap.Duration("backoff", backoff),
			zap.Error(err),
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, lastErr
}

func (decorator *OCRRetryDecorator) ExtractText(ctx context.Context, req port.OCRRequest) (*port.OCRTextResult, error) {
	var lastErr error
	for attempt := range decorator.maxRetries {
		result, err := decorator.inner.ExtractText(ctx, req)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !isRetryable(err) || attempt == decorator.maxRetries-1 {
			break
		}

		backoff := decorator.baseDelay * time.Duration(1<<attempt)
		if backoff > maxBackoff {
			backoff = maxBackoff
		}

		logger.Warn(ctx, "[OCR] retrying extract-text",
			zap.Int("attempt", attempt+1),
			zap.Int("max", decorator.maxRetries),
			zap.Duration("backoff", backoff),
			zap.Error(err),
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, lastErr
}

func isRetryable(err error) bool {
	appErr, ok := apperr.IsAppError(err)
	if !ok {
		return true
	}
	return appErr.Code() == apperr.CodeServiceUnavailable || appErr.Code() == apperr.CodeInternalServerError
}
