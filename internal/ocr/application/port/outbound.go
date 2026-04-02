package port

import "context"

type OCRServicePort interface {
	Recognize(ctx context.Context, req OCRRequest) (*OCRResult, error)
	ExtractText(ctx context.Context, req OCRRequest) (*OCRTextResult, error)
}

type OCRTextBlock struct {
	Text        string
	BoundingBox *BoundingBox
	Confidence  float64
}

type OCRTextResult struct {
	Blocks []OCRTextBlock
	Engine string
}

type OCRRequest struct {
	Image    []byte
	Language string // "zh" | "vi" | "en"
}

type OCRResult struct {
	Characters []OCRCharacter
	Engine     string // "paddleocr" | "tesseract" | "google_vision" | "baidu_ocr"
}

type BoundingBox struct {
	Vertices []Point
}

type Point struct {
	X int
	Y int
}

type OCRCharacter struct {
	Text          string
	Pronunciation string
	Confidence    float64
	Candidates    []string
	BoundingBox   *BoundingBox
}

type OCREngineKey string

const (
	OCREnginePaddleOCR    OCREngineKey = "paddleocr"
	OCREngineGoogleVision OCREngineKey = "google_vision"
	OCREngineBaiduOCR     OCREngineKey = "baidu_ocr"
	OCREngineTesseract    OCREngineKey = "tesseract"
)

type OCREngineRegistry map[OCREngineKey]OCRServicePort
