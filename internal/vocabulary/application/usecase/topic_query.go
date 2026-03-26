package usecase

import (
	"context"
	apperr "learning-go/internal/shared/error"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
)

type TopicQuery struct {
	topicRepo port.TopicRepositoryPort
}

func NewTopicQuery(topicRepo port.TopicRepositoryPort) port.TopicQueryPort {
	return &TopicQuery{topicRepo: topicRepo}
}

func (useCase *TopicQuery) ListTopics(ctx context.Context) ([]*vdto.TopicResponse, error) {
	topics, err := useCase.topicRepo.FindAll(ctx)
	if err != nil {
		return nil, apperr.InternalServerError("topic.query_failed", err)
	}

	result := make([]*vdto.TopicResponse, 0, len(topics))
	for _, t := range topics {
		resp := mapper.ToTopicResponse(t)
		result = append(result, &resp)
	}
	return result, nil
}
