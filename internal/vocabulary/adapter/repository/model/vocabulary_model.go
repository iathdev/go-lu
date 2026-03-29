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

func (m *VocabularyModel) ToEntity() *domain.Vocabulary {
	examples := make([]domain.Example, len(m.Examples.Data))
	copy(examples, m.Examples.Data)

	radicals := make([]string, len(m.Radicals))
	copy(radicals, m.Radicals)

	return &domain.Vocabulary{
		ID:              domain.VocabularyIDFromUUID(m.ID),
		Hanzi:           domain.Hanzi(m.Hanzi),
		Pinyin:          domain.Pinyin(m.Pinyin),
		MeaningVI:       m.MeaningVI,
		MeaningEN:       m.MeaningEN,
		HSKLevel:        domain.HSKLevel(m.HSKLevel),
		AudioURL:        m.AudioURL,
		Examples:        examples,
		Radicals:        radicals,
		StrokeCount:     m.StrokeCount,
		StrokeDataURL:   m.StrokeDataURL,
		RecognitionOnly: m.RecognitionOnly,
		FrequencyRank:   m.FrequencyRank,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func FromVocabularyEntity(vocab *domain.Vocabulary) *VocabularyModel {
	radicals := make(pq.StringArray, len(vocab.Radicals))
	copy(radicals, vocab.Radicals)

	return &VocabularyModel{
		ID:              vocab.ID.UUID(),
		Hanzi:           vocab.Hanzi.String(),
		Pinyin:          vocab.Pinyin.String(),
		MeaningVI:       vocab.MeaningVI,
		MeaningEN:       vocab.MeaningEN,
		HSKLevel:        vocab.HSKLevel.Int(),
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
