package usecase

import (
	"context"

	apperr "learning-go/internal/shared/error"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"
)

type ImportCommand struct {
	vocabRepo port.VocabularyRepositoryPort
}

func NewImportCommand(vocabRepo port.VocabularyRepositoryPort) port.ImportCommandPort {
	return &ImportCommand{vocabRepo: vocabRepo}
}

func (useCase *ImportCommand) ImportVocabularies(ctx context.Context, req vdto.BulkImportRequest) (*vdto.BulkImportResponse, error) {
	if len(req.Vocabularies) == 0 {
		return &vdto.BulkImportResponse{Total: 0}, nil
	}

	// Group vocabularies by language to check duplicates per language
	type langGroup struct {
		languageID domain.LanguageID
		words      []string
		items      []vdto.CreateVocabularyRequest
	}

	groupMap := make(map[string]*langGroup)
	for _, item := range req.Vocabularies {
		langID, err := domain.ParseLanguageID(item.LanguageID)
		if err != nil {
			continue // skip items with invalid language ID
		}
		key := langID.String()
		group, ok := groupMap[key]
		if !ok {
			group = &langGroup{languageID: langID}
			groupMap[key] = group
		}
		group.words = append(group.words, item.Word)
		group.items = append(group.items, item)
	}

	// Build set of existing words per language
	existingSet := make(map[string]map[string]bool)
	for key, group := range groupMap {
		existing, err := useCase.vocabRepo.FindByWordList(ctx, group.languageID, group.words)
		if err != nil {
			return nil, apperr.InternalServerError("common.internal_server_error", err)
		}
		wordSet := make(map[string]bool, len(existing))
		for _, vocab := range existing {
			wordSet[vocab.Word] = true
		}
		existingSet[key] = wordSet
	}

	// Filter duplicates and create domain entities
	var newVocabs []*domain.Vocabulary
	skipped := 0
	for key, group := range groupMap {
		wordSet := existingSet[key]
		for _, item := range group.items {
			if wordSet[item.Word] {
				skipped++
				continue
			}

			params, parseErr := mapper.ToVocabularyParams(
				item.LanguageID, item.ProficiencyLevelID, item.Word,
				item.Phonetic, item.AudioURL, item.ImageURL,
				item.FrequencyRank, item.Metadata, item.Meanings,
			)
			if parseErr != nil {
				skipped++
				continue
			}

			vocab, err := domain.NewVocabularyFromParams(params)
			if err != nil {
				skipped++
				continue
			}
			newVocabs = append(newVocabs, vocab)
		}
	}

	imported := 0
	if len(newVocabs) > 0 {
		var err error
		imported, err = useCase.vocabRepo.SaveBatch(ctx, newVocabs)
		if err != nil {
			return nil, apperr.InternalServerError("common.internal_server_error", err)
		}
	}

	return &vdto.BulkImportResponse{
		Imported: imported,
		Skipped:  skipped,
		Total:    len(req.Vocabularies),
	}, nil
}
