package domain

import "errors"

// Vocabulary entity errors
var (
	ErrHanziRequired   = errors.New("hanzi is required")
	ErrPinyinRequired  = errors.New("pinyin is required")
	ErrInvalidHSKLevel = errors.New("hsk level must be between 1 and 9")
	ErrMeaningRequired = errors.New("at least one meaning (vi or en) is required")
)

// ErrFolderNameRequired Folder entity errors
var (
	ErrFolderNameRequired = errors.New("folder name is required")
)

// ErrTopicSlugRequired Topic entity errors
var (
	ErrTopicSlugRequired = errors.New("topic slug is required")
)

// GrammarPoint entity errors
var (
	ErrGrammarPointCodeRequired    = errors.New("grammar point code is required")
	ErrGrammarPointPatternRequired = errors.New("grammar point pattern is required")
)
