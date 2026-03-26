package model

import "github.com/google/uuid"

type VocabularyTopicModel struct {
	VocabularyID uuid.UUID `gorm:"type:uuid;primaryKey"`
	TopicID      uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (VocabularyTopicModel) TableName() string { return "vocabulary_topics" }

type VocabularyGrammarPointModel struct {
	VocabularyID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	GrammarPointID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (VocabularyGrammarPointModel) TableName() string { return "vocabulary_grammar_points" }
