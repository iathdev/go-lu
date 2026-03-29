package domain

import "time"

// GrammarPoint is an entity representing a grammar pattern linked to vocabularies.
type GrammarPoint struct {
	ID            GrammarPointID
	Code          string
	Pattern       string
	ExampleCN     string
	ExampleVI     string
	Rule          string
	CommonMistake string
	HSKLevel      HSKLevel
	CreatedAt     time.Time
}

func NewGrammarPoint(code, pattern string, hskLevel int) (*GrammarPoint, error) {
	if code == "" {
		return nil, ErrGrammarPointCodeRequired
	}
	if pattern == "" {
		return nil, ErrGrammarPointPatternRequired
	}
	level, err := NewHSKLevel(hskLevel)
	if err != nil {
		return nil, err
	}

	return &GrammarPoint{
		ID:       NewGrammarPointID(),
		Code:     code,
		Pattern:  pattern,
		HSKLevel: level,
	}, nil
}
