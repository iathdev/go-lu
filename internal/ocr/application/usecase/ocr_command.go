package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"

	apperr "learning-go/internal/shared/error"
	"learning-go/internal/shared/logger"
	"learning-go/internal/ocr/application/dto"
	"learning-go/internal/ocr/application/mapper"
	"learning-go/internal/ocr/application/port"
)

const (
	ocrConfidenceLowThreshold = 0.70
)

type OCRCommand struct {
	engines port.OCREngineRegistry
}

func NewOCRCommand(engines port.OCREngineRegistry) port.OCRCommandPort {
	return &OCRCommand{engines: engines}
}

func (useCase *OCRCommand) ProcessScan(ctx context.Context, req dto.OCRScanRequest) (*dto.OCRScanResponse, error) {
	start := time.Now()

	ocrResult, err := useCase.recognize(ctx, req)
	if err != nil {
		return nil, err
	}

	useCase.enrichPronunciation(ocrResult, req.Language)

	totalDetected := len(ocrResult.Characters)
	items, lowConfidence := useCase.classifyByConfidence(ocrResult.Characters)

	resp := &dto.OCRScanResponse{
		Items:         items,
		LowConfidence: lowConfidence,
		Metadata: dto.OCRScanMetadata{
			EngineUsed:       ocrResult.Engine,
			TotalDetected:    totalDetected,
			ProcessingTimeMs: time.Since(start).Milliseconds(),
		},
	}

	logger.Info(ctx, "[OCR] scan completed",
		zap.Int("items", len(items)),
		zap.Int("low_confidence", len(lowConfidence)),
		zap.String("engine", ocrResult.Engine),
		zap.Int64("processing_ms", resp.Metadata.ProcessingTimeMs),
	)

	return resp, nil
}

func (useCase *OCRCommand) recognize(ctx context.Context, req dto.OCRScanRequest) (*port.OCRResult, error) {
	engine, _ := useCase.resolveEngine(req.Type, req.Language, req.Engine)
	if engine == nil {
		return nil, apperr.ServiceUnavailable("ocr.no_engine_available", nil)
	}

	ocrResult, err := engine.Recognize(ctx, port.OCRRequest{Image: req.Image, Language: req.Language})
	if err != nil {
		if _, ok := apperr.IsAppError(err); ok {
			return nil, err
		}
		return nil, apperr.ServiceUnavailable("ocr.recognize_failed", err)
	}
	return ocrResult, nil
}

func (useCase *OCRCommand) enrichPronunciation(result *port.OCRResult, language string) {
	if language != "zh" {
		return
	}
	for i := range result.Characters {
		if result.Characters[i].Pronunciation == "" {
			result.Characters[i].Pronunciation = mapper.ConvertToPinyin(result.Characters[i].Text)
		}
	}
}

func (useCase *OCRCommand) classifyByConfidence(characters []port.OCRCharacter) ([]dto.OCRCharacterItem, []dto.OCRCharacterItem) {
	items := []dto.OCRCharacterItem{}
	lowConfidence := []dto.OCRCharacterItem{}

	for _, char := range characters {
		item := dto.OCRCharacterItem{
			Text:          char.Text,
			Pronunciation: char.Pronunciation,
			Confidence:    char.Confidence,
			Candidates:    char.Candidates,
		}
		if char.Confidence < ocrConfidenceLowThreshold {
			lowConfidence = append(lowConfidence, item)
		} else {
			items = append(items, item)
		}
	}
	return items, lowConfidence
}

// --- Engine routing ---

func (useCase *OCRCommand) resolveEngine(ocrType, language, forceEngine string) (port.OCRServicePort, port.OCREngineKey) {
	if forceEngine != "" {
		return useCase.getEngine(port.OCREngineKey(forceEngine))
	}

	switch ocrType {
	case "printed":
		return useCase.getFirstAvailable(
			port.OCREngineGoogleVision,
			port.OCREnginePaddleOCR,
			port.OCREngineTesseract,
		)

	case "handwritten":
		if language == "zh" {
			return useCase.getFirstAvailable(
				port.OCREngineBaiduOCR,
				port.OCREnginePaddleOCR,
				port.OCREngineGoogleVision,
			)
		}
		return useCase.getEngine(port.OCREngineGoogleVision)

	default: // "auto"
		return useCase.getFirstAvailable(
			port.OCREngineGoogleVision,
			port.OCREnginePaddleOCR,
			port.OCREngineTesseract,
		)
	}
}

func (useCase *OCRCommand) getEngine(key port.OCREngineKey) (port.OCRServicePort, port.OCREngineKey) {
	if engine, ok := useCase.engines[key]; ok {
		return engine, key
	}
	return nil, ""
}

func (useCase *OCRCommand) getFirstAvailable(keys ...port.OCREngineKey) (port.OCRServicePort, port.OCREngineKey) {
	for _, key := range keys {
		if engine, ok := useCase.engines[key]; ok {
			return engine, key
		}
	}
	return nil, ""
}
