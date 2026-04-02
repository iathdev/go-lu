package domain

import "time"

// GrammarPoint is an entity representing a grammar pattern, scoped to a category.
type GrammarPoint struct {
	ID                 GrammarPointID
	CategoryID         CategoryID
	LevelID LevelID
	Code               string
	Pattern            string
	Examples           map[string]any
	Rule               map[string]any
	CommonMistakes     map[string]any
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewGrammarPoint(categoryID CategoryID, code, pattern string) (*GrammarPoint, error) {
	if code == "" {
		return nil, ErrGrammarPointCodeRequired
	}
	if pattern == "" {
		return nil, ErrGrammarPointPatternRequired
	}

	return &GrammarPoint{
		ID:         NewGrammarPointID(),
		CategoryID: categoryID,
		Code:       code,
		Pattern:    pattern,
	}, nil
}
