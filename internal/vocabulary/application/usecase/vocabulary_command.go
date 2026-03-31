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
	params, err := mapper.ToVocabularyParams(req.LanguageID, req.ProficiencyLevelID, req.Word, req.Phonetic, req.AudioURL, req.ImageURL, req.FrequencyRank, req.Metadata, req.Meanings)
	if err != nil {
		return nil, mapVocabEntityError(err)
	}

	vocab, err := domain.NewVocabularyFromParams(params)
	if err != nil {
		return nil, mapVocabEntityError(err)
	}

	if err := useCase.vocabRepo.Save(ctx, vocab); err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	return mapper.ToVocabularyResponse(vocab), nil
}

func (useCase *VocabularyCommand) UpdateVocabulary(ctx context.Context, id string, req vdto.UpdateVocabularyRequest) (*vdto.VocabularyResponse, error) {
	vocabID, err := domain.ParseVocabularyID(id)
	if err != nil {
		return nil, apperr.BadRequest("vocabulary.invalid_id")
	}

	existing, err := useCase.vocabRepo.FindByID(ctx, vocabID)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	if existing == nil {
		return nil, apperr.NotFound("vocabulary.not_found")
	}

	params, err := mapper.ToVocabularyParams(req.LanguageID, req.ProficiencyLevelID, req.Word, req.Phonetic, req.AudioURL, req.ImageURL, req.FrequencyRank, req.Metadata, req.Meanings)
	if err != nil {
		return nil, mapVocabEntityError(err)
	}

	if err := existing.Update(params); err != nil {
		return nil, mapVocabEntityError(err)
	}

	if err := useCase.vocabRepo.Update(ctx, existing); err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	// Set topics if provided
	if req.TopicIDs != nil {
		topicIDs, parseErr := mapper.ParseTopicIDs(req.TopicIDs)
		if parseErr != nil {
			return nil, apperr.BadRequest("vocabulary.invalid_topic_id")
		}

		found, findErr := useCase.topicRepo.FindByIDs(ctx, topicIDs)
		if findErr != nil {
			return nil, apperr.InternalServerError("common.internal_server_error", findErr)
		}

		if len(found) != len(topicIDs) {
			return nil, apperr.BadRequest("vocabulary.invalid_topic_id")
		}

		existing.SetTopics(topicIDs)
		if err := useCase.vocabRepo.SetTopics(ctx, existing.ID, topicIDs); err != nil {
			return nil, apperr.InternalServerError("common.internal_server_error", err)
		}
	}

	// Set grammar points if provided
	if req.GrammarPointIDs != nil {
		gpIDs, parseErr := mapper.ParseGrammarPointIDs(req.GrammarPointIDs)
		if parseErr != nil {
			return nil, apperr.BadRequest("vocabulary.invalid_grammar_point_id")
		}
		found, findErr := useCase.grammarRepo.FindByIDs(ctx, gpIDs)
		if findErr != nil {
			return nil, apperr.InternalServerError("common.internal_server_error", findErr)
		}
		if len(found) != len(gpIDs) {
			return nil, apperr.BadRequest("vocabulary.invalid_grammar_point_id")
		}
		existing.SetGrammarPoints(gpIDs)
		if err := useCase.vocabRepo.SetGrammarPoints(ctx, existing.ID, gpIDs); err != nil {
			return nil, apperr.InternalServerError("common.internal_server_error", err)
		}
	}

	return mapper.ToVocabularyResponse(existing), nil
}

func (useCase *VocabularyCommand) DeleteVocabulary(ctx context.Context, id string) error {
	vocabID, err := domain.ParseVocabularyID(id)
	if err != nil {
		return apperr.BadRequest("vocabulary.invalid_id")
	}

	vocab, err := useCase.vocabRepo.FindByID(ctx, vocabID)
	if err != nil {
		return apperr.InternalServerError("common.internal_server_error", err)
	}

	if vocab == nil {
		return apperr.NotFound("vocabulary.not_found")
	}

	if err := useCase.vocabRepo.Delete(ctx, vocabID); err != nil {
		return apperr.InternalServerError("common.internal_server_error", err)
	}

	return nil
}

func (useCase *VocabularyCommand) SetTopics(ctx context.Context, id string, topicIDs []string) error {
	vocabID, err := domain.ParseVocabularyID(id)
	if err != nil {
		return apperr.BadRequest("vocabulary.invalid_id")
	}

	vocab, err := useCase.vocabRepo.FindByID(ctx, vocabID)
	if err != nil {
		return apperr.InternalServerError("common.internal_server_error", err)
	}

	if vocab == nil {
		return apperr.NotFound("vocabulary.not_found")
	}

	parsed, err := mapper.ParseTopicIDs(topicIDs)
	if err != nil {
		return apperr.BadRequest("vocabulary.invalid_topic_id")
	}

	if len(parsed) > 0 {
		found, findErr := useCase.topicRepo.FindByIDs(ctx, parsed)
		if findErr != nil {
			return apperr.InternalServerError("common.internal_server_error", findErr)
		}

		if len(found) != len(parsed) {
			return apperr.BadRequest("vocabulary.invalid_topic_id")
		}
	}

	if err := useCase.vocabRepo.SetTopics(ctx, vocabID, parsed); err != nil {
		return apperr.InternalServerError("common.internal_server_error", err)
	}

	return nil
}

func (useCase *VocabularyCommand) SetGrammarPoints(ctx context.Context, id string, grammarPointIDs []string) error {
	vocabID, err := domain.ParseVocabularyID(id)
	if err != nil {
		return apperr.BadRequest("vocabulary.invalid_id")
	}

	vocab, err := useCase.vocabRepo.FindByID(ctx, vocabID)
	if err != nil {
		return apperr.InternalServerError("common.internal_server_error", err)
	}

	if vocab == nil {
		return apperr.NotFound("vocabulary.not_found")
	}

	parsed, err := mapper.ParseGrammarPointIDs(grammarPointIDs)
	if err != nil {
		return apperr.BadRequest("vocabulary.invalid_grammar_point_id")
	}

	if len(parsed) > 0 {
		found, findErr := useCase.grammarRepo.FindByIDs(ctx, parsed)
		if findErr != nil {
			return apperr.InternalServerError("common.internal_server_error", findErr)
		}

		if len(found) != len(parsed) {
			return apperr.BadRequest("vocabulary.invalid_grammar_point_id")
		}
	}

	if err := useCase.vocabRepo.SetGrammarPoints(ctx, vocabID, parsed); err != nil {
		return apperr.InternalServerError("common.internal_server_error", err)
	}

	return nil
}
