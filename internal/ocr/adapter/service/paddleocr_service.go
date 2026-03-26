package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"learning-go/internal/infrastructure/circuitbreaker"
	"learning-go/internal/ocr/application/port"
	apperr "learning-go/internal/shared/error"
	"learning-go/internal/shared/logger"
)

type PaddleOCRService struct {
	baseURL string
	client  *http.Client
	breaker *circuitbreaker.Breaker
}

func NewPaddleOCRService(baseURL string, breaker *circuitbreaker.Breaker) port.OCRServicePort {
	return &PaddleOCRService{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
		breaker: breaker,
	}
}

func (service *PaddleOCRService) Recognize(ctx context.Context, req port.OCRRequest) (*port.OCRResult, error) {
	result, err := service.breaker.Execute(func() (any, error) {
		return callSelfHostedOCR(ctx, service.client, service.baseURL, "paddleocr", req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*port.OCRResult), nil
}

type selfHostedOCRRequest struct {
	Image    string `json:"image"`
	Language string `json:"language"`
	Engine   string `json:"engine,omitempty"`
}

type selfHostedOCRResponse struct {
	Characters []selfHostedOCRCharacter `json:"characters"`
	Engine     string                   `json:"engine"`
}

type selfHostedOCRCharacter struct {
	Text       string   `json:"text"`
	Pinyin     string   `json:"pinyin"`
	Confidence float64  `json:"confidence"`
	Candidates []string `json:"candidates"`
}

func callSelfHostedOCR(ctx context.Context, client *http.Client, baseURL string, engine string, req port.OCRRequest) (*port.OCRResult, error) {
	language := req.Language
	if language == "" {
		language = "zh"
	}

	payload := selfHostedOCRRequest{
		Image:    base64.StdEncoding.EncodeToString(req.Image),
		Language: language,
		Engine:   engine,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/recognize", bytes.NewReader(body))
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		logger.WithContext(ctx).Error("[OCR] service connection failed", zap.Error(err))
		return nil, apperr.ServiceUnavailable("ocr.service_connection_failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		statusErr := fmt.Errorf("status %s: %s", resp.Status, string(respBody))
		logger.WithContext(ctx).Error("[OCR] service returned error", zap.Int("status", resp.StatusCode), zap.String("response", string(respBody)))
		return nil, apperr.ServiceUnavailable("ocr.service_error", statusErr)
	}

	var ocrResp selfHostedOCRResponse
	if err := json.NewDecoder(resp.Body).Decode(&ocrResp); err != nil {
		logger.WithContext(ctx).Error("[OCR] failed to decode response", zap.Error(err))
		return nil, apperr.ServiceUnavailable("ocr.service_invalid_response", err)
	}

	characters := make([]port.OCRCharacter, 0, len(ocrResp.Characters))
	for _, char := range ocrResp.Characters {
		characters = append(characters, port.OCRCharacter{
			Text:          char.Text,
			Pronunciation: char.Pinyin,
			Confidence:    char.Confidence,
			Candidates:    char.Candidates,
		})
	}

	return &port.OCRResult{
		Characters: characters,
		Engine:     ocrResp.Engine,
	}, nil
}
