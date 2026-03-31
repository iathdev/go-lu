package usecase

import (
	"context"

	apperr "learning-go/internal/shared/error"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"
)

type LanguageQuery struct {
	languageRepo port.LanguageRepositoryPort
}

func NewLanguageQuery(languageRepo port.LanguageRepositoryPort) port.LanguageQueryPort {
	return &LanguageQuery{languageRepo: languageRepo}
}

func (useCase *LanguageQuery) ListLanguages(ctx context.Context, activeOnly bool) ([]*vdto.LanguageResponse, error) {
	languages, err := useCase.languageRepo.FindAll(ctx, activeOnly)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	result := make([]*vdto.LanguageResponse, 0, len(languages))
	for _, lang := range languages {
		result = append(result, mapper.ToLanguageResponse(lang))
	}
	return result, nil
}

func (useCase *LanguageQuery) GetLanguage(ctx context.Context, id string) (*vdto.LanguageResponse, error) {
	langID, err := domain.ParseLanguageID(id)
	if err != nil {
		return nil, apperr.BadRequest("language.invalid_id")
	}

	lang, err := useCase.languageRepo.FindByID(ctx, langID)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}
	if lang == nil {
		return nil, apperr.NotFound("language.not_found")
	}

	return mapper.ToLanguageResponse(lang), nil
}
