package usecase

import (
	"context"

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

func (useCase *FolderQuery) ListFolders(ctx context.Context, userID string, languageID string) ([]*vdto.FolderResponse, error) {
	uid, err := domain.ParseUserID(userID)
	if err != nil {
		return nil, apperr.BadRequest("folder.invalid_user_id")
	}

	var langIDPtr *domain.LanguageID
	if languageID != "" {
		parsed, err := domain.ParseLanguageID(languageID)
		if err != nil {
			return nil, apperr.BadRequest("folder.invalid_language_id")
		}
		langIDPtr = &parsed
	}

	folders, err := useCase.folderRepo.FindByUserID(ctx, uid, langIDPtr)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	// Collect folder IDs for batch count
	folderIDs := make([]domain.FolderID, 0, len(folders))
	for _, folder := range folders {
		folderIDs = append(folderIDs, folder.ID)
	}

	countMap := make(map[domain.FolderID]int)
	if len(folderIDs) > 0 {
		countMap, err = useCase.folderRepo.CountVocabulariesByFolderIDs(ctx, folderIDs)
		if err != nil {
			// Non-critical: return folders with zero counts rather than failing
			countMap = make(map[domain.FolderID]int)
		}
	}

	result := make([]*vdto.FolderResponse, 0, len(folders))
	for _, folder := range folders {
		result = append(result, mapper.ToFolderResponse(folder, countMap[folder.ID]))
	}
	return result, nil
}

func (useCase *FolderQuery) ListVocabularies(ctx context.Context, folderID string, userID string, _ string, pagination dto.PaginationRequest) (*dto.ListResult[*vdto.VocabularyListResponse], error) {
	folder, err := getOwnedFolder(ctx, useCase.folderRepo, folderID, userID)
	if err != nil {
		return nil, err
	}

	normalizePagination(&pagination)
	offset := (pagination.Page - 1) * pagination.PageSize

	total, err := useCase.folderRepo.CountVocabularies(ctx, folder.ID)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	vocabs, err := useCase.folderRepo.FindVocabularies(ctx, folder.ID, offset, pagination.PageSize)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	return mapper.ToPaginatedListResult(vocabs, total, pagination), nil
}
