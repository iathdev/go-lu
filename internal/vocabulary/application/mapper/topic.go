package mapper

import (
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/domain"
)

func ToTopicResponse(topic *domain.Topic) vdto.TopicResponse {
	return vdto.TopicResponse{
		ID:     topic.ID.String(),
		NameCN: topic.NameCN,
		NameVI: topic.NameVI,
		NameEN: topic.NameEN,
		Slug:   topic.Slug,
	}
}

func ToGrammarPointResponse(gp *domain.GrammarPoint) vdto.GrammarPointResponse {
	return vdto.GrammarPointResponse{
		ID:            gp.ID.String(),
		Code:          gp.Code,
		Pattern:       gp.Pattern,
		ExampleCN:     gp.ExampleCN,
		ExampleVI:     gp.ExampleVI,
		Rule:          gp.Rule,
		CommonMistake: gp.CommonMistake,
		HSKLevel:      gp.HSKLevel.Int(),
	}
}
