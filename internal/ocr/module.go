package ocr

import (
	"learning-go/internal/ocr/application/port"
	"learning-go/internal/ocr/application/usecase"
)

type Module struct {
	OCRCommand port.OCRCommandPort
}

func NewModule(engines port.OCREngineRegistry) *Module {
	ocrCmd := usecase.NewOCRCommand(engines)
	return &Module{OCRCommand: ocrCmd}
}
