package repository

import (
	"context"
	"errors"
	"learning-go/internal/vocabulary/adapter/repository/model"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"

	"gorm.io/gorm"
)

type ProficiencyLevelRepository struct {
	db *gorm.DB
}

func NewProficiencyLevelRepository(db *gorm.DB) port.ProficiencyLevelRepositoryPort {
	return &ProficiencyLevelRepository{db: db}
}

func (repo *ProficiencyLevelRepository) FindAll(ctx context.Context, categoryID *domain.CategoryID) ([]*domain.ProficiencyLevel, error) {
	query := repo.db.WithContext(ctx)
	if categoryID != nil {
		query = query.Where("category_id = ?", categoryID.UUID())
	}

	var models []model.ProficiencyLevelModel
	if err := query.Order(`"offset" ASC`).Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.ProficiencyLevel, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result, nil
}

func (repo *ProficiencyLevelRepository) FindByID(ctx context.Context, id domain.LevelID) (*domain.ProficiencyLevel, error) {
	var m model.ProficiencyLevelModel
	if err := repo.db.WithContext(ctx).First(&m, "id = ?", id.UUID()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return m.ToEntity(), nil
}
