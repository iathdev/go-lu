package usecase

import (
	"context"

	apperr "learning-go/internal/shared/error"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"
)

type TopicQuery struct {
	topicRepo port.TopicRepositoryPort
}

func NewTopicQuery(topicRepo port.TopicRepositoryPort) port.TopicQueryPort {
	return &TopicQuery{topicRepo: topicRepo}
}

func (useCase *TopicQuery) ListTopics(ctx context.Context, categoryID string) ([]*vdto.TopicResponse, error) {
	var catIDPtr *domain.CategoryID
	if categoryID != "" {
		parsed, err := domain.ParseCategoryID(categoryID)
		if err != nil {
			return nil, apperr.BadRequest("topic.invalid_category_id")
		}
		catIDPtr = &parsed
	}

	topics, err := useCase.topicRepo.FindAll(ctx, catIDPtr)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	result := make([]*vdto.TopicResponse, 0, len(topics))
	for _, topic := range topics {
		resp := mapper.ToTopicResponse(topic)
		result = append(result, &resp)
	}
	return result, nil
}

func (useCase *TopicQuery) GetTopic(ctx context.Context, id string) (*vdto.TopicResponse, error) {
	topicID, err := domain.ParseTopicID(id)
	if err != nil {
		return nil, apperr.BadRequest("topic.invalid_id")
	}

	topic, err := useCase.topicRepo.FindByID(ctx, topicID)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}
	if topic == nil {
		return nil, apperr.NotFound("topic.not_found")
	}

	resp := mapper.ToTopicResponse(topic)
	return &resp, nil
}
