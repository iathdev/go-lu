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

func (model *GrammarPointModel) ToEntity() *domain.GrammarPoint {
	return &domain.GrammarPoint{
		ID:            model.ID,
		Code:          model.Code,
		Pattern:       model.Pattern,
		ExampleCN:     model.ExampleCN,
		ExampleVI:     model.ExampleVI,
		Rule:          model.Rule,
		CommonMistake: model.CommonMistake,
		HSKLevel:      model.HSKLevel,
		CreatedAt:     model.CreatedAt,
	}
}
