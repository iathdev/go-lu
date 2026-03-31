package usecase

import (
	"context"
	"errors"

	"learning-go/internal/shared/dto"
	apperr "learning-go/internal/shared/error"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"
)

func getOwnedFolder(ctx context.Context, folderRepo port.FolderRepositoryPort, id string, userID string) (*domain.Folder, error) {
	fid, err := domain.ParseFolderID(id)
	if err != nil {
		return nil, apperr.BadRequest("folder.invalid_id")
	}

	uid, err := domain.ParseUserID(userID)
	if err != nil {
		return nil, apperr.BadRequest("folder.invalid_user_id")
	}

	folder, err := folderRepo.FindByID(ctx, fid)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}
	if folder == nil {
		return nil, apperr.NotFound("folder.not_found")
	}

	if folder.UserID != uid {
		return nil, apperr.NotFound("folder.not_found")
	}

	return folder, nil
}

func normalizePagination(pagination *dto.PaginationRequest) {
	if pagination.Page < 1 {
		pagination.Page = dto.DefaultPage
	}
	if pagination.PageSize < 1 {
		pagination.PageSize = dto.DefaultPageSize
	}
	if pagination.PageSize > dto.MaxPageSize {
		pagination.PageSize = dto.MaxPageSize
	}
}

func mapVocabEntityError(err error) error {
	switch {
	case errors.Is(err, domain.ErrWordRequired):
		return apperr.ValidationFailed("vocabulary.word_required")
	case errors.Is(err, domain.ErrMeaningRequired):
		return apperr.ValidationFailed("vocabulary.meaning_required")
	case errors.Is(err, domain.ErrInvalidLanguageID):
		return apperr.BadRequest("vocabulary.invalid_language_id")
	case errors.Is(err, domain.ErrInvalidProficiencyLevelID):
		return apperr.BadRequest("vocabulary.invalid_proficiency_level_id")
	default:
		return apperr.InternalServerError("common.internal_server_error", err)
	}
}
