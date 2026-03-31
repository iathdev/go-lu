package dto

import "time"

// --- Requests ---

type CreateFolderRequest struct {
	LanguageID  string `json:"language_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateFolderRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type FolderVocabularyRequest struct {
	VocabularyID string `json:"vocabulary_id" binding:"required"`
}

// --- Responses ---

type FolderResponse struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	LanguageID      string    `json:"language_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	VocabularyCount int       `json:"vocabulary_count"`
	CreatedAt       time.Time `json:"created_at"`
}
