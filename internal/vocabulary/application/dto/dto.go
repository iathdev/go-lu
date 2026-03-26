package dto

import "time"

// --- Vocabulary DTOs ---

type ExampleDTO struct {
	SentenceCN string `json:"sentence_cn"`
	SentenceVI string `json:"sentence_vi"`
	AudioURL   string `json:"audio_url,omitempty"`
}

type CreateVocabularyRequest struct {
	Hanzi           string       `json:"hanzi" binding:"required"`
	Pinyin          string       `json:"pinyin" binding:"required"`
	MeaningVI       string       `json:"meaning_vi"`
	MeaningEN       string       `json:"meaning_en"`
	HSKLevel        int          `json:"hsk_level" binding:"required,min=1,max=9"`
	AudioURL        string       `json:"audio_url"`
	Examples        []ExampleDTO `json:"examples"`
	Radicals        []string     `json:"radicals"`
	StrokeCount     int          `json:"stroke_count"`
	StrokeDataURL   string       `json:"stroke_data_url"`
	RecognitionOnly bool         `json:"recognition_only"`
	FrequencyRank   int          `json:"frequency_rank"`
}

type UpdateVocabularyRequest struct {
	Hanzi           string       `json:"hanzi" binding:"required"`
	Pinyin          string       `json:"pinyin" binding:"required"`
	MeaningVI       string       `json:"meaning_vi"`
	MeaningEN       string       `json:"meaning_en"`
	HSKLevel        int          `json:"hsk_level" binding:"required,min=1,max=9"`
	AudioURL        string       `json:"audio_url"`
	Examples        []ExampleDTO `json:"examples"`
	Radicals        []string     `json:"radicals"`
	StrokeCount     int          `json:"stroke_count"`
	StrokeDataURL   string       `json:"stroke_data_url"`
	RecognitionOnly bool         `json:"recognition_only"`
	FrequencyRank   int          `json:"frequency_rank"`
	TopicIDs        []string     `json:"topic_ids"`
	GrammarPointIDs []string     `json:"grammar_point_ids"`
}

type VocabularyResponse struct {
	ID              string       `json:"id"`
	Hanzi           string       `json:"hanzi"`
	Pinyin          string       `json:"pinyin"`
	MeaningVI       string       `json:"meaning_vi"`
	MeaningEN       string       `json:"meaning_en"`
	HSKLevel        int          `json:"hsk_level"`
	AudioURL        string       `json:"audio_url,omitempty"`
	Examples        []ExampleDTO `json:"examples,omitempty"`
	Radicals        []string     `json:"radicals,omitempty"`
	StrokeCount     int          `json:"stroke_count,omitempty"`
	StrokeDataURL   string       `json:"stroke_data_url,omitempty"`
	RecognitionOnly bool         `json:"recognition_only"`
	FrequencyRank   int          `json:"frequency_rank,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
}

type VocabularyDetailResponse struct {
	VocabularyResponse
	Topics        []TopicResponse        `json:"topics"`
	GrammarPoints []GrammarPointResponse `json:"grammar_points"`
}

type VocabularyListResponse struct {
	ID        string `json:"id"`
	Hanzi     string `json:"hanzi"`
	Pinyin    string `json:"pinyin"`
	MeaningVI string `json:"meaning_vi"`
	MeaningEN string `json:"meaning_en"`
	HSKLevel  int    `json:"hsk_level"`
}

// --- Topic DTOs ---

type TopicResponse struct {
	ID     string `json:"id"`
	NameCN string `json:"name_cn"`
	NameVI string `json:"name_vi"`
	NameEN string `json:"name_en"`
	Slug   string `json:"slug"`
}

// --- Grammar Point DTOs ---

type GrammarPointResponse struct {
	ID            string `json:"id"`
	Code          string `json:"code"`
	Pattern       string `json:"pattern"`
	ExampleCN     string `json:"example_cn,omitempty"`
	ExampleVI     string `json:"example_vi,omitempty"`
	Rule          string `json:"rule,omitempty"`
	CommonMistake string `json:"common_mistake,omitempty"`
	HSKLevel      int    `json:"hsk_level"`
}

// --- OCR DTOs ---

type OCRScanHTTPRequest struct {
	ImageURL string `json:"image_url" binding:"required,url"`
	Type     string `json:"type" binding:"omitempty,oneof=printed handwritten auto"`
	Language string `json:"language" binding:"omitempty,oneof=zh vi en"`
	Engine   string `json:"engine" binding:"omitempty,oneof=paddleocr tesseract google_vision baidu_ocr"`
}

type OCRScanCharacterItem struct {
	Text          string   `json:"text"`
	Pronunciation string   `json:"pronunciation,omitempty"`
	Confidence    float64  `json:"confidence"`
	Candidates    []string `json:"candidates,omitempty"`
}

type OCRScanExistingItem struct {
	VocabularyListResponse
	Confidence float64  `json:"confidence"`
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

// --- Bulk Import DTOs ---

type BulkImportRequest struct {
	Vocabularies []CreateVocabularyRequest `json:"vocabularies" binding:"required,min=1"`
}

type BulkImportResponse struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Total    int `json:"total"`
}

// --- Folder DTOs ---

type CreateFolderRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateFolderRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type FolderResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type FolderVocabularyRequest struct {
	VocabularyID string `json:"vocabulary_id" binding:"required"`
}
