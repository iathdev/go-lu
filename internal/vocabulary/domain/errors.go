package domain

import "errors"

// Vocabulary entity errors
var (
	ErrWordRequired              = errors.New("word is required")
	ErrMeaningRequired           = errors.New("at least one meaning is required")
	ErrInvalidLanguageID         = errors.New("invalid language ID")
	ErrInvalidLevelID = errors.New("invalid level ID")
)

// Folder entity errors
var (
	ErrFolderNameRequired = errors.New("folder name is required")
)

// Topic entity errors
var (
	ErrTopicSlugRequired = errors.New("topic slug is required")
)

// GrammarPoint entity errors
var (
	ErrGrammarPointCodeRequired    = errors.New("grammar point code is required")
	ErrGrammarPointPatternRequired = errors.New("grammar point pattern is required")
)
