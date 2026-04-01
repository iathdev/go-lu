package domain

import "time"

// Category groups proficiency systems per language (e.g. HSK, JLPT, CEFR).
type Category struct {
	ID             CategoryID
	LanguageID     LanguageID
	PrepCategoryID *int
	Code           string
	Name           string
	IsPublic       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
