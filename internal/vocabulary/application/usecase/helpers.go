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

func parseTopicIDs(ids []string) ([]domain.TopicID, error) {
	result := make([]domain.TopicID, 0, len(ids))
	for _, id := range ids {
		parsed, err := domain.ParseTopicID(id)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}

func parseGrammarPointIDs(ids []string) ([]domain.GrammarPointID, error) {
	result := make([]domain.GrammarPointID, 0, len(ids))
	for _, id := range ids {
		parsed, err := domain.ParseGrammarPointID(id)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}

func mapVocabEntityError(err error) error {
	switch {
	case errors.Is(err, domain.ErrHanziRequired):
		return apperr.UnprocessableEntity("vocabulary.hanzi_required")
	case errors.Is(err, domain.ErrPinyinRequired):
		return apperr.UnprocessableEntity("vocabulary.pinyin_required")
	case errors.Is(err, domain.ErrMeaningRequired):
		return apperr.UnprocessableEntity("vocabulary.meaning_required")
	case errors.Is(err, domain.ErrInvalidHSKLevel):
		return apperr.UnprocessableEntity("vocabulary.invalid_hsk_level")
	default:
		return apperr.InternalServerError("common.internal_server_error", err)
	}
}
