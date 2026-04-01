package mapper

import (
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/domain"
)

// ToLanguageResponse maps domain.Language to LanguageResponse.
func ToLanguageResponse(lang *domain.Language) *vdto.LanguageResponse {
	return &vdto.LanguageResponse{
		ID:         lang.ID.String(),
		Code:       lang.Code,
		NameEN:     lang.NameEN,
		NameNative: lang.NameNative,
		IsActive:   lang.IsActive,
		Config:     lang.Config,
	}
}

// ToCategoryResponse maps domain.Category to CategoryResponse.
func ToCategoryResponse(cat *domain.Category) *vdto.CategoryResponse {
	return &vdto.CategoryResponse{
		ID:             cat.ID.String(),
		LanguageID:     cat.LanguageID.String(),
		PrepCategoryID: cat.PrepCategoryID,
		Code:           cat.Code,
		Name:           cat.Name,
		IsPublic:       cat.IsPublic,
	}
}

// ToProficiencyLevelResponse maps domain.ProficiencyLevel to ProficiencyLevelResponse.
func ToProficiencyLevelResponse(profLevel *domain.ProficiencyLevel) *vdto.ProficiencyLevelResponse {
	return &vdto.ProficiencyLevelResponse{
		ID:            profLevel.ID.String(),
		CategoryID:    profLevel.CategoryID.String(),
		PrepLevelID:   profLevel.PrepLevelID,
		Code:          profLevel.Code,
		Name:          profLevel.Name,
		Target:        profLevel.Target,
		DisplayTarget: profLevel.DisplayTarget,
		Offset:        profLevel.Offset,
	}
}
