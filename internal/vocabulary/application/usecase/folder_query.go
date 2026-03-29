package usecase

import (
	"context"
	"math"

	"learning-go/internal/shared/dto"
	apperr "learning-go/internal/shared/error"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"
)

type FolderQuery struct {
	folderRepo port.FolderRepositoryPort
}

func NewFolderQuery(folderRepo port.FolderRepositoryPort) port.FolderQueryPort {
	return &FolderQuery{folderRepo: folderRepo}
}

func (useCase *FolderQuery) ListFolders(ctx context.Context, userID string) ([]*vdto.FolderResponse, error) {
	uid, err := domain.ParseUserID(userID)
	if err != nil {
		return nil, apperr.BadRequest("folder.invalid_user_id")
	}

	folders, err := useCase.folderRepo.FindByUserID(ctx, uid)
	if err != nil {
		return nil, apperr.InternalServerError("folder.query_failed", err)
	}

	result := make([]*vdto.FolderResponse, 0, len(folders))
	for _, folder := range folders {
		result = append(result, mapper.ToFolderResponse(folder))
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
	for _, vocab := range vocabs {
		items = append(items, mapper.ToVocabularyResponse(vocab))
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
