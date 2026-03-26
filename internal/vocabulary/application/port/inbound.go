package port

import (
	"context"
	"learning-go/internal/shared/dto"
	vdto "learning-go/internal/vocabulary/application/dto"
)

// Input ports (driving) — used by handlers to call usecases

// --- Vocabulary Ports ---

type VocabularyCommandPort interface {
	CreateVocabulary(ctx context.Context, req vdto.CreateVocabularyRequest) (*vdto.VocabularyResponse, error)
	UpdateVocabulary(ctx context.Context, id string, req vdto.UpdateVocabularyRequest) (*vdto.VocabularyResponse, error)
	DeleteVocabulary(ctx context.Context, id string) error
}

type VocabularyQueryPort interface {
	GetVocabulary(ctx context.Context, id string) (*vdto.VocabularyResponse, error)
	GetVocabularyDetail(ctx context.Context, id string) (*vdto.VocabularyDetailResponse, error)
	ListByHSKLevel(ctx context.Context, level int, pagination dto.PaginationRequest) (*dto.PaginatedResponse, error)
	ListByTopic(ctx context.Context, slug string, pagination dto.PaginationRequest) (*dto.PaginatedResponse, error)
	SearchVocabulary(ctx context.Context, query string, pagination dto.PaginationRequest) (*dto.PaginatedResponse, error)
}

// --- Topic Ports ---

type TopicQueryPort interface {
	ListTopics(ctx context.Context) ([]*vdto.TopicResponse, error)
}

// --- OCR Ports ---

type OCRScanCommandPort interface {
	ProcessOCRScan(ctx context.Context, req OCRScanInput) (*vdto.OCRScanResponse, error)
}

// --- Import Ports ---

type ImportCommandPort interface {
	ImportVocabularies(ctx context.Context, req vdto.BulkImportRequest) (*vdto.BulkImportResponse, error)
}

// --- Folder Ports ---

type FolderCommandPort interface {
	CreateFolder(ctx context.Context, userID string, req vdto.CreateFolderRequest) (*vdto.FolderResponse, error)
	UpdateFolder(ctx context.Context, id string, userID string, req vdto.UpdateFolderRequest) (*vdto.FolderResponse, error)
	DeleteFolder(ctx context.Context, id string, userID string) error
	AddVocabulary(ctx context.Context, folderID string, vocabID string, userID string) error
	RemoveVocabulary(ctx context.Context, folderID string, vocabID string, userID string) error
}

type FolderQueryPort interface {
	ListFolders(ctx context.Context, userID string) ([]*vdto.FolderResponse, error)
	ListVocabularies(ctx context.Context, folderID string, userID string, pagination dto.PaginationRequest) (*dto.PaginatedResponse, error)
}
