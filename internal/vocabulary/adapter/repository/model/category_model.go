package model

import (
	"learning-go/internal/vocabulary/domain"
	"time"

	"github.com/google/uuid"
)

type CategoryModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key"`
	LanguageID     uuid.UUID `gorm:"type:uuid;not null"`
	PrepCategoryID *int      `gorm:"column:prep_category_id"`
	Code           string    `gorm:"not null"`
	Name           string    `gorm:"not null"`
	IsPublic       bool      `gorm:"default:false"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (CategoryModel) TableName() string { return "categories" }

func (model *CategoryModel) ToEntity() *domain.Category {
	return &domain.Category{
		ID:             domain.CategoryIDFromUUID(model.ID),
		LanguageID:     domain.LanguageIDFromUUID(model.LanguageID),
		PrepCategoryID: model.PrepCategoryID,
		Code:           model.Code,
		Name:           model.Name,
		IsPublic:       model.IsPublic,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}
}

func FromCategoryEntity(cat *domain.Category) *CategoryModel {
	return &CategoryModel{
		ID:             cat.ID.UUID(),
		LanguageID:     cat.LanguageID.UUID(),
		PrepCategoryID: cat.PrepCategoryID,
		Code:           cat.Code,
		Name:           cat.Name,
		IsPublic:       cat.IsPublic,
		CreatedAt:      cat.CreatedAt,
		UpdatedAt:      cat.UpdatedAt,
	}
}
