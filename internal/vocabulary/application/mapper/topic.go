package mapper

import (
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/domain"
)

// ToTopicResponse maps domain.Topic to TopicResponse.
func ToTopicResponse(topic *domain.Topic) vdto.TopicResponse {
	return vdto.TopicResponse{
		ID:         topic.ID.String(),
		CategoryID: topic.CategoryID.String(),
		Slug:       topic.Slug,
		Names:      topic.Names,
		Offset:     topic.Offset,
	}
}

// ToGrammarPointResponse maps domain.GrammarPoint to GrammarPointResponse.
func ToGrammarPointResponse(grammarPoint *domain.GrammarPoint) vdto.GrammarPointResponse {
	return vdto.GrammarPointResponse{
		ID:                 grammarPoint.ID.String(),
		CategoryID:         grammarPoint.CategoryID.String(),
		LevelID:            grammarPoint.LevelID.String(),
		Code:               grammarPoint.Code,
		Pattern:            grammarPoint.Pattern,
		Examples:           grammarPoint.Examples,
		Rule:               grammarPoint.Rule,
		CommonMistakes:     grammarPoint.CommonMistakes,
	}
}
