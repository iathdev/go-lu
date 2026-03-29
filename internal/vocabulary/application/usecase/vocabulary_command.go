package usecase

import (
	"context"

	apperr "learning-go/internal/shared/error"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"
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
	vocabID, err := domain.ParseVocabularyID(id)
	if err != nil {
		return nil, apperr.BadRequest("vocabulary.invalid_id")
	}

	vocab, err := useCase.vocabRepo.FindByID(ctx, vocabID)
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
		topicIDs, parseErr := parseTopicIDs(req.TopicIDs)
		if parseErr != nil {
			return nil, apperr.BadRequest("vocabulary.invalid_topic_id")
		}
		found, err := useCase.topicRepo.FindByIDs(ctx, topicIDs)
		if err != nil {
			return nil, apperr.InternalServerError("topic.query_failed", err)
		}
		if len(found) != len(topicIDs) {
			return nil, apperr.BadRequest("vocabulary.invalid_topic_id")
		}
		if err := useCase.vocabRepo.SetTopics(ctx, vocabID, topicIDs); err != nil {
			return nil, apperr.InternalServerError("vocabulary.set_topics_failed", err)
		}
	}

	// Set grammar points if provided
	if req.GrammarPointIDs != nil {
		gpIDs, parseErr := parseGrammarPointIDs(req.GrammarPointIDs)
		if parseErr != nil {
			return nil, apperr.BadRequest("vocabulary.invalid_grammar_point_id")
		}
		found, err := useCase.grammarRepo.FindByIDs(ctx, gpIDs)
		if err != nil {
			return nil, apperr.InternalServerError("grammar_point.query_failed", err)
		}
		if len(found) != len(gpIDs) {
			return nil, apperr.BadRequest("vocabulary.invalid_grammar_point_id")
		}
		if err := useCase.vocabRepo.SetGrammarPoints(ctx, vocabID, gpIDs); err != nil {
			return nil, apperr.InternalServerError("vocabulary.set_grammar_points_failed", err)
		}
	}

	return mapper.ToVocabularyResponse(vocab), nil
}

func (useCase *VocabularyCommand) DeleteVocabulary(ctx context.Context, id string) error {
	vocabID, err := domain.ParseVocabularyID(id)
	if err != nil {
		return apperr.BadRequest("vocabulary.invalid_id")
	}

	vocab, err := useCase.vocabRepo.FindByID(ctx, vocabID)
	if err != nil {
		return apperr.InternalServerError("vocabulary.query_failed", err)
	}
	if vocab == nil {
		return apperr.NotFound("vocabulary.not_found")
	}

	if err := useCase.vocabRepo.Delete(ctx, vocabID); err != nil {
		return apperr.InternalServerError("vocabulary.delete_failed", err)
	}

	return nil
}
