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

// Save creates a vocabulary with its meanings and examples in a single transaction.
func (repo *VocabularyRepository) Save(ctx context.Context, vocab *domain.Vocabulary) error {
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		vocabModel := model.FromVocabularyEntity(vocab)
		if err := tx.Create(vocabModel).Error; err != nil {
			return err
		}
		vocab.CreatedAt = vocabModel.CreatedAt
		vocab.UpdatedAt = vocabModel.UpdatedAt

		for idx := range vocab.Meanings {
			meaning := &vocab.Meanings[idx]
			meaningModel := model.FromMeaningEntity(meaning)
			if err := tx.Create(meaningModel).Error; err != nil {
				return err
			}
			meaning.CreatedAt = meaningModel.CreatedAt
			meaning.UpdatedAt = meaningModel.UpdatedAt

			for exIdx := range meaning.Examples {
				example := &meaning.Examples[exIdx]
				exampleModel := model.FromExampleEntity(example)
				if err := tx.Create(exampleModel).Error; err != nil {
					return err
				}
				example.CreatedAt = exampleModel.CreatedAt
				example.UpdatedAt = exampleModel.UpdatedAt
			}
		}

		return nil
	})
}

// FindByID finds a vocabulary by ID with all meanings and examples loaded.
func (repo *VocabularyRepository) FindByID(ctx context.Context, id domain.VocabularyID) (*domain.Vocabulary, error) {
	var vocabModel model.VocabularyModel
	if err := repo.db.WithContext(ctx).First(&vocabModel, "id = ?", id.UUID()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	vocab := vocabModel.ToEntity()
	if err := repo.loadMeaningsWithExamples(ctx, []*domain.Vocabulary{vocab}); err != nil {
		return nil, err
	}
	return vocab, nil
}

// FindByWord finds a vocabulary by language and exact word match (non-deleted only).
func (repo *VocabularyRepository) FindByWord(ctx context.Context, languageID domain.LanguageID, word string) (*domain.Vocabulary, error) {
	var vocabModel model.VocabularyModel
	if err := repo.db.WithContext(ctx).
		Where("language_id = ? AND word = ?", languageID.UUID(), word).
		First(&vocabModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return vocabModel.ToEntity(), nil
}

// FindByWordList finds vocabularies matching any of the given words in a language.
func (repo *VocabularyRepository) FindByWordList(ctx context.Context, languageID domain.LanguageID, words []string) ([]*domain.Vocabulary, error) {
	if len(words) == 0 {
		return []*domain.Vocabulary{}, nil
	}

	var models []model.VocabularyModel
	if err := repo.db.WithContext(ctx).
		Where("language_id = ? AND word IN ?", languageID.UUID(), words).
		Find(&models).Error; err != nil {
		return nil, err
	}
	return toVocabEntities(models), nil
}

// FindAll lists vocabularies with optional filters, returning lightweight entities (no meanings).
func (repo *VocabularyRepository) FindAll(ctx context.Context, languageID *domain.LanguageID, profLevelID *domain.LevelID, topicID *domain.TopicID, offset, limit int) ([]*domain.Vocabulary, error) {
	query := repo.db.WithContext(ctx).Model(&model.VocabularyModel{})
	query = applyVocabFilters(query, languageID, profLevelID, topicID)

	var models []model.VocabularyModel
	if err := query.Offset(offset).Limit(limit).Order("word ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	vocabs := toVocabEntities(models)
	if err := repo.loadMeanings(ctx, vocabs); err != nil {
		return nil, err
	}
	return vocabs, nil
}

// CountAll counts vocabularies matching the given filters.
func (repo *VocabularyRepository) CountAll(ctx context.Context, languageID *domain.LanguageID, profLevelID *domain.LevelID, topicID *domain.TopicID) (int64, error) {
	query := repo.db.WithContext(ctx).Model(&model.VocabularyModel{})
	query = applyVocabFilters(query, languageID, profLevelID, topicID)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Search searches vocabularies by word, phonetic, or meaning text.
func (repo *VocabularyRepository) Search(ctx context.Context, query string, languageID *domain.LanguageID, offset, limit int) ([]*domain.Vocabulary, error) {
	like := "%" + query + "%"

	dbQuery := repo.db.WithContext(ctx).Model(&model.VocabularyModel{})
	if languageID != nil {
		dbQuery = dbQuery.Where("language_id = ?", languageID.UUID())
	}

	dbQuery = dbQuery.Where(
		"word ILIKE ? OR phonetic ILIKE ? OR id IN (SELECT vocabulary_id FROM vocabulary_meanings WHERE meaning ILIKE ?)",
		like, like, like,
	)

	var models []model.VocabularyModel
	if err := dbQuery.Offset(offset).Limit(limit).Order("word ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	vocabs := toVocabEntities(models)
	if err := repo.loadMeanings(ctx, vocabs); err != nil {
		return nil, err
	}
	return vocabs, nil
}

// CountSearch counts vocabularies matching the search query.
func (repo *VocabularyRepository) CountSearch(ctx context.Context, query string, languageID *domain.LanguageID) (int64, error) {
	like := "%" + query + "%"

	dbQuery := repo.db.WithContext(ctx).Model(&model.VocabularyModel{})
	if languageID != nil {
		dbQuery = dbQuery.Where("language_id = ?", languageID.UUID())
	}

	dbQuery = dbQuery.Where(
		"word ILIKE ? OR phonetic ILIKE ? OR id IN (SELECT vocabulary_id FROM vocabulary_meanings WHERE meaning ILIKE ?)",
		like, like, like,
	)

	var count int64
	if err := dbQuery.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Update updates a vocabulary and replaces all meanings and examples in a transaction.
func (repo *VocabularyRepository) Update(ctx context.Context, vocab *domain.Vocabulary) error {
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		vocabModel := model.FromVocabularyEntity(vocab)
		if err := tx.Save(vocabModel).Error; err != nil {
			return err
		}
		vocab.UpdatedAt = vocabModel.UpdatedAt

		// Delete old examples (via meaning IDs) then old meanings
		var oldMeaningIDs []uuid.UUID
		if err := tx.Model(&model.MeaningModel{}).
			Where("vocabulary_id = ?", vocab.ID.UUID()).
			Pluck("id", &oldMeaningIDs).Error; err != nil {
			return err
		}

		if len(oldMeaningIDs) > 0 {
			if err := tx.Where("meaning_id IN ?", oldMeaningIDs).
				Delete(&model.ExampleModel{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("vocabulary_id = ?", vocab.ID.UUID()).
			Delete(&model.MeaningModel{}).Error; err != nil {
			return err
		}

		// Insert new meanings and examples
		for idx := range vocab.Meanings {
			meaning := &vocab.Meanings[idx]
			meaningModel := model.FromMeaningEntity(meaning)
			if err := tx.Create(meaningModel).Error; err != nil {
				return err
			}
			meaning.CreatedAt = meaningModel.CreatedAt
			meaning.UpdatedAt = meaningModel.UpdatedAt

			for exIdx := range meaning.Examples {
				example := &meaning.Examples[exIdx]
				exampleModel := model.FromExampleEntity(example)
				if err := tx.Create(exampleModel).Error; err != nil {
					return err
				}
				example.CreatedAt = exampleModel.CreatedAt
				example.UpdatedAt = exampleModel.UpdatedAt
			}
		}

		return nil
	})
}

// Delete soft-deletes a vocabulary by ID.
func (repo *VocabularyRepository) Delete(ctx context.Context, id domain.VocabularyID) error {
	return repo.db.WithContext(ctx).Delete(&model.VocabularyModel{}, "id = ?", id.UUID()).Error
}

// SaveBatch creates multiple vocabularies with their meanings and examples in a transaction.
func (repo *VocabularyRepository) SaveBatch(ctx context.Context, vocabs []*domain.Vocabulary) (int, error) {
	if len(vocabs) == 0 {
		return 0, nil
	}

	var totalCreated int
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		vocabModels := make([]model.VocabularyModel, 0, len(vocabs))
		for _, vocab := range vocabs {
			vocabModels = append(vocabModels, *model.FromVocabularyEntity(vocab))
		}

		result := tx.CreateInBatches(vocabModels, 100)
		if result.Error != nil {
			return result.Error
		}
		totalCreated = int(result.RowsAffected)

		// Insert meanings and examples for all vocabs
		for _, vocab := range vocabs {
			for idx := range vocab.Meanings {
				meaning := &vocab.Meanings[idx]
				meaningModel := model.FromMeaningEntity(meaning)
				if err := tx.Create(meaningModel).Error; err != nil {
					return err
				}

				for exIdx := range meaning.Examples {
					example := &meaning.Examples[exIdx]
					exampleModel := model.FromExampleEntity(example)
					if err := tx.Create(exampleModel).Error; err != nil {
						return err
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}
	return totalCreated, nil
}

// SetTopics replaces all topic associations for a vocabulary.
func (repo *VocabularyRepository) SetTopics(ctx context.Context, vocabID domain.VocabularyID, topicIDs []domain.TopicID) error {
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("vocabulary_id = ?", vocabID.UUID()).Delete(&model.VocabularyTopicModel{}).Error; err != nil {
			return err
		}
		for _, topicID := range topicIDs {
			join := model.VocabularyTopicModel{VocabularyID: vocabID.UUID(), TopicID: topicID.UUID()}
			if err := tx.Create(&join).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SetGrammarPoints replaces all grammar point associations for a vocabulary.
func (repo *VocabularyRepository) SetGrammarPoints(ctx context.Context, vocabID domain.VocabularyID, grammarPointIDs []domain.GrammarPointID) error {
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("vocabulary_id = ?", vocabID.UUID()).Delete(&model.VocabularyGrammarPointModel{}).Error; err != nil {
			return err
		}
		for _, gpID := range grammarPointIDs {
			join := model.VocabularyGrammarPointModel{VocabularyID: vocabID.UUID(), GrammarPointID: gpID.UUID()}
			if err := tx.Create(&join).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// loadMeanings loads meanings (without examples) for a list of vocabularies.
func (repo *VocabularyRepository) loadMeanings(ctx context.Context, vocabs []*domain.Vocabulary) error {
	if len(vocabs) == 0 {
		return nil
	}

	vocabIDs := make([]uuid.UUID, 0, len(vocabs))
	vocabMap := make(map[uuid.UUID]*domain.Vocabulary, len(vocabs))
	for _, vocab := range vocabs {
		id := vocab.ID.UUID()
		vocabIDs = append(vocabIDs, id)
		vocabMap[id] = vocab
	}

	var meaningModels []model.MeaningModel
	if err := repo.db.WithContext(ctx).
		Where("vocabulary_id IN ?", vocabIDs).
		Order(`"offset" ASC`).
		Find(&meaningModels).Error; err != nil {
		return err
	}

	for _, mm := range meaningModels {
		vocab, ok := vocabMap[mm.VocabularyID]
		if !ok {
			continue
		}
		vocab.Meanings = append(vocab.Meanings, *mm.ToEntity())
	}

	return nil
}

// loadMeaningsWithExamples loads meanings and their examples for a list of vocabularies.
func (repo *VocabularyRepository) loadMeaningsWithExamples(ctx context.Context, vocabs []*domain.Vocabulary) error {
	if len(vocabs) == 0 {
		return nil
	}

	vocabIDs := make([]uuid.UUID, 0, len(vocabs))
	vocabMap := make(map[uuid.UUID]*domain.Vocabulary, len(vocabs))
	for _, vocab := range vocabs {
		id := vocab.ID.UUID()
		vocabIDs = append(vocabIDs, id)
		vocabMap[id] = vocab
	}

	// Load meanings
	var meaningModels []model.MeaningModel
	if err := repo.db.WithContext(ctx).
		Where("vocabulary_id IN ?", vocabIDs).
		Order(`"offset" ASC`).
		Find(&meaningModels).Error; err != nil {
		return err
	}

	if len(meaningModels) == 0 {
		return nil
	}

	meaningIDs := make([]uuid.UUID, 0, len(meaningModels))

	// Attach meanings to vocabs
	for _, mm := range meaningModels {
		vocab, ok := vocabMap[mm.VocabularyID]
		if !ok {
			continue
		}
		entity := mm.ToEntity()
		vocab.Meanings = append(vocab.Meanings, *entity)
		meaningIDs = append(meaningIDs, mm.ID)
	}

	// Build a map from meaning UUID to the actual slice element inside vocab.Meanings
	meaningRefMap := make(map[uuid.UUID]*domain.VocabularyMeaning, len(meaningIDs))
	for _, vocab := range vocabs {
		for idx := range vocab.Meanings {
			meaningRefMap[vocab.Meanings[idx].ID.UUID()] = &vocab.Meanings[idx]
		}
	}

	// Load examples
	var exampleModels []model.ExampleModel
	if err := repo.db.WithContext(ctx).
		Where("meaning_id IN ?", meaningIDs).
		Order(`"offset" ASC`).
		Find(&exampleModels).Error; err != nil {
		return err
	}

	for _, em := range exampleModels {
		meaning, ok := meaningRefMap[em.MeaningID]
		if !ok {
			continue
		}
		meaning.Examples = append(meaning.Examples, *em.ToEntity())
	}

	return nil
}

// applyVocabFilters adds optional WHERE clauses for vocabulary list queries.
func applyVocabFilters(query *gorm.DB, languageID *domain.LanguageID, profLevelID *domain.LevelID, topicID *domain.TopicID) *gorm.DB {
	if languageID != nil {
		query = query.Where("vocabularies.language_id = ?", languageID.UUID())
	}
	if profLevelID != nil {
		query = query.Where("vocabularies.level_id = ?", profLevelID.UUID())
	}
	if topicID != nil {
		query = query.Joins("JOIN vocabulary_topics vt ON vt.vocabulary_id = vocabularies.id").
			Where("vt.topic_id = ?", topicID.UUID())
	}
	return query
}

func toVocabEntities(models []model.VocabularyModel) []*domain.Vocabulary {
	result := make([]*domain.Vocabulary, 0, len(models))
	for _, m := range models {
		result = append(result, m.ToEntity())
	}
	return result
}
