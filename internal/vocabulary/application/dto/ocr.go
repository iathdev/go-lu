package dto

// --- Requests ---

type OCRScanHTTPRequest struct {
	Type     string `form:"type" binding:"omitempty,oneof=printed handwritten auto"`
	Language string `form:"language"`
	Engine   string `form:"engine" binding:"omitempty,oneof=paddleocr tesseract google_vision baidu_ocr"`
}

// --- Responses ---

type OCRScanCharacterItem struct {
	Text          string   `json:"text"`
	Pronunciation string   `json:"pronunciation"`
	Confidence    float64  `json:"confidence"`
	Candidates    []string `json:"candidates"`
}

type OCRScanExistingItem struct {
	VocabularyListResponse
	Confidence float64 `json:"confidence"`
}

type OCRScanMetadata struct {
	EngineUsed       string `json:"engine_used"`
	TotalDetected    int    `json:"total_detected"`
	ProcessingTimeMs int64  `json:"processing_time_ms"`
}

type OCRScanResponse struct {
	NewItems      []OCRScanCharacterItem `json:"new_items"`
	ExistingItems []OCRScanExistingItem  `json:"existing_items"`
	LowConfidence []OCRScanCharacterItem `json:"low_confidence"`
	Metadata      OCRScanMetadata        `json:"metadata"`
}
