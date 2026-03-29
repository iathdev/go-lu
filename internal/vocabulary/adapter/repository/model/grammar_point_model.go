package model

import (
	"learning-go/internal/vocabulary/domain"
	"time"

	"github.com/google/uuid"
)

type GrammarPointModel struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;"`
	Code          string    `gorm:"not null;uniqueIndex"`
	Pattern       string    `gorm:"not null"`
	ExampleCN     string    `gorm:"column:example_cn"`
	ExampleVI     string    `gorm:"column:example_vi"`
	Rule          string
	CommonMistake string    `gorm:"column:common_mistake"`
	HSKLevel      int       `gorm:"not null;index"`
	CreatedAt     time.Time
}

func (GrammarPointModel) TableName() string { return "grammar_points" }

func (m *GrammarPointModel) ToEntity() *domain.GrammarPoint {
	return &domain.GrammarPoint{
		ID:            domain.GrammarPointIDFromUUID(m.ID),
		Code:          m.Code,
		Pattern:       m.Pattern,
		ExampleCN:     m.ExampleCN,
		ExampleVI:     m.ExampleVI,
		Rule:          m.Rule,
		CommonMistake: m.CommonMistake,
		HSKLevel:      domain.HSKLevel(m.HSKLevel),
		CreatedAt:     m.CreatedAt,
	}
}
