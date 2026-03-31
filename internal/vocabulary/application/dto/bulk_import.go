package dto

type BulkImportRequest struct {
	Vocabularies []CreateVocabularyRequest `json:"vocabularies" binding:"required,min=1"`
}

type BulkImportResponse struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Total    int `json:"total"`
}
