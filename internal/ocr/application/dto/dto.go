package dto

type OCRScanHTTPRequest struct {
	ImageURL string `json:"image_url" binding:"required,url"`
	Type     string `json:"type" binding:"omitempty,oneof=printed handwritten auto"`
	Language string `json:"language" binding:"omitempty,oneof=zh vi en"`
	Engine   string `json:"engine" binding:"omitempty,oneof=paddleocr tesseract google_vision baidu_ocr"`
}

type OCRScanRequest struct {
	Image    []byte
	Type     string // "printed" | "handwritten" | "auto"
	Language string // "zh" | "vi" | "en"
	Engine   string // optional: force specific engine
}

type OCRCharacterItem struct {
	Text          string   `json:"text"`
	Pronunciation string   `json:"pronunciation,omitempty"`
	Confidence    float64  `json:"confidence"`
	Candidates    []string `json:"candidates,omitempty"`
}

type OCRScanMetadata struct {
	EngineUsed       string `json:"engine_used"`
	TotalDetected    int    `json:"total_detected"`
	ProcessingTimeMs int64  `json:"processing_time_ms"`
}

type OCRScanResponse struct {
	Items            []OCRCharacterItem `json:"items"`
	LowConfidence    []OCRCharacterItem `json:"low_confidence"`
	Metadata         OCRScanMetadata    `json:"metadata"`
}
