package usecase

import (
	"context"

	apperr "learning-go/internal/shared/error"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"
)

type CategoryQuery struct {
	categoryRepo port.CategoryRepositoryPort
}

func NewCategoryQuery(categoryRepo port.CategoryRepositoryPort) port.CategoryQueryPort {
	return &CategoryQuery{categoryRepo: categoryRepo}
}

func (useCase *CategoryQuery) ListCategories(ctx context.Context, languageID string, isPublic *bool) ([]*vdto.CategoryResponse, error) {
	var langIDPtr *domain.LanguageID
	if languageID != "" {
		parsed, err := domain.ParseLanguageID(languageID)
		if err != nil {
			return nil, apperr.BadRequest("category.invalid_language_id")
		}
		langIDPtr = &parsed
	}

	categories, err := useCase.categoryRepo.FindAll(ctx, langIDPtr, isPublic)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	result := make([]*vdto.CategoryResponse, 0, len(categories))
	for _, cat := range categories {
		result = append(result, mapper.ToCategoryResponse(cat))
	}
	return result, nil
}

func (useCase *CategoryQuery) GetCategory(ctx context.Context, id string) (*vdto.CategoryResponse, error) {
	catID, err := domain.ParseCategoryID(id)
	if err != nil {
		return nil, apperr.BadRequest("category.invalid_id")
	}

	cat, err := useCase.categoryRepo.FindByID(ctx, catID)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}
	if cat == nil {
		return nil, apperr.NotFound("category.not_found")
	}

	return mapper.ToCategoryResponse(cat), nil
}
