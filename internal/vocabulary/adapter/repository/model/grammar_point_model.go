package model

import (
	"learning-go/internal/shared/common"
	"learning-go/internal/vocabulary/domain"
	"time"

	"github.com/google/uuid"
)

type GenericJSON = common.JSONB[map[string]any]

type GrammarPointModel struct {
	ID                 uuid.UUID   `gorm:"type:uuid;primary_key"`
	CategoryID         uuid.UUID   `gorm:"type:uuid;not null"`
	LevelID uuid.UUID   `gorm:"column:level_id;type:uuid"`
	Code               string      `gorm:"not null"`
	Pattern            string      `gorm:"not null"`
	Examples           GenericJSON `gorm:"type:jsonb;default:'{}'"`
	Rule               GenericJSON `gorm:"type:jsonb;default:'{}'"`
	CommonMistakes     GenericJSON `gorm:"column:common_mistakes;type:jsonb;default:'{}'"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (GrammarPointModel) TableName() string { return "grammar_points" }

func (model *GrammarPointModel) ToEntity() *domain.GrammarPoint {
	examples := make(map[string]any)
	for key, val := range model.Examples.Data {
		examples[key] = val
	}

	rule := make(map[string]any)
	for key, val := range model.Rule.Data {
		rule[key] = val
	}

	commonMistakes := make(map[string]any)
	for key, val := range model.CommonMistakes.Data {
		commonMistakes[key] = val
	}

	return &domain.GrammarPoint{
		ID:                 domain.GrammarPointIDFromUUID(model.ID),
		CategoryID:         domain.CategoryIDFromUUID(model.CategoryID),
		LevelID: domain.LevelIDFromUUID(model.LevelID),
		Code:               model.Code,
		Pattern:            model.Pattern,
		Examples:           examples,
		Rule:               rule,
		CommonMistakes:     commonMistakes,
		CreatedAt:          model.CreatedAt,
		UpdatedAt:          model.UpdatedAt,
	}
}

func FromGrammarPointEntity(gp *domain.GrammarPoint) *GrammarPointModel {
	return &GrammarPointModel{
		ID:                 gp.ID.UUID(),
		CategoryID:         gp.CategoryID.UUID(),
		LevelID: gp.LevelID.UUID(),
		Code:               gp.Code,
		Pattern:            gp.Pattern,
		Examples:           common.NewJSONB(gp.Examples),
		Rule:               common.NewJSONB(gp.Rule),
		CommonMistakes:     common.NewJSONB(gp.CommonMistakes),
		CreatedAt:          gp.CreatedAt,
		UpdatedAt:          gp.UpdatedAt,
	}
}
