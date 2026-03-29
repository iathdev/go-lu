package repository

import (
	"context"
	"errors"
	"learning-go/internal/vocabulary/adapter/repository/model"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"

	"gorm.io/gorm"
)

type TopicRepository struct {
	db *gorm.DB
}

func NewTopicRepository(db *gorm.DB) port.TopicRepositoryPort {
	return &TopicRepository{db: db}
}

func (repo *TopicRepository) FindAll(ctx context.Context) ([]*domain.Topic, error) {
	var models []model.TopicModel
	if err := repo.db.WithContext(ctx).Order("sort_order ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.Topic, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result, nil
}

func (repo *TopicRepository) FindBySlug(ctx context.Context, slug string) (*domain.Topic, error) {
	var m model.TopicModel
	if err := repo.db.WithContext(ctx).First(&m, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return m.ToEntity(), nil
}

func (repo *TopicRepository) FindByIDs(ctx context.Context, ids []domain.TopicID) ([]*domain.Topic, error) {
	if len(ids) == 0 {
		return []*domain.Topic{}, nil
	}
	uuids := make([]any, 0, len(ids))
	for _, id := range ids {
		uuids = append(uuids, id.UUID())
	}
	var models []model.TopicModel
	if err := repo.db.WithContext(ctx).Where("id IN ?", uuids).Order("sort_order ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.Topic, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result, nil
}

func (repo *TopicRepository) FindByVocabularyID(ctx context.Context, vocabID domain.VocabularyID) ([]*domain.Topic, error) {
	var models []model.TopicModel
	if err := repo.db.WithContext(ctx).
		Joins("JOIN vocabulary_topics vt ON vt.topic_id = topics.id").
		Where("vt.vocabulary_id = ?", vocabID.UUID()).
		Order("sort_order ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.Topic, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result, nil
}
