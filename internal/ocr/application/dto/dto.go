package dto

type OCRScanRequest struct {
	Image    []byte
	Type     string // "printed" | "handwritten" | "auto"
	Language string // "zh" | "vi" | "en"
	Engine   string // optional: force specific engine
}

type OCRCharacterItem struct {
	Text          string   `json:"text"`
	Pronunciation string   `json:"pronunciation"`
	Confidence    float64  `json:"confidence"`
	Candidates    []string `json:"candidates"`
}

type OCRScanMetadata struct {
	EngineUsed       string `json:"engine_used"`
	TotalDetected    int    `json:"total_detected"`
	ProcessingTimeMs int64  `json:"processing_time_ms"`
}

type OCRScanResponse struct {
	Items         []OCRCharacterItem `json:"items"`
	LowConfidence []OCRCharacterItem `json:"low_confidence"`
	Metadata      OCRScanMetadata    `json:"metadata"`
}
