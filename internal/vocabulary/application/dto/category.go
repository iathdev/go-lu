package dto

type CategoryResponse struct {
	ID             string `json:"id"`
	LanguageID     string `json:"language_id"`
	PrepCategoryID *int   `json:"prep_category_id,omitempty"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	IsPublic       bool   `json:"is_public"`
}

type ProficiencyLevelResponse struct {
	ID            string  `json:"id"`
	CategoryID    string  `json:"category_id"`
	PrepLevelID   *int    `json:"prep_level_id,omitempty"`
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Target        float64 `json:"target"`
	DisplayTarget string  `json:"display_target"`
	Offset        int     `json:"offset"`
}
