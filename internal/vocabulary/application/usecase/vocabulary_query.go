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

func (useCase *VocabularyQuery) GetVocabulary(ctx context.Context, id string, _ string) (*vdto.VocabularyResponse, error) {
	vocabID, err := domain.ParseVocabularyID(id)
	if err != nil {
		return nil, apperr.BadRequest("vocabulary.invalid_id")
	}

	vocab, err := useCase.vocabRepo.FindByID(ctx, vocabID)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_error", err)
	}
	if vocab == nil {
		return nil, apperr.NotFound("vocabulary.not_found")
	}

	return mapper.ToVocabularyResponse(vocab), nil
}

func (useCase *VocabularyQuery) GetVocabularyDetail(ctx context.Context, id string, _ string) (*vdto.VocabularyDetailResponse, error) {
	vocabID, err := domain.ParseVocabularyID(id)
	if err != nil {
		return nil, apperr.BadRequest("vocabulary.invalid_id")
	}

	vocab, err := useCase.vocabRepo.FindByID(ctx, vocabID)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_error", err)
	}
	if vocab == nil {
		return nil, apperr.NotFound("vocabulary.not_found")
	}

	// Fetch related topics
	var topicResponses []vdto.TopicResponse
	topics, err := useCase.topicRepo.FindByVocabularyID(ctx, vocabID)
	if err != nil {
		logger.Warn(ctx, "[VOCABULARY] error fetching topics for vocabulary", zap.Error(err))
		topicResponses = []vdto.TopicResponse{}
	} else {
		topicResponses = make([]vdto.TopicResponse, 0, len(topics))
		for _, topic := range topics {
			topicResponses = append(topicResponses, mapper.ToTopicResponse(topic))
		}
	}

	// Fetch related grammar points
	var gpResponses []vdto.GrammarPointResponse
	grammarPoints, err := useCase.grammarRepo.FindByVocabularyID(ctx, vocabID)
	if err != nil {
		logger.Warn(ctx, "[VOCABULARY] error fetching grammar points for vocabulary", zap.Error(err))
		gpResponses = []vdto.GrammarPointResponse{}
	} else {
		gpResponses = make([]vdto.GrammarPointResponse, 0, len(grammarPoints))
		for _, grammarPoint := range grammarPoints {
			gpResponses = append(gpResponses, mapper.ToGrammarPointResponse(grammarPoint))
		}
	}

	return &vdto.VocabularyDetailResponse{
		VocabularyResponse: *mapper.ToVocabularyResponse(vocab),
		Topics:             topicResponses,
		GrammarPoints:      gpResponses,
	}, nil
}

func (useCase *VocabularyQuery) ListVocabularies(ctx context.Context, filter vdto.VocabularyFilter, pagination dto.PaginationRequest) (*dto.ListResult[*vdto.VocabularyListResponse], error) {
	normalizePagination(&pagination)
	offset := (pagination.Page - 1) * pagination.PageSize

	var langIDPtr *domain.LanguageID
	if filter.LanguageID != "" {
		parsed, err := domain.ParseLanguageID(filter.LanguageID)
		if err != nil {
			return nil, apperr.BadRequest("vocabulary.invalid_language_id")
		}
		langIDPtr = &parsed
	}

	var profLevelIDPtr *domain.LevelID
	if filter.LevelID != "" {
		parsed, err := domain.ParseLevelID(filter.LevelID)
		if err != nil {
			return nil, apperr.BadRequest("vocabulary.invalid_level_id")
		}
		profLevelIDPtr = &parsed
	}

	var topicIDPtr *domain.TopicID
	if filter.TopicID != "" {
		parsed, err := domain.ParseTopicID(filter.TopicID)
		if err != nil {
			return nil, apperr.BadRequest("vocabulary.invalid_topic_id")
		}
		topicIDPtr = &parsed
	}

	total, err := useCase.vocabRepo.CountAll(ctx, langIDPtr, profLevelIDPtr, topicIDPtr)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_error", err)
	}

	vocabs, err := useCase.vocabRepo.FindAll(ctx, langIDPtr, profLevelIDPtr, topicIDPtr, offset, pagination.PageSize)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_error", err)
	}

	return mapper.ToPaginatedListResult(vocabs, total, pagination), nil
}

func (useCase *VocabularyQuery) SearchVocabulary(ctx context.Context, query string, languageID string, _ string, pagination dto.PaginationRequest) (*dto.ListResult[*vdto.VocabularyListResponse], error) {
	normalizePagination(&pagination)
	offset := (pagination.Page - 1) * pagination.PageSize

	var langIDPtr *domain.LanguageID
	if languageID != "" {
		parsed, err := domain.ParseLanguageID(languageID)
		if err != nil {
			return nil, apperr.BadRequest("vocabulary.invalid_language_id")
		}
		langIDPtr = &parsed
	}

	total, err := useCase.vocabRepo.CountSearch(ctx, query, langIDPtr)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_error", err)
	}

	vocabs, err := useCase.vocabRepo.Search(ctx, query, langIDPtr, offset, pagination.PageSize)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_error", err)
	}

	return mapper.ToPaginatedListResult(vocabs, total, pagination), nil
}
