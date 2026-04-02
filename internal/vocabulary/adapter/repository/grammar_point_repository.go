package repository

import (
	"context"
	"errors"
	"learning-go/internal/vocabulary/adapter/repository/model"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"

	"gorm.io/gorm"
)

type GrammarPointRepository struct {
	db *gorm.DB
}

func NewGrammarPointRepository(db *gorm.DB) port.GrammarPointRepositoryPort {
	return &GrammarPointRepository{db: db}
}

func (repo *GrammarPointRepository) FindAll(ctx context.Context, categoryID *domain.CategoryID, profLevelID *domain.LevelID, offset, limit int) ([]*domain.GrammarPoint, error) {
	query := repo.db.WithContext(ctx)
	if categoryID != nil {
		query = query.Where("category_id = ?", categoryID.UUID())
	}
	if profLevelID != nil {
		query = query.Where("level_id = ?", profLevelID.UUID())
	}

	var models []model.GrammarPointModel
	if err := query.Offset(offset).Limit(limit).Order("code ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.GrammarPoint, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result, nil
}

func (repo *GrammarPointRepository) CountAll(ctx context.Context, categoryID *domain.CategoryID, profLevelID *domain.LevelID) (int64, error) {
	query := repo.db.WithContext(ctx).Model(&model.GrammarPointModel{})
	if categoryID != nil {
		query = query.Where("category_id = ?", categoryID.UUID())
	}
	if profLevelID != nil {
		query = query.Where("level_id = ?", profLevelID.UUID())
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (repo *GrammarPointRepository) FindByID(ctx context.Context, id domain.GrammarPointID) (*domain.GrammarPoint, error) {
	var m model.GrammarPointModel
	if err := repo.db.WithContext(ctx).First(&m, "id = ?", id.UUID()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return m.ToEntity(), nil
}

func (repo *GrammarPointRepository) FindByIDs(ctx context.Context, ids []domain.GrammarPointID) ([]*domain.GrammarPoint, error) {
	if len(ids) == 0 {
		return []*domain.GrammarPoint{}, nil
	}

	uuids := make([]any, 0, len(ids))
	for _, id := range ids {
		uuids = append(uuids, id.UUID())
	}

	var models []model.GrammarPointModel
	if err := repo.db.WithContext(ctx).Where("id IN ?", uuids).Order("code ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.GrammarPoint, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result, nil
}

func (repo *GrammarPointRepository) FindByVocabularyID(ctx context.Context, vocabID domain.VocabularyID) ([]*domain.GrammarPoint, error) {
	var models []model.GrammarPointModel
	if err := repo.db.WithContext(ctx).
		Joins("JOIN vocabulary_grammar_points vgp ON vgp.grammar_point_id = grammar_points.id").
		Where("vgp.vocabulary_id = ?", vocabID.UUID()).
		Order("code ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.GrammarPoint, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result, nil
}
