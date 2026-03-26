package model

import (
	"learning-go/internal/vocabulary/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Name        string    `gorm:"not null"`
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (FolderModel) TableName() string { return "folders" }

func (model *FolderModel) ToEntity() *domain.Folder {
	return &domain.Folder{
		ID:          model.ID,
		UserID:      model.UserID,
		Name:        model.Name,
		Description: model.Description,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

func FromFolderEntity(folder *domain.Folder) *FolderModel {
	return &FolderModel{
		ID:          folder.ID,
		UserID:      folder.UserID,
		Name:        folder.Name,
		Description: folder.Description,
		CreatedAt:   folder.CreatedAt,
		UpdatedAt:   folder.UpdatedAt,
	}
}

type FolderVocabularyModel struct {
	FolderID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	VocabularyID uuid.UUID `gorm:"type:uuid;primaryKey"`
	AddedAt      time.Time
}

func (FolderVocabularyModel) TableName() string { return "folder_vocabularies" }
