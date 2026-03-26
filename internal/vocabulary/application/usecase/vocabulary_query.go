package usecase

import (
	"context"
	"learning-go/internal/shared/dto"
	apperr "learning-go/internal/shared/error"
	"learning-go/internal/shared/logger"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type VocabularyQuery struct {
	vocabRepo   port.VocabularyRepositoryPort
	topicRepo   port.TopicRepositoryPort
	grammarRepo port.GrammarPointRepositoryPort
}

func NewVocabularyQuery(
	vocabRepo port.VocabularyRepositoryPort,
	topicRepo port.TopicRepositoryPort,
	grammarRepo port.GrammarPointRepositoryPort,
) port.VocabularyQueryPort {
	return &VocabularyQuery{
		vocabRepo:   vocabRepo,
		topicRepo:   topicRepo,
		grammarRepo: grammarRepo,
	}
}

func (useCase *VocabularyQuery) GetVocabulary(ctx context.Context, id string) (*vdto.VocabularyResponse, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return nil, apperr.BadRequest("vocabulary.invalid_id")
	}

	vocab, err := useCase.vocabRepo.FindByID(ctx, uuidID)
	if err != nil {
		return nil, apperr.InternalServerError("vocabulary.query_failed", err)
	}
	if vocab == nil {
		return nil, apperr.NotFound("vocabulary.not_found")
	}

	return mapper.ToVocabularyResponse(vocab), nil
}

func (useCase *VocabularyQuery) GetVocabularyDetail(ctx context.Context, id string) (*vdto.VocabularyDetailResponse, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return nil, apperr.BadRequest("vocabulary.invalid_id")
	}

	vocab, err := useCase.vocabRepo.FindByID(ctx, uuidID)
	if err != nil {
		return nil, apperr.InternalServerError("vocabulary.query_failed", err)
	}
	if vocab == nil {
		return nil, apperr.NotFound("vocabulary.not_found")
	}

	// Fetch related topics
	var topicResponses []vdto.TopicResponse
	topics, err := useCase.findVocabTopics(ctx, uuidID)
	if err != nil {
		logger.Warn(ctx, "[VOCABULARY] error fetching topics for vocabulary", zap.Error(err))
		topicResponses = []vdto.TopicResponse{}
	} else {
		topicResponses = make([]vdto.TopicResponse, 0, len(topics))
		for _, t := range topics {
			topicResponses = append(topicResponses, mapper.ToTopicResponse(t))
		}
	}

	// Fetch related grammar points
	var gpResponses []vdto.GrammarPointResponse
	grammarPoints, err := useCase.grammarRepo.FindByVocabularyID(ctx, uuidID)
	if err != nil {
		logger.Warn(ctx, "[VOCABULARY] error fetching grammar points for vocabulary", zap.Error(err))
		gpResponses = []vdto.GrammarPointResponse{}
	} else {
		gpResponses = make([]vdto.GrammarPointResponse, 0, len(grammarPoints))
		for _, gp := range grammarPoints {
			gpResponses = append(gpResponses, mapper.ToGrammarPointResponse(gp))
		}
	}

	return &vdto.VocabularyDetailResponse{
		VocabularyResponse: *mapper.ToVocabularyResponse(vocab),
		Topics:             topicResponses,
		GrammarPoints:      gpResponses,
	}, nil
}

func (useCase *VocabularyQuery) findVocabTopics(ctx context.Context, vocabID uuid.UUID) ([]*domain.Topic, error) {
	return useCase.topicRepo.FindByVocabularyID(ctx, vocabID)
}

func (useCase *VocabularyQuery) ListByHSKLevel(ctx context.Context, level int, pagination dto.PaginationRequest) (*dto.PaginatedResponse, error) {
	normalizePagination(&pagination)
	offset := (pagination.Page - 1) * pagination.PageSize

	total, err := useCase.vocabRepo.CountByHSKLevel(ctx, level)
	if err != nil {
		return nil, apperr.InternalServerError("vocabulary.query_failed", err)
	}

	vocabs, err := useCase.vocabRepo.FindByHSKLevel(ctx, level, offset, pagination.PageSize)
	if err != nil {
		return nil, apperr.InternalServerError("vocabulary.query_failed", err)
	}

	return mapper.ToPaginatedResponse(vocabs, total, pagination), nil
}

func (useCase *VocabularyQuery) ListByTopic(ctx context.Context, slug string, pagination dto.PaginationRequest) (*dto.PaginatedResponse, error) {
	topic, err := useCase.topicRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, apperr.InternalServerError("topic.query_failed", err)
	}
	if topic == nil {
		return nil, apperr.NotFound("topic.not_found")
	}

	normalizePagination(&pagination)
	offset := (pagination.Page - 1) * pagination.PageSize

	total, err := useCase.vocabRepo.CountByTopicID(ctx, topic.ID)
	if err != nil {
		return nil, apperr.InternalServerError("vocabulary.query_failed", err)
	}

	vocabs, err := useCase.vocabRepo.FindByTopicID(ctx, topic.ID, offset, pagination.PageSize)
	if err != nil {
		return nil, apperr.InternalServerError("vocabulary.query_failed", err)
	}

	return mapper.ToPaginatedResponse(vocabs, total, pagination), nil
}

func (useCase *VocabularyQuery) SearchVocabulary(ctx context.Context, query string, pagination dto.PaginationRequest) (*dto.PaginatedResponse, error) {
	normalizePagination(&pagination)
	offset := (pagination.Page - 1) * pagination.PageSize

	total, err := useCase.vocabRepo.CountSearch(ctx, query)
	if err != nil {
		return nil, apperr.InternalServerError("vocabulary.query_failed", err)
	}

	vocabs, err := useCase.vocabRepo.Search(ctx, query, offset, pagination.PageSize)
	if err != nil {
		return nil, apperr.InternalServerError("vocabulary.query_failed", err)
	}

	return mapper.ToPaginatedResponse(vocabs, total, pagination), nil
}

func normalizePagination(p *dto.PaginationRequest) {
	if p.Page < 1 {
		p.Page = dto.DefaultPage
	}
	if p.PageSize < 1 {
		p.PageSize = dto.DefaultPageSize
	}
	if p.PageSize > dto.MaxPageSize {
		p.PageSize = dto.MaxPageSize
	}
}
