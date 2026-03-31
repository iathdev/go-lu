package domain

import "time"

// Vocabulary is the aggregate root for vocabulary content.
// Language-agnostic: uses generic field names (word, phonetic).
// Language-specific data goes into Metadata JSONB.
type Vocabulary struct {
	ID                 VocabularyID
	LanguageID         LanguageID
	ProficiencyLevelID ProficiencyLevelID
	Word               string
	Phonetic           string
	AudioURL           string
	ImageURL           string
	FrequencyRank      int
	Metadata           map[string]any
	Meanings           []VocabularyMeaning
	TopicIDs           []TopicID
	GrammarPointIDs    []GrammarPointID
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// VocabularyMeaning represents a meaning in a target language.
type VocabularyMeaning struct {
	ID           MeaningID
	VocabularyID VocabularyID
	LanguageID   LanguageID
	Meaning      string
	WordType     string
	IsPrimary    bool
	Offset       int
	Examples     []VocabularyExample
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// VocabularyExample is an example sentence for a meaning.
type VocabularyExample struct {
	ID           ExampleID
	MeaningID    MeaningID
	Sentence     string
	Phonetic     string
	Translations map[string]string
	AudioURL     string
	Offset       int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// VocabularyParams carries typed input for creating or updating a Vocabulary.
type VocabularyParams struct {
	LanguageID         LanguageID
	ProficiencyLevelID ProficiencyLevelID
	Word               string
	Phonetic           string
	AudioURL           string
	ImageURL           string
	FrequencyRank      int
	Metadata           map[string]any
	Meanings           []MeaningParams
}

// MeaningParams carries typed input for a meaning.
type MeaningParams struct {
	LanguageID LanguageID
	Meaning    string
	WordType   string
	IsPrimary  bool
	Examples   []ExampleParams
}

// ExampleParams carries raw input for an example.
type ExampleParams struct {
	Sentence     string
	Phonetic     string
	Translations map[string]string
	AudioURL     string
}

func NewVocabularyFromParams(params VocabularyParams) (*Vocabulary, error) {
	if params.Word == "" {
		return nil, ErrWordRequired
	}
	if len(params.Meanings) == 0 {
		return nil, ErrMeaningRequired
	}

	vocabID := NewVocabularyID()
	meanings := buildMeanings(vocabID, params.Meanings)

	return &Vocabulary{
		ID:                 vocabID,
		LanguageID:         params.LanguageID,
		ProficiencyLevelID: params.ProficiencyLevelID,
		Word:               params.Word,
		Phonetic:           params.Phonetic,
		AudioURL:           params.AudioURL,
		ImageURL:           params.ImageURL,
		FrequencyRank:      params.FrequencyRank,
		Metadata:           params.Metadata,
		Meanings:           meanings,
	}, nil
}

func (vocab *Vocabulary) Update(params VocabularyParams) error {
	if params.Word == "" {
		return ErrWordRequired
	}
	if len(params.Meanings) == 0 {
		return ErrMeaningRequired
	}

	vocab.LanguageID = params.LanguageID
	vocab.ProficiencyLevelID = params.ProficiencyLevelID
	vocab.Word = params.Word
	vocab.Phonetic = params.Phonetic
	vocab.AudioURL = params.AudioURL
	vocab.ImageURL = params.ImageURL
	vocab.FrequencyRank = params.FrequencyRank
	vocab.Metadata = params.Metadata
	vocab.Meanings = buildMeanings(vocab.ID, params.Meanings)
	return nil
}

func (vocab *Vocabulary) SetTopics(topicIDs []TopicID) {
	vocab.TopicIDs = topicIDs
}

func (vocab *Vocabulary) SetGrammarPoints(gpIDs []GrammarPointID) {
	vocab.GrammarPointIDs = gpIDs
}

func buildMeanings(vocabID VocabularyID, params []MeaningParams) []VocabularyMeaning {
	meanings := make([]VocabularyMeaning, 0, len(params))
	for idx, mp := range params {
		meaningID := NewMeaningID()
		examples := make([]VocabularyExample, 0, len(mp.Examples))
		for exIdx, ep := range mp.Examples {
			examples = append(examples, VocabularyExample{
				ID:           NewExampleID(),
				MeaningID:    meaningID,
				Sentence:     ep.Sentence,
				Phonetic:     ep.Phonetic,
				Translations: ep.Translations,
				AudioURL:     ep.AudioURL,
				Offset:       exIdx,
			})
		}

		meanings = append(meanings, VocabularyMeaning{
			ID:           meaningID,
			VocabularyID: vocabID,
			LanguageID:   mp.LanguageID,
			Meaning:      mp.Meaning,
			WordType:     mp.WordType,
			IsPrimary:    mp.IsPrimary,
			Offset:       idx,
			Examples:     examples,
		})
	}
	return meanings
}
