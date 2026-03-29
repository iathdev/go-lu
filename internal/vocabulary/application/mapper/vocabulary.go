package mapper

import (
	"math"

	"learning-go/internal/shared/dto"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/domain"
)

func ToVocabularyListResponse(vocab *domain.Vocabulary) vdto.VocabularyListResponse {
	return vdto.VocabularyListResponse{
		ID:        vocab.ID.String(),
		Hanzi:     vocab.Hanzi.String(),
		Pinyin:    vocab.Pinyin.String(),
		MeaningVI: vocab.MeaningVI,
		MeaningEN: vocab.MeaningEN,
		HSKLevel:  vocab.HSKLevel.Int(),
	}
}

func ToVocabularyResponse(vocab *domain.Vocabulary) *vdto.VocabularyResponse {
	var examples []vdto.ExampleDTO
	if len(vocab.Examples) > 0 {
		examples = make([]vdto.ExampleDTO, 0, len(vocab.Examples))
		for _, example := range vocab.Examples {
			examples = append(examples, vdto.ExampleDTO{
				SentenceCN: example.SentenceCN,
				SentenceVI: example.SentenceVI,
				AudioURL:   example.AudioURL,
			})
		}
	}

	return &vdto.VocabularyResponse{
		ID:              vocab.ID.String(),
		Hanzi:           vocab.Hanzi.String(),
		Pinyin:          vocab.Pinyin.String(),
		MeaningVI:       vocab.MeaningVI,
		MeaningEN:       vocab.MeaningEN,
		HSKLevel:        vocab.HSKLevel.Int(),
		AudioURL:        vocab.AudioURL,
		Examples:        examples,
		Radicals:        vocab.Radicals,
		StrokeCount:     vocab.StrokeCount,
		StrokeDataURL:   vocab.StrokeDataURL,
		RecognitionOnly: vocab.RecognitionOnly,
		FrequencyRank:   vocab.FrequencyRank,
		CreatedAt:       vocab.CreatedAt,
	}
}

func ToExampleEntities(dtos []vdto.ExampleDTO) []domain.Example {
	if dtos == nil {
		return nil
	}
	examples := make([]domain.Example, 0, len(dtos))
	for _, item := range dtos {
		examples = append(examples, domain.Example{
			SentenceCN: item.SentenceCN,
			SentenceVI: item.SentenceVI,
			AudioURL:   item.AudioURL,
		})
	}
	return examples
}

func ToPaginatedResponse(vocabs []*domain.Vocabulary, total int64, pagination dto.PaginationRequest) *dto.PaginatedResponse {
	items := make([]*vdto.VocabularyResponse, 0, len(vocabs))
	for _, vocab := range vocabs {
		items = append(items, ToVocabularyResponse(vocab))
	}
	totalPages := int(math.Ceil(float64(total) / float64(pagination.PageSize)))
	return &dto.PaginatedResponse{
		Items: items,
		Metadata: dto.PaginationMeta{
			Total:      total,
			Page:       pagination.Page,
			PageSize:   pagination.PageSize,
			TotalPages: totalPages,
		},
	}
}
