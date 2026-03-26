package vocabulary

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"learning-go/internal/shared/middleware"
	"learning-go/internal/vocabulary/adapter/handler"
	"learning-go/internal/vocabulary/adapter/repository"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/application/usecase"
)

type Module struct {
	handler *handler.VocabularyHandler
}

func NewModule(db *gorm.DB, ocrScanner port.OCRScannerPort) *Module {
	vocabRepo := repository.NewVocabularyRepository(db)
	folderRepo := repository.NewFolderRepository(db)
	topicRepo := repository.NewTopicRepository(db)
	grammarRepo := repository.NewGrammarPointRepository(db)

	vocabCmd := usecase.NewVocabularyCommand(vocabRepo, topicRepo, grammarRepo)
	vocabQry := usecase.NewVocabularyQuery(vocabRepo, topicRepo, grammarRepo)
	folderCmd := usecase.NewFolderCommand(folderRepo, vocabRepo)
	folderQry := usecase.NewFolderQuery(folderRepo)
	topicQry := usecase.NewTopicQuery(topicRepo)
	importCmd := usecase.NewImportCommand(vocabRepo)

	vocabHandler := handler.NewVocabularyHandler(vocabCmd, vocabQry, folderCmd, folderQry, topicQry, importCmd, ocrScanner)

	return &Module{handler: vocabHandler}
}

func (module *Module) RegisterRoutes(public, protected *gin.RouterGroup) {
	v1 := protected.Group("/v1")

	// Topics
	v1.GET("/topics", module.handler.ListTopics)

	// Vocabulary CRUD
	v1.POST("/vocabularies", module.handler.CreateVocabulary)
	v1.GET("/vocabularies/:id", module.handler.GetVocabulary)
	v1.GET("/vocabularies/:id/detail", module.handler.GetVocabularyDetail)
	v1.GET("/vocabularies/hsk/:level", module.handler.ListByHSKLevel)
	v1.GET("/vocabularies/topic/:slug", module.handler.ListByTopic)
	v1.GET("/vocabularies/search", module.handler.SearchVocabulary)
	v1.PUT("/vocabularies/:id", module.handler.UpdateVocabulary)
	v1.DELETE("/vocabularies/:id", module.handler.DeleteVocabulary)

	// OCR scan
	public.POST("/vocabularies/ocr-scan", middleware.TimeoutMiddleware(60*time.Second), module.handler.ProcessOCRScan)

	// Admin import
	v1.POST("/admin/vocabularies/import", module.handler.ImportVocabularies)

	// Folder CRUD
	v1.POST("/folders", module.handler.CreateFolder)
	v1.GET("/folders", module.handler.ListFolders)
	v1.PUT("/folders/:id", module.handler.UpdateFolder)
	v1.DELETE("/folders/:id", module.handler.DeleteFolder)

	// Folder-Vocabulary operations
	v1.POST("/folders/:id/vocabularies", module.handler.AddVocabularyToFolder)
	v1.DELETE("/folders/:id/vocabularies/:vocab_id", module.handler.RemoveVocabularyFromFolder)
	v1.GET("/folders/:id/vocabularies", module.handler.ListFolderVocabularies)
}
