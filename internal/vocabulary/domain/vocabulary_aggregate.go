package domain

import "time"

// VocabularyParams carries raw input for creating or updating a Vocabulary.
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

// Vocabulary is the aggregate root for vocabulary content.
// Invariants: Hanzi + Pinyin required, HSKLevel 1-9, at least one meaning.
type Vocabulary struct {
	ID              VocabularyID
	Hanzi           Hanzi
	Pinyin          Pinyin
	MeaningVI       string
	MeaningEN       string
	HSKLevel        HSKLevel
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

func NewVocabularyFromParams(params VocabularyParams) (*Vocabulary, error) {
	hanzi, err := NewHanzi(params.Hanzi)
	if err != nil {
		return nil, err
	}
	pinyin, err := NewPinyin(params.Pinyin)
	if err != nil {
		return nil, err
	}
	if params.MeaningVI == "" && params.MeaningEN == "" {
		return nil, ErrMeaningRequired
	}
	hskLevel, err := NewHSKLevel(params.HSKLevel)
	if err != nil {
		return nil, err
	}

	return &Vocabulary{
		ID:              NewVocabularyID(),
		Hanzi:           hanzi,
		Pinyin:          pinyin,
		MeaningVI:       params.MeaningVI,
		MeaningEN:       params.MeaningEN,
		HSKLevel:        hskLevel,
		AudioURL:        params.AudioURL,
		Examples:        params.Examples,
		Radicals:        params.Radicals,
		StrokeCount:     params.StrokeCount,
		StrokeDataURL:   params.StrokeDataURL,
		RecognitionOnly: params.RecognitionOnly,
		FrequencyRank:   params.FrequencyRank,
	}, nil
}

func (vocab *Vocabulary) UpdateFromParams(params VocabularyParams) error {
	hanzi, err := NewHanzi(params.Hanzi)
	if err != nil {
		return err
	}
	pinyin, err := NewPinyin(params.Pinyin)
	if err != nil {
		return err
	}
	if params.MeaningVI == "" && params.MeaningEN == "" {
		return ErrMeaningRequired
	}
	hskLevel, err := NewHSKLevel(params.HSKLevel)
	if err != nil {
		return err
	}

	vocab.Hanzi = hanzi
	vocab.Pinyin = pinyin
	vocab.MeaningVI = params.MeaningVI
	vocab.MeaningEN = params.MeaningEN
	vocab.HSKLevel = hskLevel
	vocab.AudioURL = params.AudioURL
	vocab.Examples = params.Examples
	vocab.Radicals = params.Radicals
	vocab.StrokeCount = params.StrokeCount
	vocab.StrokeDataURL = params.StrokeDataURL
	vocab.RecognitionOnly = params.RecognitionOnly
	vocab.FrequencyRank = params.FrequencyRank
	return nil
}
