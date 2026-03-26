package repository

import (
	"context"
	"errors"
	"learning-go/internal/vocabulary/adapter/repository/model"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GrammarPointRepository struct {
	db *gorm.DB
}

func NewGrammarPointRepository(db *gorm.DB) port.GrammarPointRepositoryPort {
	return &GrammarPointRepository{db: db}
}

func (repo *GrammarPointRepository) FindByVocabularyID(ctx context.Context, vocabID uuid.UUID) ([]*domain.GrammarPoint, error) {
	var models []model.GrammarPointModel
	if err := repo.db.WithContext(ctx).
		Joins("JOIN vocabulary_grammar_points vgp ON vgp.grammar_point_id = grammar_points.id").
		Where("vgp.vocabulary_id = ?", vocabID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.GrammarPoint, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result, nil
}

func (repo *GrammarPointRepository) FindByHSKLevel(ctx context.Context, level int) ([]*domain.GrammarPoint, error) {
	var models []model.GrammarPointModel
	if err := repo.db.WithContext(ctx).Where("hsk_level = ?", level).Order("code ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.GrammarPoint, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result, nil
}

func (repo *GrammarPointRepository) FindByCode(ctx context.Context, code string) (*domain.GrammarPoint, error) {
	var m model.GrammarPointModel
	if err := repo.db.WithContext(ctx).First(&m, "code = ?", code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return m.ToEntity(), nil
}

func (repo *GrammarPointRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.GrammarPoint, error) {
	if len(ids) == 0 {
		return []*domain.GrammarPoint{}, nil
	}
	var models []model.GrammarPointModel
	if err := repo.db.WithContext(ctx).Where("id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.GrammarPoint, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result, nil
}
