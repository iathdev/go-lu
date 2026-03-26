package port

import (
	"context"
	"learning-go/internal/vocabulary/domain"

	"github.com/google/uuid"
)

type VocabularyRepositoryPort interface {
	Save(ctx context.Context, vocab *domain.Vocabulary) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Vocabulary, error)
	FindByHanzi(ctx context.Context, hanzi string) (*domain.Vocabulary, error)
	FindByHanziList(ctx context.Context, hanziList []string) ([]*domain.Vocabulary, error)
	FindByHSKLevel(ctx context.Context, level int, offset, limit int) ([]*domain.Vocabulary, error)
	CountByHSKLevel(ctx context.Context, level int) (int64, error)
	FindByTopicID(ctx context.Context, topicID uuid.UUID, offset, limit int) ([]*domain.Vocabulary, error)
	CountByTopicID(ctx context.Context, topicID uuid.UUID) (int64, error)
	Search(ctx context.Context, query string, offset, limit int) ([]*domain.Vocabulary, error)
	CountSearch(ctx context.Context, query string) (int64, error)
	Update(ctx context.Context, vocab *domain.Vocabulary) error
	Delete(ctx context.Context, id uuid.UUID) error
	SaveBatch(ctx context.Context, vocabs []*domain.Vocabulary) (int, error)
	SetTopics(ctx context.Context, vocabID uuid.UUID, topicIDs []uuid.UUID) error
	SetGrammarPoints(ctx context.Context, vocabID uuid.UUID, grammarPointIDs []uuid.UUID) error
}

type FolderRepositoryPort interface {
	Save(ctx context.Context, folder *domain.Folder) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Folder, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Folder, error)
	Update(ctx context.Context, folder *domain.Folder) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddVocabulary(ctx context.Context, folderID, vocabID uuid.UUID) error
	RemoveVocabulary(ctx context.Context, folderID, vocabID uuid.UUID) error
	FindVocabularies(ctx context.Context, folderID uuid.UUID, offset, limit int) ([]*domain.Vocabulary, error)
	CountVocabularies(ctx context.Context, folderID uuid.UUID) (int64, error)
}

type TopicRepositoryPort interface {
	FindAll(ctx context.Context) ([]*domain.Topic, error)
	FindBySlug(ctx context.Context, slug string) (*domain.Topic, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Topic, error)
	FindByVocabularyID(ctx context.Context, vocabID uuid.UUID) ([]*domain.Topic, error)
}

type GrammarPointRepositoryPort interface {
	FindByVocabularyID(ctx context.Context, vocabID uuid.UUID) ([]*domain.GrammarPoint, error)
	FindByHSKLevel(ctx context.Context, level int) ([]*domain.GrammarPoint, error)
	FindByCode(ctx context.Context, code string) (*domain.GrammarPoint, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.GrammarPoint, error)
}

type OCRScannerPort interface {
	ProcessScan(ctx context.Context, req OCRScanInput) (*OCRScanOutput, error)
}

type OCRScanInput struct {
	Image    []byte
	Type     string // "printed" | "handwritten" | "auto"
	Language string // "zh" | "vi" | "en"
	Engine   string // optional: force specific engine
}

type OCRScanOutput struct {
	Items         []OCRCharacterOutput `json:"items"`
	LowConfidence []OCRCharacterOutput `json:"low_confidence"`
	EngineUsed    string               `json:"engine_used"`
	TotalDetected int                  `json:"total_detected"`
	ProcessingMs  int64                `json:"processing_ms"`
}

type OCRCharacterOutput struct {
	Text          string   `json:"text"`
	Pronunciation string   `json:"pronunciation,omitempty"`
	Confidence    float64  `json:"confidence"`
	Candidates    []string `json:"candidates,omitempty"`
}
