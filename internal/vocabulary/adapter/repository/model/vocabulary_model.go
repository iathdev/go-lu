package model

import (
	"learning-go/internal/shared/common"
	"learning-go/internal/vocabulary/domain"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type ExamplesJSON = common.JSONB[[]domain.Example]

type VocabularyModel struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key;"`
	Hanzi           string         `gorm:"not null"`
	Pinyin          string         `gorm:"not null"`
	MeaningVI       string
	MeaningEN       string
	HSKLevel        int            `gorm:"not null;index"`
	AudioURL        string
	Examples        ExamplesJSON   `gorm:"type:jsonb;default:'[]'"`
	Radicals        pq.StringArray `gorm:"type:text[];default:'{}'"`
	StrokeCount     int
	StrokeDataURL   string         `gorm:"column:stroke_data_url"`
	RecognitionOnly bool           `gorm:"default:false"`
	FrequencyRank   int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (VocabularyModel) TableName() string { return "vocabularies" }

func (model *VocabularyModel) ToEntity() *domain.Vocabulary {
	examples := make([]domain.Example, len(model.Examples.Data))
	copy(examples, model.Examples.Data)

	radicals := make([]string, len(model.Radicals))
	copy(radicals, model.Radicals)

	return &domain.Vocabulary{
		ID:              model.ID,
		Hanzi:           model.Hanzi,
		Pinyin:          model.Pinyin,
		MeaningVI:       model.MeaningVI,
		MeaningEN:       model.MeaningEN,
		HSKLevel:        model.HSKLevel,
		AudioURL:        model.AudioURL,
		Examples:        examples,
		Radicals:        radicals,
		StrokeCount:     model.StrokeCount,
		StrokeDataURL:   model.StrokeDataURL,
		RecognitionOnly: model.RecognitionOnly,
		FrequencyRank:   model.FrequencyRank,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

func FromVocabularyEntity(vocab *domain.Vocabulary) *VocabularyModel {
	radicals := make(pq.StringArray, len(vocab.Radicals))
	copy(radicals, vocab.Radicals)

	return &VocabularyModel{
		ID:              vocab.ID,
		Hanzi:           vocab.Hanzi,
		Pinyin:          vocab.Pinyin,
		MeaningVI:       vocab.MeaningVI,
		MeaningEN:       vocab.MeaningEN,
		HSKLevel:        vocab.HSKLevel,
		AudioURL:        vocab.AudioURL,
		Examples:        common.NewJSONB(vocab.Examples),
		Radicals:        radicals,
		StrokeCount:     vocab.StrokeCount,
		StrokeDataURL:   vocab.StrokeDataURL,
		RecognitionOnly: vocab.RecognitionOnly,
		FrequencyRank:   vocab.FrequencyRank,
		CreatedAt:       vocab.CreatedAt,
		UpdatedAt:       vocab.UpdatedAt,
	}
}
