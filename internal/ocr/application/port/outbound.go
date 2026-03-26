package port

import "context"

type OCRServicePort interface {
	Recognize(ctx context.Context, req OCRRequest) (*OCRResult, error)
}

type OCRRequest struct {
	Image    []byte
	Language string // "zh" | "vi" | "en"
}

type OCRResult struct {
	Characters []OCRCharacter
	Engine     string // "paddleocr" | "tesseract" | "google_vision" | "baidu_ocr"
}

type OCRCharacter struct {
	Text          string
	Pronunciation string
	Confidence    float64
	Candidates    []string
}

type OCREngineKey string

const (
	OCREnginePaddleOCR    OCREngineKey = "paddleocr"
	OCREngineGoogleVision OCREngineKey = "google_vision"
	OCREngineBaiduOCR     OCREngineKey = "baidu_ocr"
	OCREngineTesseract    OCREngineKey = "tesseract"
)

type OCREngineRegistry map[OCREngineKey]OCRServicePort
