package usecase

import (
	"context"
	"learning-go/internal/shared/dto"
	apperr "learning-go/internal/shared/error"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"
	"math"

	"github.com/google/uuid"
)

type FolderQuery struct {
	folderRepo port.FolderRepositoryPort
}

func NewFolderQuery(folderRepo port.FolderRepositoryPort) port.FolderQueryPort {
	return &FolderQuery{folderRepo: folderRepo}
}

func (useCase *FolderQuery) ListFolders(ctx context.Context, userID string) ([]*vdto.FolderResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, apperr.BadRequest("folder.invalid_user_id")
	}

	folders, err := useCase.folderRepo.FindByUserID(ctx, uid)
	if err != nil {
		return nil, apperr.InternalServerError("folder.query_failed", err)
	}

	result := make([]*vdto.FolderResponse, 0, len(folders))
	for _, f := range folders {
		result = append(result, mapper.ToFolderResponse(f))
	}
	return result, nil
}

func (useCase *FolderQuery) ListVocabularies(ctx context.Context, folderID string, userID string, pagination dto.PaginationRequest) (*dto.PaginatedResponse, error) {
	folder, err := getOwnedFolder(ctx, useCase.folderRepo, folderID, userID)
	if err != nil {
		return nil, err
	}

	normalizePagination(&pagination)
	offset := (pagination.Page - 1) * pagination.PageSize

	total, err := useCase.folderRepo.CountVocabularies(ctx, folder.ID)
	if err != nil {
		return nil, apperr.InternalServerError("folder.query_failed", err)
	}

	vocabs, err := useCase.folderRepo.FindVocabularies(ctx, folder.ID, offset, pagination.PageSize)
	if err != nil {
		return nil, apperr.InternalServerError("folder.query_failed", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(pagination.PageSize)))
	items := make([]*vdto.VocabularyResponse, 0, len(vocabs))
	for _, v := range vocabs {
		items = append(items, mapper.ToVocabularyResponse(v))
	}

	return &dto.PaginatedResponse{
		Items: items,
		Metadata: dto.PaginationMeta{
			Total:      total,
			Page:       pagination.Page,
			PageSize:   pagination.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// shared helpers

func getOwnedFolder(ctx context.Context, folderRepo port.FolderRepositoryPort, id string, userID string) (*domain.Folder, error) {
	fid, err := uuid.Parse(id)
	if err != nil {
		return nil, apperr.BadRequest("folder.invalid_id")
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, apperr.BadRequest("folder.invalid_user_id")
	}

	folder, err := folderRepo.FindByID(ctx, fid)
	if err != nil {
		return nil, apperr.InternalServerError("folder.query_failed", err)
	}
	if folder == nil {
		return nil, apperr.NotFound("folder.not_found")
	}

	if folder.UserID != uid {
		return nil, apperr.NotFound("folder.not_found")
	}

	return folder, nil
}

