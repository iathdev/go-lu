package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"learning-go/internal/infrastructure/circuitbreaker"
	apperr "learning-go/internal/shared/error"
	"learning-go/internal/shared/logger"
	"learning-go/internal/ocr/application/port"
)

const (
	baiduTokenURL       = "https://aip.baidubce.com/oauth/2.0/token"
	baiduHandwritingURL = "https://aip.baidubce.com/rest/2.0/ocr/v1/handwriting"
	baiduGeneralURL     = "https://aip.baidubce.com/rest/2.0/ocr/v1/general_basic"

	baiduTokenCacheKey = "baidu_ocr:access_token"
	baiduTokenTTL      = 29 * 24 * time.Hour
)

type BaiduOCRService struct {
	apiKey    string
	secretKey string
	client    *http.Client
	breaker   *circuitbreaker.Breaker
	redis     *redis.Client
}

type baiduTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type baiduOCRResponse struct {
	WordsResult []baiduWordsResult `json:"words_result"`
	ErrorCode   int                `json:"error_code"`
	ErrorMsg    string             `json:"error_msg"`
}

type baiduWordsResult struct {
	Words    string            `json:"words"`
	Chars    []baiduCharResult `json:"chars"`
	Location baiduLocation     `json:"location"`
}

type baiduCharResult struct {
	Char        string `json:"char"`
	Probability int    `json:"probability"`
}

type baiduLocation struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func NewBaiduOCRService(apiKey, secretKey string, breaker *circuitbreaker.Breaker, redisClient *redis.Client) port.OCRServicePort {
	return &BaiduOCRService{
		apiKey:    apiKey,
		secretKey: secretKey,
		client:    &http.Client{Timeout: 10 * time.Second},
		breaker:   breaker,
		redis:     redisClient,
	}
}

func (svc *BaiduOCRService) Recognize(ctx context.Context, req port.OCRRequest) (*port.OCRResult, error) {
	result, err := svc.breaker.Execute(func() (any, error) {
		token, err := svc.getAccessToken(ctx)
		if err != nil {
			return nil, err
		}

		endpoint := baiduGeneralURL
		if req.Language == "zh" {
			endpoint = baiduHandwritingURL
		}

		imageB64 := base64.StdEncoding.EncodeToString(req.Image)
		form := url.Values{
			"image":                 {imageB64},
			"recognize_granularity": {"small"},
		}

		reqURL := fmt.Sprintf("%s?access_token=%s", endpoint, token)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, apperr.InternalServerError("common.internal_error", err)
		}
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := svc.client.Do(httpReq)
		if err != nil {
			logger.Error(ctx, "[OCR] Baidu connection failed", zap.Error(err))
			return nil, apperr.ServiceUnavailable("ocr.service_connection_failed", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			logger.Error(ctx, "[OCR] Baidu returned error",
				zap.Int("status", resp.StatusCode),
				zap.String("response", string(respBody)),
			)
			return nil, apperr.ServiceUnavailable("ocr.service_error", fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody)))
		}

		var ocrResp baiduOCRResponse
		if err := json.NewDecoder(resp.Body).Decode(&ocrResp); err != nil {
			logger.Error(ctx, "[OCR] Baidu invalid response", zap.Error(err))
			return nil, apperr.ServiceUnavailable("ocr.service_invalid_response", err)
		}

		if ocrResp.ErrorCode != 0 {
			logger.Error(ctx, "[OCR] Baidu API error",
				zap.Int("error_code", ocrResp.ErrorCode),
				zap.String("error_msg", ocrResp.ErrorMsg),
			)
			return nil, apperr.ServiceUnavailable("ocr.service_error", fmt.Errorf("baidu error %d: %s", ocrResp.ErrorCode, ocrResp.ErrorMsg))
		}

		characters := svc.parseResponse(ocrResp, req.Language)
		return &port.OCRResult{Characters: characters, Engine: "baidu_ocr"}, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*port.OCRResult), nil
}

func (svc *BaiduOCRService) parseResponse(resp baiduOCRResponse, language string) []port.OCRCharacter {
	seen := make(map[string]struct{})
	var characters []port.OCRCharacter

	for _, word := range resp.WordsResult {
		bbox := baiduLocationToBBox(word.Location)
		if len(word.Chars) > 0 {
			for _, char := range word.Chars {
				text := char.Char
				if language == "zh" && !isCJKBaidu(text) {
					continue
				}
				if _, exists := seen[text]; exists {
					continue
				}
				seen[text] = struct{}{}
				characters = append(characters, port.OCRCharacter{
					Text:        text,
					Confidence:  float64(char.Probability) / 100.0,
					BoundingBox: bbox,
				})
			}
		} else {
			text := strings.TrimSpace(word.Words)
			if text == "" {
				continue
			}
			if _, exists := seen[text]; exists {
				continue
			}
			seen[text] = struct{}{}
			characters = append(characters, port.OCRCharacter{
				Text:        text,
				Confidence:  0.5,
				BoundingBox: bbox,
			})
		}
	}

	return characters
}

func (svc *BaiduOCRService) ExtractText(ctx context.Context, req port.OCRRequest) (*port.OCRTextResult, error) {
	result, err := svc.breaker.Execute(func() (any, error) {
		token, err := svc.getAccessToken(ctx)
		if err != nil {
			return nil, err
		}

		endpoint := baiduGeneralURL
		if req.Language == "zh" {
			endpoint = baiduHandwritingURL
		}

		imageB64 := base64.StdEncoding.EncodeToString(req.Image)
		form := url.Values{
			"image":                 {imageB64},
			"recognize_granularity": {"big"},
		}

		reqURL := fmt.Sprintf("%s?access_token=%s", endpoint, token)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, apperr.InternalServerError("common.internal_error", err)
		}
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := svc.client.Do(httpReq)
		if err != nil {
			logger.Error(ctx, "[OCR] Baidu connection failed", zap.Error(err))
			return nil, apperr.ServiceUnavailable("ocr.service_connection_failed", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			logger.Error(ctx, "[OCR] Baidu returned error",
				zap.Int("status", resp.StatusCode),
				zap.String("response", string(respBody)),
			)
			return nil, apperr.ServiceUnavailable("ocr.service_error", fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody)))
		}

		var ocrResp baiduOCRResponse
		if err := json.NewDecoder(resp.Body).Decode(&ocrResp); err != nil {
			logger.Error(ctx, "[OCR] Baidu invalid response", zap.Error(err))
			return nil, apperr.ServiceUnavailable("ocr.service_invalid_response", err)
		}

		if ocrResp.ErrorCode != 0 {
			logger.Error(ctx, "[OCR] Baidu API error",
				zap.Int("error_code", ocrResp.ErrorCode),
				zap.String("error_msg", ocrResp.ErrorMsg),
			)
			return nil, apperr.ServiceUnavailable("ocr.service_error", fmt.Errorf("baidu error %d: %s", ocrResp.ErrorCode, ocrResp.ErrorMsg))
		}

		blocks := make([]port.OCRTextBlock, 0, len(ocrResp.WordsResult))
		for _, word := range ocrResp.WordsResult {
			text := strings.TrimSpace(word.Words)
			if text == "" {
				continue
			}
			blocks = append(blocks, port.OCRTextBlock{
				Text:        text,
				BoundingBox: baiduLocationToBBox(word.Location),
				Confidence:  0.0,
			})
		}

		return &port.OCRTextResult{Blocks: blocks, Engine: "baidu_ocr"}, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*port.OCRTextResult), nil
}

func baiduLocationToBBox(loc baiduLocation) *port.BoundingBox {
	if loc.Width == 0 && loc.Height == 0 {
		return nil
	}
	return &port.BoundingBox{
		Vertices: []port.Point{
			{X: loc.Left, Y: loc.Top},
			{X: loc.Left + loc.Width, Y: loc.Top},
			{X: loc.Left + loc.Width, Y: loc.Top + loc.Height},
			{X: loc.Left, Y: loc.Top + loc.Height},
		},
	}
}

func (svc *BaiduOCRService) getAccessToken(ctx context.Context) (string, error) {
	token, err := svc.redis.Get(ctx, baiduTokenCacheKey).Result()
	if err == nil && token != "" {
		return token, nil
	}

	tokenURL := fmt.Sprintf("%s?grant_type=client_credentials&client_id=%s&client_secret=%s",
		baiduTokenURL, svc.apiKey, svc.secretKey)

	resp, err := svc.client.Post(tokenURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		logger.WithContext(ctx).Error("[OCR] Baidu token refresh failed", zap.Error(err))
		return "", apperr.ServiceUnavailable("ocr.service_connection_failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.WithContext(ctx).Error("[OCR] Baidu token endpoint error",
			zap.Int("status", resp.StatusCode),
			zap.String("response", string(respBody)),
		)
		return "", apperr.ServiceUnavailable("ocr.service_error", fmt.Errorf("token request failed: status %d", resp.StatusCode))
	}

	var tokenResp baiduTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		logger.WithContext(ctx).Error("[OCR] Baidu token invalid response", zap.Error(err))
		return "", apperr.ServiceUnavailable("ocr.service_invalid_response", err)
	}

	if tokenResp.AccessToken == "" {
		return "", apperr.ServiceUnavailable("ocr.service_error", fmt.Errorf("empty access token"))
	}

	svc.redis.Set(ctx, baiduTokenCacheKey, tokenResp.AccessToken, baiduTokenTTL)

	return tokenResp.AccessToken, nil
}

func isCJKBaidu(text string) bool {
	for _, char := range text {
		if unicode.Is(unicode.Han, char) {
			return true
		}
	}
	return false
}
