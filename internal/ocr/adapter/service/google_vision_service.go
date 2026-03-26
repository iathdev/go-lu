package service

import (
	"context"
	"unicode"

	vision "cloud.google.com/go/vision/v2/apiv1"
	visionpb "cloud.google.com/go/vision/v2/apiv1/visionpb"
	"google.golang.org/api/option"

	"learning-go/internal/infrastructure/circuitbreaker"
	apperr "learning-go/internal/shared/error"
	"learning-go/internal/shared/logger"
	"learning-go/internal/ocr/application/port"

	"go.uber.org/zap"
)

type GoogleVisionService struct {
	client  *vision.ImageAnnotatorClient
	breaker *circuitbreaker.Breaker
}

func NewGoogleVisionService(credFile string, breaker *circuitbreaker.Breaker) (port.OCRServicePort, func(), error) {
	ctx := context.Background()
	client, err := vision.NewImageAnnotatorClient(ctx,
		option.WithCredentialsFile(credFile),
	)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { client.Close() }
	return &GoogleVisionService{client: client, breaker: breaker}, cleanup, nil
}

func (svc *GoogleVisionService) Recognize(ctx context.Context, req port.OCRRequest) (*port.OCRResult, error) {
	result, err := svc.breaker.Execute(func() (any, error) {
		resp, err := svc.client.BatchAnnotateImages(ctx, &visionpb.BatchAnnotateImagesRequest{
			Requests: []*visionpb.AnnotateImageRequest{
				{
					Image:    &visionpb.Image{Content: req.Image},
					Features: []*visionpb.Feature{{Type: visionpb.Feature_DOCUMENT_TEXT_DETECTION}},
				},
			},
		})
		if err != nil {
			logger.WithContext(ctx).Error("[OCR] Google Vision API error", zap.Error(err))
			return nil, apperr.ServiceUnavailable("ocr.service_error", err)
		}

		responses := resp.GetResponses()
		if len(responses) == 0 || responses[0].GetFullTextAnnotation() == nil {
			return &port.OCRResult{Characters: []port.OCRCharacter{}, Engine: string(port.OCREngineGoogleVision)}, nil
		}

		annotation := responses[0].GetFullTextAnnotation()
		characters := extractCharacters(annotation, req.Language)

		return &port.OCRResult{Characters: characters, Engine: string(port.OCREngineGoogleVision)}, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*port.OCRResult), nil
}

func extractCharacters(annotation *visionpb.TextAnnotation, language string) []port.OCRCharacter {
	seen := make(map[string]struct{})
	var characters []port.OCRCharacter

	for _, page := range annotation.GetPages() {
		for _, block := range page.GetBlocks() {
			for _, paragraph := range block.GetParagraphs() {
				if isCJKLanguage(language) {
					characters = extractChinese(paragraph, seen, characters)
				} else {
					characters = extractWordsWithLang(paragraph, language, seen, characters)
				}
			}
		}
	}

	return characters
}

func extractChinese(paragraph *visionpb.Paragraph, seen map[string]struct{}, characters []port.OCRCharacter) []port.OCRCharacter {
	for _, word := range paragraph.GetWords() {
		var wordText string
		for _, symbol := range word.GetSymbols() {
			text := symbol.GetText()
			if isCJK(text) {
				wordText += text
			}
		}
		if wordText == "" {
			continue
		}
		if _, exists := seen[wordText]; exists {
			continue
		}
		seen[wordText] = struct{}{}
		characters = append(characters, port.OCRCharacter{
			Text:       wordText,
			Confidence: float64(word.GetConfidence()),
		})
	}
	return characters
}

func extractWordsWithLang(paragraph *visionpb.Paragraph, requestedLang string, seen map[string]struct{}, characters []port.OCRCharacter) []port.OCRCharacter {
	for _, word := range paragraph.GetWords() {
		detectedLang := detectWordLanguage(word)
		if detectedLang != "" && detectedLang != requestedLang {
			continue
		}

		var wordText string
		for _, symbol := range word.GetSymbols() {
			wordText += symbol.GetText()
		}
		if wordText == "" {
			continue
		}
		if _, exists := seen[wordText]; exists {
			continue
		}
		seen[wordText] = struct{}{}
		characters = append(characters, port.OCRCharacter{
			Text:       wordText,
			Confidence: float64(word.GetConfidence()),
		})
	}
	return characters
}

func detectWordLanguage(word *visionpb.Word) string {
	if prop := word.GetProperty(); prop != nil {
		for _, detectedLang := range prop.GetDetectedLanguages() {
			return mapBCP47ToLang(detectedLang.GetLanguageCode())
		}
	}
	return ""
}

func mapBCP47ToLang(code string) string {
	switch {
	case code == "zh" || code == "zh-Hans" || code == "zh-Hant" || code == "zh-CN" || code == "zh-TW":
		return "zh"
	case code == "vi":
		return "vi"
	case code == "en":
		return "en"
	default:
		return code
	}
}

func isCJKLanguage(language string) bool {
	switch language {
	case "zh", "ja", "ko":
		return true
	default:
		return false
	}
}

func isCJK(text string) bool {
	for _, char := range text {
		if unicode.Is(unicode.Han, char) {
			return true
		}
	}
	return false
}
