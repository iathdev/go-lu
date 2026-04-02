package usecase

import (
	"context"

	apperr "learning-go/internal/shared/error"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"
)

type ProficiencyLevelQuery struct {
	profLevelRepo port.ProficiencyLevelRepositoryPort
}

func NewProficiencyLevelQuery(profLevelRepo port.ProficiencyLevelRepositoryPort) port.ProficiencyLevelQueryPort {
	return &ProficiencyLevelQuery{profLevelRepo: profLevelRepo}
}

func (useCase *ProficiencyLevelQuery) ListProficiencyLevels(ctx context.Context, categoryID string) ([]*vdto.ProficiencyLevelResponse, error) {
	var catIDPtr *domain.CategoryID
	if categoryID != "" {
		parsed, err := domain.ParseCategoryID(categoryID)
		if err != nil {
			return nil, apperr.BadRequest("proficiency_level.invalid_category_id")
		}
		catIDPtr = &parsed
	}

	levels, err := useCase.profLevelRepo.FindAll(ctx, catIDPtr)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_error", err)
	}

	result := make([]*vdto.ProficiencyLevelResponse, 0, len(levels))
	for _, level := range levels {
		result = append(result, mapper.ToProficiencyLevelResponse(level))
	}
	return result, nil
}

func (useCase *ProficiencyLevelQuery) GetProficiencyLevel(ctx context.Context, id string) (*vdto.ProficiencyLevelResponse, error) {
	plID, err := domain.ParseLevelID(id)
	if err != nil {
		return nil, apperr.BadRequest("proficiency_level.invalid_id")
	}

	level, err := useCase.profLevelRepo.FindByID(ctx, plID)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_error", err)
	}
	if level == nil {
		return nil, apperr.NotFound("proficiency_level.not_found")
	}

	return mapper.ToProficiencyLevelResponse(level), nil
}
