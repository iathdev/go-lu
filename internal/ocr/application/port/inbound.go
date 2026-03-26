package port

import (
	"context"
	"learning-go/internal/ocr/application/dto"
)

type OCRCommandPort interface {
	ProcessScan(ctx context.Context, req dto.OCRScanRequest) (*dto.OCRScanResponse, error)
}
