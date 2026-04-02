package port

import (
	"context"
	"learning-go/internal/vocabulary/domain"
)

type LanguageRepositoryPort interface {
	FindAll(ctx context.Context, activeOnly bool) ([]*domain.Language, error)
	FindByID(ctx context.Context, id domain.LanguageID) (*domain.Language, error)
}

type CategoryRepositoryPort interface {
	FindAll(ctx context.Context, languageID *domain.LanguageID, isPublic *bool) ([]*domain.Category, error)
	FindByID(ctx context.Context, id domain.CategoryID) (*domain.Category, error)
}

type ProficiencyLevelRepositoryPort interface {
	FindAll(ctx context.Context, categoryID *domain.CategoryID) ([]*domain.ProficiencyLevel, error)
	FindByID(ctx context.Context, id domain.LevelID) (*domain.ProficiencyLevel, error)
}

type VocabularyRepositoryPort interface {
	Save(ctx context.Context, vocab *domain.Vocabulary) error
	FindByID(ctx context.Context, id domain.VocabularyID) (*domain.Vocabulary, error)
	FindByWord(ctx context.Context, languageID domain.LanguageID, word string) (*domain.Vocabulary, error)
	FindByWordList(ctx context.Context, languageID domain.LanguageID, words []string) ([]*domain.Vocabulary, error)
	FindAll(ctx context.Context, languageID *domain.LanguageID, profLevelID *domain.LevelID, topicID *domain.TopicID, offset, limit int) ([]*domain.Vocabulary, error)
	CountAll(ctx context.Context, languageID *domain.LanguageID, profLevelID *domain.LevelID, topicID *domain.TopicID) (int64, error)
	Search(ctx context.Context, query string, languageID *domain.LanguageID, offset, limit int) ([]*domain.Vocabulary, error)
	CountSearch(ctx context.Context, query string, languageID *domain.LanguageID) (int64, error)
	Update(ctx context.Context, vocab *domain.Vocabulary) error
	Delete(ctx context.Context, id domain.VocabularyID) error
	SaveBatch(ctx context.Context, vocabs []*domain.Vocabulary) (int, error)
	SetTopics(ctx context.Context, vocabID domain.VocabularyID, topicIDs []domain.TopicID) error
	SetGrammarPoints(ctx context.Context, vocabID domain.VocabularyID, grammarPointIDs []domain.GrammarPointID) error
}

type FolderRepositoryPort interface {
	Save(ctx context.Context, folder *domain.Folder) error
	FindByID(ctx context.Context, id domain.FolderID) (*domain.Folder, error)
	FindByUserID(ctx context.Context, userID domain.UserID, languageID *domain.LanguageID) ([]*domain.Folder, error)
	CountVocabulariesByFolderIDs(ctx context.Context, folderIDs []domain.FolderID) (map[domain.FolderID]int, error)
	Update(ctx context.Context, folder *domain.Folder) error
	Delete(ctx context.Context, id domain.FolderID) error
	AddVocabulary(ctx context.Context, folderID domain.FolderID, vocabID domain.VocabularyID) error
	RemoveVocabulary(ctx context.Context, folderID domain.FolderID, vocabID domain.VocabularyID) error
	FindVocabularies(ctx context.Context, folderID domain.FolderID, offset, limit int) ([]*domain.Vocabulary, error)
	CountVocabularies(ctx context.Context, folderID domain.FolderID) (int64, error)
}

type TopicRepositoryPort interface {
	FindAll(ctx context.Context, categoryID *domain.CategoryID) ([]*domain.Topic, error)
	FindByID(ctx context.Context, id domain.TopicID) (*domain.Topic, error)
	FindByIDs(ctx context.Context, ids []domain.TopicID) ([]*domain.Topic, error)
	FindByVocabularyID(ctx context.Context, vocabID domain.VocabularyID) ([]*domain.Topic, error)
}

type GrammarPointRepositoryPort interface {
	FindAll(ctx context.Context, categoryID *domain.CategoryID, profLevelID *domain.LevelID, offset, limit int) ([]*domain.GrammarPoint, error)
	CountAll(ctx context.Context, categoryID *domain.CategoryID, profLevelID *domain.LevelID) (int64, error)
	FindByID(ctx context.Context, id domain.GrammarPointID) (*domain.GrammarPoint, error)
	FindByIDs(ctx context.Context, ids []domain.GrammarPointID) ([]*domain.GrammarPoint, error)
	FindByVocabularyID(ctx context.Context, vocabID domain.VocabularyID) ([]*domain.GrammarPoint, error)
}

type OCRScannerPort interface {
	ProcessScan(ctx context.Context, req OCRScanInput) (*OCRScanOutput, error)
	ExtractText(ctx context.Context, req OCRScanInput) (*OCRExtractTextOutput, error)
}

type OCRScanInput struct {
	Image    []byte
	Type     string
	Language string
	Engine   string
}

type OCRScanOutput struct {
	Items         []OCRCharacterOutput `json:"items"`
	LowConfidence []OCRCharacterOutput `json:"low_confidence"`
	EngineUsed    string               `json:"engine_used"`
	TotalDetected int                  `json:"total_detected"`
	ProcessingMs  int64                `json:"processing_ms"`
}

type OCRExtractTextOutput struct {
	Blocks   []OCRTextBlockOutput `json:"blocks"`
	FullText string               `json:"full_text"`
	Metadata OCRExtractTextMeta   `json:"metadata"`
}

type OCRTextBlockOutput struct {
	Text        string             `json:"text"`
	BoundingBox *BoundingBoxOutput `json:"bounding_box,omitempty"`
	Confidence  float64            `json:"confidence"`
}

type OCRExtractTextMeta struct {
	EngineUsed       string `json:"engine_used"`
	TotalBlocks      int    `json:"total_blocks"`
	ProcessingTimeMs int64  `json:"processing_time_ms"`
}

type BoundingBoxOutput struct {
	Vertices []PointOutput `json:"vertices"`
}

type PointOutput struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type OCRCharacterOutput struct {
	Text          string            `json:"text"`
	Pronunciation string            `json:"pronunciation,omitempty"`
	Confidence    float64           `json:"confidence"`
	Candidates    []string          `json:"candidates,omitempty"`
	BoundingBox   *BoundingBoxOutput `json:"bounding_box,omitempty"`
}
