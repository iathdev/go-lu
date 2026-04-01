package model

import (
	"learning-go/internal/vocabulary/domain"
	"time"

	"github.com/google/uuid"
)

type ProficiencyLevelModel struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key"`
	CategoryID    uuid.UUID `gorm:"type:uuid;not null"`
	PrepLevelID   *int      `gorm:"column:prep_level_id"`
	Code          string    `gorm:"not null"`
	Name          string    `gorm:"not null"`
	Target        float64   `gorm:"type:decimal(8,2)"`
	DisplayTarget string    `gorm:"column:display_target"`
	Offset        int       `gorm:"column:offset;not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (ProficiencyLevelModel) TableName() string { return "proficiency_levels" }

func (model *ProficiencyLevelModel) ToEntity() *domain.ProficiencyLevel {
	return &domain.ProficiencyLevel{
		ID:            domain.ProficiencyLevelIDFromUUID(model.ID),
		CategoryID:    domain.CategoryIDFromUUID(model.CategoryID),
		PrepLevelID:   model.PrepLevelID,
		Code:          model.Code,
		Name:          model.Name,
		Target:        model.Target,
		DisplayTarget: model.DisplayTarget,
		Offset:        model.Offset,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

func FromProficiencyLevelEntity(level *domain.ProficiencyLevel) *ProficiencyLevelModel {
	return &ProficiencyLevelModel{
		ID:            level.ID.UUID(),
		CategoryID:    level.CategoryID.UUID(),
		PrepLevelID:   level.PrepLevelID,
		Code:          level.Code,
		Name:          level.Name,
		Target:        level.Target,
		DisplayTarget: level.DisplayTarget,
		Offset:        level.Offset,
		CreatedAt:     level.CreatedAt,
		UpdatedAt:     level.UpdatedAt,
	}
}
