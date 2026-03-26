package service

import (
	"context"
	"net/http"
	"time"

	"learning-go/internal/infrastructure/circuitbreaker"
	"learning-go/internal/ocr/application/port"
)

type TesseractOCRService struct {
	baseURL string
	client  *http.Client
	breaker *circuitbreaker.Breaker
}

func NewTesseractOCRService(baseURL string, breaker *circuitbreaker.Breaker) port.OCRServicePort {
	return &TesseractOCRService{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
		breaker: breaker,
	}
}

func (service *TesseractOCRService) Recognize(ctx context.Context, req port.OCRRequest) (*port.OCRResult, error) {
	result, err := service.breaker.Execute(func() (any, error) {
		return callSelfHostedOCR(ctx, service.client, service.baseURL, "tesseract", req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*port.OCRResult), nil
}
