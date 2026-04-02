package service

import (
	"context"

	ocrdto "learning-go/internal/ocr/application/dto"
	ocrport "learning-go/internal/ocr/application/port"
	vocabport "learning-go/internal/vocabulary/application/port"
)

type OCRAdapter struct {
	ocrCmd ocrport.OCRCommandPort
}

func NewOCRAdapter(ocrCmd ocrport.OCRCommandPort) vocabport.OCRScannerPort {
	return &OCRAdapter{ocrCmd: ocrCmd}
}

func (adapter *OCRAdapter) ProcessScan(ctx context.Context, req vocabport.OCRScanInput) (*vocabport.OCRScanOutput, error) {
	result, err := adapter.ocrCmd.ProcessScan(ctx, ocrdto.OCRScanRequest{
		Image:    req.Image,
		Type:     req.Type,
		Language: req.Language,
		Engine:   req.Engine,
	})
	if err != nil {
		return nil, err
	}

	return &vocabport.OCRScanOutput{
		Items:         toCharacterOutputs(result.Items),
		LowConfidence: toCharacterOutputs(result.LowConfidence),
		EngineUsed:    result.Metadata.EngineUsed,
		TotalDetected: result.Metadata.TotalDetected,
		ProcessingMs:  result.Metadata.ProcessingTimeMs,
	}, nil
}

func (adapter *OCRAdapter) ExtractText(ctx context.Context, req vocabport.OCRScanInput) (*vocabport.OCRExtractTextOutput, error) {
	result, err := adapter.ocrCmd.ExtractText(ctx, ocrdto.OCRExtractTextRequest{
		Image:    req.Image,
		Type:     req.Type,
		Language: req.Language,
		Engine:   req.Engine,
	})
	if err != nil {
		return nil, err
	}

	blocks := make([]vocabport.OCRTextBlockOutput, 0, len(result.Blocks))
	for _, block := range result.Blocks {
		blocks = append(blocks, vocabport.OCRTextBlockOutput{
			Text:        block.Text,
			BoundingBox: mapBoundingBoxOutput(block.BoundingBox),
			Confidence:  block.Confidence,
		})
	}

	return &vocabport.OCRExtractTextOutput{
		Blocks:   blocks,
		FullText: result.FullText,
		Metadata: vocabport.OCRExtractTextMeta{
			EngineUsed:       result.Metadata.EngineUsed,
			TotalBlocks:      result.Metadata.TotalBlocks,
			ProcessingTimeMs: result.Metadata.ProcessingTimeMs,
		},
	}, nil
}

func toCharacterOutputs(items []ocrdto.OCRCharacterItem) []vocabport.OCRCharacterOutput {
	outputs := make([]vocabport.OCRCharacterOutput, 0, len(items))
	for _, item := range items {
		outputs = append(outputs, vocabport.OCRCharacterOutput{
			Text:          item.Text,
			Pronunciation: item.Pronunciation,
			Confidence:    item.Confidence,
			Candidates:    item.Candidates,
			BoundingBox:   mapBoundingBoxOutput(item.BoundingBox),
		})
	}
	return outputs
}

func mapBoundingBoxOutput(bbox *ocrdto.BoundingBoxDTO) *vocabport.BoundingBoxOutput {
	if bbox == nil || len(bbox.Vertices) == 0 {
		return nil
	}
	vertices := make([]vocabport.PointOutput, len(bbox.Vertices))
	for i, v := range bbox.Vertices {
		vertices[i] = vocabport.PointOutput{X: v.X, Y: v.Y}
	}
	return &vocabport.BoundingBoxOutput{Vertices: vertices}
}
