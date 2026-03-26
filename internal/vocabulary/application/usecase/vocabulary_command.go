package usecase

import (
	"context"
	"errors"
	apperr "learning-go/internal/shared/error"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"

	"github.com/google/uuid"
)

type VocabularyCommand struct {
	vocabRepo   port.VocabularyRepositoryPort
	topicRepo   port.TopicRepositoryPort
	grammarRepo port.GrammarPointRepositoryPort
}

func NewVocabularyCommand(
	vocabRepo port.VocabularyRepositoryPort,
	topicRepo port.TopicRepositoryPort,
	grammarRepo port.GrammarPointRepositoryPort,
) port.VocabularyCommandPort {
	return &VocabularyCommand{vocabRepo: vocabRepo, topicRepo: topicRepo, grammarRepo: grammarRepo}
}

func (useCase *VocabularyCommand) CreateVocabulary(ctx context.Context, req vdto.CreateVocabularyRequest) (*vdto.VocabularyResponse, error) {
	params := domain.VocabularyParams{
		Hanzi:           req.Hanzi,
		Pinyin:          req.Pinyin,
		MeaningVI:       req.MeaningVI,
		MeaningEN:       req.MeaningEN,
		HSKLevel:        req.HSKLevel,
		AudioURL:        req.AudioURL,
		Examples:        mapper.ToExampleEntities(req.Examples),
		Radicals:        req.Radicals,
		StrokeCount:     req.StrokeCount,
		StrokeDataURL:   req.StrokeDataURL,
		RecognitionOnly: req.RecognitionOnly,
		FrequencyRank:   req.FrequencyRank,
	}

	vocab, err := domain.NewVocabularyFromParams(params)
	if err != nil {
		return nil, mapVocabEntityError(err)
	}

	if err := useCase.vocabRepo.Save(ctx, vocab); err != nil {
		return nil, apperr.InternalServerError("vocabulary.save_failed", err)
	}

	return mapper.ToVocabularyResponse(vocab), nil
}

func (useCase *VocabularyCommand) UpdateVocabulary(ctx context.Context, id string, req vdto.UpdateVocabularyRequest) (*vdto.VocabularyResponse, error) {
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

	params := domain.VocabularyParams{
		Hanzi:           req.Hanzi,
		Pinyin:          req.Pinyin,
		MeaningVI:       req.MeaningVI,
		MeaningEN:       req.MeaningEN,
		HSKLevel:        req.HSKLevel,
		AudioURL:        req.AudioURL,
		Examples:        mapper.ToExampleEntities(req.Examples),
		Radicals:        req.Radicals,
		StrokeCount:     req.StrokeCount,
		StrokeDataURL:   req.StrokeDataURL,
		RecognitionOnly: req.RecognitionOnly,
		FrequencyRank:   req.FrequencyRank,
	}

	if err := vocab.UpdateFromParams(params); err != nil {
		return nil, mapVocabEntityError(err)
	}

	if err := useCase.vocabRepo.Update(ctx, vocab); err != nil {
		return nil, apperr.InternalServerError("vocabulary.update_failed", err)
	}

	// Set topics if provided
	if req.TopicIDs != nil {
		topicUUIDs, parseErr := parseUUIDs(req.TopicIDs)
		if parseErr != nil {
			return nil, apperr.BadRequest("vocabulary.invalid_topic_id")
		}
		found, err := useCase.topicRepo.FindByIDs(ctx, topicUUIDs)
		if err != nil {
			return nil, apperr.InternalServerError("topic.query_failed", err)
		}
		if len(found) != len(topicUUIDs) {
			return nil, apperr.BadRequest("vocabulary.invalid_topic_id")
		}
		if err := useCase.vocabRepo.SetTopics(ctx, uuidID, topicUUIDs); err != nil {
			return nil, apperr.InternalServerError("vocabulary.set_topics_failed", err)
		}
	}

	// Set grammar points if provided
	if req.GrammarPointIDs != nil {
		gpUUIDs, parseErr := parseUUIDs(req.GrammarPointIDs)
		if parseErr != nil {
			return nil, apperr.BadRequest("vocabulary.invalid_grammar_point_id")
		}
		found, err := useCase.grammarRepo.FindByIDs(ctx, gpUUIDs)
		if err != nil {
			return nil, apperr.InternalServerError("grammar_point.query_failed", err)
		}
		if len(found) != len(gpUUIDs) {
			return nil, apperr.BadRequest("vocabulary.invalid_grammar_point_id")
		}
		if err := useCase.vocabRepo.SetGrammarPoints(ctx, uuidID, gpUUIDs); err != nil {
			return nil, apperr.InternalServerError("vocabulary.set_grammar_points_failed", err)
		}
	}

	return mapper.ToVocabularyResponse(vocab), nil
}

func (useCase *VocabularyCommand) DeleteVocabulary(ctx context.Context, id string) error {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return apperr.BadRequest("vocabulary.invalid_id")
	}

	vocab, err := useCase.vocabRepo.FindByID(ctx, uuidID)
	if err != nil {
		return apperr.InternalServerError("vocabulary.query_failed", err)
	}
	if vocab == nil {
		return apperr.NotFound("vocabulary.not_found")
	}

	if err := useCase.vocabRepo.Delete(ctx, uuidID); err != nil {
		return apperr.InternalServerError("vocabulary.delete_failed", err)
	}

	return nil
}

func parseUUIDs(ids []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		u, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, nil
}

func mapVocabEntityError(err error) error {
	switch {
	case errors.Is(err, domain.ErrHanziRequired):
		return apperr.UnprocessableEntity("vocabulary.hanzi_required")
	case errors.Is(err, domain.ErrPinyinRequired):
		return apperr.UnprocessableEntity("vocabulary.pinyin_required")
	case errors.Is(err, domain.ErrMeaningRequired):
		return apperr.UnprocessableEntity("vocabulary.meaning_required")
	case errors.Is(err, domain.ErrInvalidHSKLevel):
		return apperr.UnprocessableEntity("vocabulary.invalid_hsk_level")
	default:
		return apperr.InternalServerError("common.internal_server_error", err)
	}
}
