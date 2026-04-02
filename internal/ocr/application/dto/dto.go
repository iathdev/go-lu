package dto

type OCRScanRequest struct {
	Image    []byte
	Type     string // "printed" | "handwritten" | "auto"
	Language string // "zh" | "vi" | "en"
	Engine   string // optional: force specific engine
}

type BoundingBoxDTO struct {
	Vertices []PointDTO `json:"vertices"`
}

type PointDTO struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type OCRCharacterItem struct {
	Text          string          `json:"text"`
	Pronunciation string          `json:"pronunciation"`
	Confidence    float64         `json:"confidence"`
	Candidates    []string        `json:"candidates"`
	BoundingBox   *BoundingBoxDTO `json:"bounding_box,omitempty"`
}

// --- Extract Text ---

type OCRExtractTextRequest struct {
	Image    []byte
	Type     string // "printed" | "handwritten" | "auto"
	Language string // "zh" | "vi" | "en"
	Engine   string // optional: force specific engine
}

type OCRTextBlockItem struct {
	Text        string          `json:"text"`
	BoundingBox *BoundingBoxDTO `json:"bounding_box,omitempty"`
	Confidence  float64         `json:"confidence"`
}

type OCRExtractTextMetadata struct {
	EngineUsed       string `json:"engine_used"`
	TotalBlocks      int    `json:"total_blocks"`
	ProcessingTimeMs int64  `json:"processing_time_ms"`
}

type OCRExtractTextResponse struct {
	Blocks   []OCRTextBlockItem     `json:"blocks"`
	FullText string                 `json:"full_text"`
	Metadata OCRExtractTextMetadata `json:"metadata"`
}

// --- Scan ---

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
