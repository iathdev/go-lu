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
	// Collect all hanzi to check for existing
	hanziList := make([]string, 0, len(req.Vocabularies))
	for _, vocab := range req.Vocabularies {
		hanziList = append(hanziList, vocab.Hanzi)
	}

	existing, err := useCase.vocabRepo.FindByHanziList(ctx, hanziList)
	if err != nil {
		return nil, apperr.InternalServerError("import.check_existing_failed", err)
	}

	existingSet := make(map[string]bool, len(existing))
	for _, vocab := range existing {
		existingSet[vocab.Hanzi.String()] = true
	}

	// Filter out duplicates and create domain entities
	var newVocabs []*domain.Vocabulary
	skipped := 0
	for _, item := range req.Vocabularies {
		if existingSet[item.Hanzi] {
			skipped++
			continue
		}

		vocab, err := domain.NewVocabularyFromParams(domain.VocabularyParams{
			Hanzi:           item.Hanzi,
			Pinyin:          item.Pinyin,
			MeaningVI:       item.MeaningVI,
			MeaningEN:       item.MeaningEN,
			HSKLevel:        item.HSKLevel,
			AudioURL:        item.AudioURL,
			Examples:        mapper.ToExampleEntities(item.Examples),
			Radicals:        item.Radicals,
			StrokeCount:     item.StrokeCount,
			StrokeDataURL:   item.StrokeDataURL,
			RecognitionOnly: item.RecognitionOnly,
			FrequencyRank:   item.FrequencyRank,
		})
		if err != nil {
			skipped++
			continue
		}
		newVocabs = append(newVocabs, vocab)
	}

	imported := 0
	if len(newVocabs) > 0 {
		imported, err = useCase.vocabRepo.SaveBatch(ctx, newVocabs)
		if err != nil {
			return nil, apperr.InternalServerError("import.save_failed", err)
		}
	}

	return &vdto.BulkImportResponse{
		Imported: imported,
		Skipped:  skipped,
		Total:    len(req.Vocabularies),
	}, nil
}
