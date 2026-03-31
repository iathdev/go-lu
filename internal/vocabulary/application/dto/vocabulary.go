package dto

import "time"

// --- Shared DTOs ---

type MeaningExampleDTO struct {
	Sentence     string            `json:"sentence" binding:"required"`
	Phonetic     string            `json:"phonetic"`
	Translations map[string]string `json:"translations"`
	AudioURL     string            `json:"audio_url"`
}

type MeaningDTO struct {
	LanguageID string              `json:"language_id" binding:"required"`
	Meaning    string              `json:"meaning" binding:"required"`
	WordType   string              `json:"word_type"`
	IsPrimary  bool                `json:"is_primary"`
	Examples   []MeaningExampleDTO `json:"examples"`
}

// --- Requests ---

type CreateVocabularyRequest struct {
	LanguageID         string         `json:"language_id" binding:"required"`
	ProficiencyLevelID string         `json:"proficiency_level_id"`
	Word               string         `json:"word" binding:"required"`
	Phonetic           string         `json:"phonetic"`
	AudioURL           string         `json:"audio_url"`
	ImageURL           string         `json:"image_url"`
	FrequencyRank      int            `json:"frequency_rank"`
	Metadata           map[string]any `json:"metadata"`
	Meanings           []MeaningDTO   `json:"meanings" binding:"required,min=1"`
}

type UpdateVocabularyRequest struct {
	LanguageID         string         `json:"language_id" binding:"required"`
	ProficiencyLevelID string         `json:"proficiency_level_id"`
	Word               string         `json:"word" binding:"required"`
	Phonetic           string         `json:"phonetic"`
	AudioURL           string         `json:"audio_url"`
	ImageURL           string         `json:"image_url"`
	FrequencyRank      int            `json:"frequency_rank"`
	Metadata           map[string]any `json:"metadata"`
	Meanings           []MeaningDTO   `json:"meanings" binding:"required,min=1"`
	TopicIDs           []string       `json:"topic_ids"`
	GrammarPointIDs    []string       `json:"grammar_point_ids"`
}

// VocabularyFilter carries query params for listing vocabularies.
type VocabularyFilter struct {
	LanguageID         string `form:"language_id"`
	ProficiencyLevelID string `form:"proficiency_level_id"`
	TopicID            string `form:"topic_id"`
	MeaningLang        string `form:"meaning_lang"`
}

// --- Responses ---

type MeaningExampleResponse struct {
	ID           string            `json:"id"`
	Sentence     string            `json:"sentence"`
	Phonetic     string            `json:"phonetic"`
	Translations map[string]string `json:"translations"`
	AudioURL     string            `json:"audio_url"`
}

type MeaningResponse struct {
	ID         string                   `json:"id"`
	LanguageID string                   `json:"language_id"`
	Meaning    string                   `json:"meaning"`
	WordType   string                   `json:"word_type"`
	IsPrimary  bool                     `json:"is_primary"`
	Offset     int                      `json:"offset"`
	Examples   []MeaningExampleResponse `json:"examples"`
}

type VocabularyResponse struct {
	ID                 string            `json:"id"`
	LanguageID         string            `json:"language_id"`
	ProficiencyLevelID string            `json:"proficiency_level_id"`
	Word               string            `json:"word"`
	Phonetic           string            `json:"phonetic"`
	AudioURL           string            `json:"audio_url"`
	ImageURL           string            `json:"image_url"`
	FrequencyRank      int               `json:"frequency_rank"`
	Metadata           map[string]any    `json:"metadata"`
	Meanings           []MeaningResponse `json:"meanings"`
	CreatedAt          time.Time         `json:"created_at"`
}

type VocabularyDetailResponse struct {
	VocabularyResponse
	Topics        []TopicResponse        `json:"topics"`
	GrammarPoints []GrammarPointResponse `json:"grammar_points"`
}

// VocabularyListResponse is a lightweight version for list endpoints.
type VocabularyListResponse struct {
	ID                 string                `json:"id"`
	Word               string                `json:"word"`
	Phonetic           string                `json:"phonetic"`
	Meanings           []MeaningListResponse `json:"meanings"`
	ProficiencyLevelID string                `json:"proficiency_level_id"`
	FrequencyRank      int                   `json:"frequency_rank"`
}

// MeaningListResponse is lightweight — no examples.
type MeaningListResponse struct {
	Meaning   string `json:"meaning"`
	WordType  string `json:"word_type"`
	IsPrimary bool   `json:"is_primary"`
}
