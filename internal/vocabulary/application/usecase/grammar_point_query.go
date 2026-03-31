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

type GrammarPointQuery struct {
	grammarRepo port.GrammarPointRepositoryPort
}

func NewGrammarPointQuery(grammarRepo port.GrammarPointRepositoryPort) port.GrammarPointQueryPort {
	return &GrammarPointQuery{grammarRepo: grammarRepo}
}

func (useCase *GrammarPointQuery) ListGrammarPoints(ctx context.Context, categoryID string, proficiencyLevelID string, pagination dto.PaginationRequest) (*dto.ListResult[*vdto.GrammarPointResponse], error) {
	normalizePagination(&pagination)
	offset := (pagination.Page - 1) * pagination.PageSize

	var catIDPtr *domain.CategoryID
	if categoryID != "" {
		parsed, err := domain.ParseCategoryID(categoryID)
		if err != nil {
			return nil, apperr.BadRequest("grammar_point.invalid_category_id")
		}
		catIDPtr = &parsed
	}

	var profLevelIDPtr *domain.ProficiencyLevelID
	if proficiencyLevelID != "" {
		parsed, err := domain.ParseProficiencyLevelID(proficiencyLevelID)
		if err != nil {
			return nil, apperr.BadRequest("grammar_point.invalid_proficiency_level_id")
		}
		profLevelIDPtr = &parsed
	}

	total, err := useCase.grammarRepo.CountAll(ctx, catIDPtr, profLevelIDPtr)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	grammarPoints, err := useCase.grammarRepo.FindAll(ctx, catIDPtr, profLevelIDPtr, offset, pagination.PageSize)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	items := make([]*vdto.GrammarPointResponse, 0, len(grammarPoints))
	for _, grammarPoint := range grammarPoints {
		resp := mapper.ToGrammarPointResponse(grammarPoint)
		items = append(items, &resp)
	}

	totalPages := 0
	if pagination.PageSize > 0 {
		totalPages = int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize))
	}

	return &dto.ListResult[*vdto.GrammarPointResponse]{
		Items:      items,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (useCase *GrammarPointQuery) GetGrammarPoint(ctx context.Context, id string) (*vdto.GrammarPointResponse, error) {
	gpID, err := domain.ParseGrammarPointID(id)
	if err != nil {
		return nil, apperr.BadRequest("grammar_point.invalid_id")
	}

	grammarPoint, err := useCase.grammarRepo.FindByID(ctx, gpID)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}
	if grammarPoint == nil {
		return nil, apperr.NotFound("grammar_point.not_found")
	}

	resp := mapper.ToGrammarPointResponse(grammarPoint)
	return &resp, nil
}
