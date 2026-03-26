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

type VocabularyRepository struct {
	db *gorm.DB
}

func NewVocabularyRepository(db *gorm.DB) port.VocabularyRepositoryPort {
	return &VocabularyRepository{db: db}
}

func (repo *VocabularyRepository) Save(ctx context.Context, vocab *domain.Vocabulary) error {
	m := model.FromVocabularyEntity(vocab)
	if err := repo.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	vocab.CreatedAt = m.CreatedAt
	vocab.UpdatedAt = m.UpdatedAt
	return nil
}

func (repo *VocabularyRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Vocabulary, error) {
	var m model.VocabularyModel
	if err := repo.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return m.ToEntity(), nil
}

func (repo *VocabularyRepository) FindByHanzi(ctx context.Context, hanzi string) (*domain.Vocabulary, error) {
	var m model.VocabularyModel
	if err := repo.db.WithContext(ctx).Where("hanzi = ?", hanzi).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return m.ToEntity(), nil
}

func (repo *VocabularyRepository) FindByHanziList(ctx context.Context, hanziList []string) ([]*domain.Vocabulary, error) {
	if len(hanziList) == 0 {
		return []*domain.Vocabulary{}, nil
	}
	var models []model.VocabularyModel
	if err := repo.db.WithContext(ctx).Where("hanzi IN ?", hanziList).Find(&models).Error; err != nil {
		return nil, err
	}
	return toVocabEntities(models), nil
}

func (repo *VocabularyRepository) FindByHSKLevel(ctx context.Context, level int, offset, limit int) ([]*domain.Vocabulary, error) {
	var models []model.VocabularyModel
	if err := repo.db.WithContext(ctx).Where("hsk_level = ?", level).Offset(offset).Limit(limit).Order("pinyin ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	return toVocabEntities(models), nil
}

func (repo *VocabularyRepository) CountByHSKLevel(ctx context.Context, level int) (int64, error) {
	var count int64
	if err := repo.db.WithContext(ctx).Model(&model.VocabularyModel{}).Where("hsk_level = ?", level).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (repo *VocabularyRepository) FindByTopicID(ctx context.Context, topicID uuid.UUID, offset, limit int) ([]*domain.Vocabulary, error) {
	var models []model.VocabularyModel
	if err := repo.db.WithContext(ctx).
		Joins("JOIN vocabulary_topics vt ON vt.vocabulary_id = vocabularies.id").
		Where("vt.topic_id = ?", topicID).
		Offset(offset).Limit(limit).Order("pinyin ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	return toVocabEntities(models), nil
}

func (repo *VocabularyRepository) CountByTopicID(ctx context.Context, topicID uuid.UUID) (int64, error) {
	var count int64
	if err := repo.db.WithContext(ctx).Model(&model.VocabularyTopicModel{}).
		Where("topic_id = ?", topicID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (repo *VocabularyRepository) Search(ctx context.Context, query string, offset, limit int) ([]*domain.Vocabulary, error) {
	var models []model.VocabularyModel
	q := "%" + query + "%"
	if err := repo.db.WithContext(ctx).
		Where("hanzi LIKE ? OR pinyin LIKE ? OR meaning_vi LIKE ? OR meaning_en LIKE ?", q, q, q, q).
		Offset(offset).Limit(limit).Order("hsk_level ASC, pinyin ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	return toVocabEntities(models), nil
}

func (repo *VocabularyRepository) CountSearch(ctx context.Context, query string) (int64, error) {
	var count int64
	q := "%" + query + "%"
	if err := repo.db.WithContext(ctx).Model(&model.VocabularyModel{}).
		Where("hanzi LIKE ? OR pinyin LIKE ? OR meaning_vi LIKE ? OR meaning_en LIKE ?", q, q, q, q).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (repo *VocabularyRepository) Update(ctx context.Context, vocab *domain.Vocabulary) error {
	m := model.FromVocabularyEntity(vocab)
	if err := repo.db.WithContext(ctx).Save(m).Error; err != nil {
		return err
	}
	vocab.UpdatedAt = m.UpdatedAt
	return nil
}

func (repo *VocabularyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return repo.db.WithContext(ctx).Delete(&model.VocabularyModel{}, "id = ?", id).Error
}

func (repo *VocabularyRepository) SaveBatch(ctx context.Context, vocabs []*domain.Vocabulary) (int, error) {
	if len(vocabs) == 0 {
		return 0, nil
	}

	models := make([]model.VocabularyModel, 0, len(vocabs))
	for _, vocab := range vocabs {
		models = append(models, *model.FromVocabularyEntity(vocab))
	}

	result := repo.db.WithContext(ctx).CreateInBatches(models, 100)
	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}

func (repo *VocabularyRepository) SetTopics(ctx context.Context, vocabID uuid.UUID, topicIDs []uuid.UUID) error {
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("vocabulary_id = ?", vocabID).Delete(&model.VocabularyTopicModel{}).Error; err != nil {
			return err
		}
		for _, tid := range topicIDs {
			vt := model.VocabularyTopicModel{VocabularyID: vocabID, TopicID: tid}
			if err := tx.Create(&vt).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (repo *VocabularyRepository) SetGrammarPoints(ctx context.Context, vocabID uuid.UUID, grammarPointIDs []uuid.UUID) error {
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("vocabulary_id = ?", vocabID).Delete(&model.VocabularyGrammarPointModel{}).Error; err != nil {
			return err
		}
		for _, gpid := range grammarPointIDs {
			vgp := model.VocabularyGrammarPointModel{VocabularyID: vocabID, GrammarPointID: gpid}
			if err := tx.Create(&vgp).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func toVocabEntities(models []model.VocabularyModel) []*domain.Vocabulary {
	result := make([]*domain.Vocabulary, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result
}
