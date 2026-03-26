package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrHanziRequired   = errors.New("hanzi is required")
	ErrPinyinRequired  = errors.New("pinyin is required")
	ErrInvalidHSKLevel = errors.New("hsk level must be between 1 and 9")
	ErrMeaningRequired = errors.New("at least one meaning (vi or en) is required")
)

type Example struct {
	SentenceCN string `json:"sentence_cn"`
	SentenceVI string `json:"sentence_vi"`
	AudioURL   string `json:"audio_url,omitempty"`
}

type VocabularyParams struct {
	Hanzi           string
	Pinyin          string
	MeaningVI       string
	MeaningEN       string
	HSKLevel        int
	AudioURL        string
	Examples        []Example
	Radicals        []string
	StrokeCount     int
	StrokeDataURL   string
	RecognitionOnly bool
	FrequencyRank   int
}

type Vocabulary struct {
	ID              uuid.UUID
	Hanzi           string
	Pinyin          string
	MeaningVI       string
	MeaningEN       string
	HSKLevel        int
	AudioURL        string
	Examples        []Example
	Radicals        []string
	StrokeCount     int
	StrokeDataURL   string
	RecognitionOnly bool
	FrequencyRank   int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewVocabulary(hanzi, pinyin, meaningVI, meaningEN string, hskLevel int, topic string) (*Vocabulary, error) {
	return NewVocabularyFromParams(VocabularyParams{
		Hanzi:     hanzi,
		Pinyin:    pinyin,
		MeaningVI: meaningVI,
		MeaningEN: meaningEN,
		HSKLevel:  hskLevel,
	})
}

func NewVocabularyFromParams(params VocabularyParams) (*Vocabulary, error) {
	if params.Hanzi == "" {
		return nil, ErrHanziRequired
	}
	if params.Pinyin == "" {
		return nil, ErrPinyinRequired
	}
	if params.MeaningVI == "" && params.MeaningEN == "" {
		return nil, ErrMeaningRequired
	}
	if params.HSKLevel < 1 || params.HSKLevel > 9 {
		return nil, ErrInvalidHSKLevel
	}

	return &Vocabulary{
		ID:              uuid.Must(uuid.NewV7()),
		Hanzi:           params.Hanzi,
		Pinyin:          params.Pinyin,
		MeaningVI:       params.MeaningVI,
		MeaningEN:       params.MeaningEN,
		HSKLevel:        params.HSKLevel,
		AudioURL:        params.AudioURL,
		Examples:        params.Examples,
		Radicals:        params.Radicals,
		StrokeCount:     params.StrokeCount,
		StrokeDataURL:   params.StrokeDataURL,
		RecognitionOnly: params.RecognitionOnly,
		FrequencyRank:   params.FrequencyRank,
	}, nil
}

func (vocab *Vocabulary) Update(hanzi, pinyin, meaningVI, meaningEN string, hskLevel int, topic string) error {
	return vocab.UpdateFromParams(VocabularyParams{
		Hanzi:           hanzi,
		Pinyin:          pinyin,
		MeaningVI:       meaningVI,
		MeaningEN:       meaningEN,
		HSKLevel:        hskLevel,
		AudioURL:        vocab.AudioURL,
		Examples:        vocab.Examples,
		Radicals:        vocab.Radicals,
		StrokeCount:     vocab.StrokeCount,
		StrokeDataURL:   vocab.StrokeDataURL,
		RecognitionOnly: vocab.RecognitionOnly,
		FrequencyRank:   vocab.FrequencyRank,
	})
}

func (vocab *Vocabulary) UpdateFromParams(params VocabularyParams) error {
	if params.Hanzi == "" {
		return ErrHanziRequired
	}
	if params.Pinyin == "" {
		return ErrPinyinRequired
	}
	if params.MeaningVI == "" && params.MeaningEN == "" {
		return ErrMeaningRequired
	}
	if params.HSKLevel < 1 || params.HSKLevel > 9 {
		return ErrInvalidHSKLevel
	}

	vocab.Hanzi = params.Hanzi
	vocab.Pinyin = params.Pinyin
	vocab.MeaningVI = params.MeaningVI
	vocab.MeaningEN = params.MeaningEN
	vocab.HSKLevel = params.HSKLevel
	vocab.AudioURL = params.AudioURL
	vocab.Examples = params.Examples
	vocab.Radicals = params.Radicals
	vocab.StrokeCount = params.StrokeCount
	vocab.StrokeDataURL = params.StrokeDataURL
	vocab.RecognitionOnly = params.RecognitionOnly
	vocab.FrequencyRank = params.FrequencyRank
	return nil
}
