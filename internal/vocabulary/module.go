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
	// Repos
	langRepo := repository.NewLanguageRepository(db)
	catRepo := repository.NewCategoryRepository(db)
	plRepo := repository.NewProficiencyLevelRepository(db)
	vocabRepo := repository.NewVocabularyRepository(db)
	folderRepo := repository.NewFolderRepository(db)
	topicRepo := repository.NewTopicRepository(db)
	grammarRepo := repository.NewGrammarPointRepository(db)

	// Use cases
	langQry := usecase.NewLanguageQuery(langRepo)
	catQry := usecase.NewCategoryQuery(catRepo)
	plQry := usecase.NewProficiencyLevelQuery(plRepo)
	vocabCmd := usecase.NewVocabularyCommand(vocabRepo, topicRepo, grammarRepo)
	vocabQry := usecase.NewVocabularyQuery(vocabRepo, topicRepo, grammarRepo)
	folderCmd := usecase.NewFolderCommand(folderRepo, vocabRepo)
	folderQry := usecase.NewFolderQuery(folderRepo)
	topicQry := usecase.NewTopicQuery(topicRepo)
	gpQry := usecase.NewGrammarPointQuery(grammarRepo)
	importCmd := usecase.NewImportCommand(vocabRepo)

	vocabHandler := handler.NewVocabularyHandler(
		langQry, catQry, plQry,
		vocabCmd, vocabQry,
		folderCmd, folderQry,
		topicQry, gpQry,
		importCmd, ocrScanner,
	)

	return &Module{handler: vocabHandler}
}

func (module *Module) RegisterRoutes(public, protected *gin.RouterGroup) {
	publicV1 := public.Group("/v1")
	v1 := protected.Group("/v1")

	// Group 1: Language & Proficiency (Public)
	publicV1.GET("/languages", module.handler.ListLanguages)
	publicV1.GET("/languages/:id", module.handler.GetLanguage)
	publicV1.GET("/categories", module.handler.ListCategories)
	publicV1.GET("/categories/:id", module.handler.GetCategory)
	publicV1.GET("/proficiency-levels", module.handler.ListProficiencyLevels)
	publicV1.GET("/proficiency-levels/:id", module.handler.GetProficiencyLevel)

	// Group 2: Vocabulary (Protected)
	v1.POST("/vocabularies", module.handler.CreateVocabulary)
	v1.GET("/vocabularies", module.handler.ListVocabularies)
	v1.GET("/vocabularies/:id", module.handler.GetVocabulary)
	v1.GET("/vocabularies/:id/detail", module.handler.GetVocabularyDetail)
	v1.GET("/vocabularies/search", module.handler.SearchVocabulary)
	v1.PUT("/vocabularies/:id", module.handler.UpdateVocabulary)
	v1.DELETE("/vocabularies/:id", module.handler.DeleteVocabulary)

	// Group 2: OCR + Import
	publicV1.POST("/vocabularies/ocr-scan", middleware.TimeoutMiddleware(60*time.Second), module.handler.ProcessOCRScan)
	publicV1.POST("/vocabularies/ocr-extract-text", middleware.TimeoutMiddleware(60*time.Second), module.handler.ExtractText)
	v1.POST("/admin/vocabularies/import", module.handler.ImportVocabularies)

	// Group 3: Classification (Public read, Protected write)
	publicV1.GET("/topics", module.handler.ListTopics)
	publicV1.GET("/topics/:id", module.handler.GetTopic)
	publicV1.GET("/grammar-points", module.handler.ListGrammarPoints)
	publicV1.GET("/grammar-points/:id", module.handler.GetGrammarPoint)
	v1.PUT("/vocabularies/:id/topics", module.handler.SetVocabularyTopics)
	v1.PUT("/vocabularies/:id/grammar-points", module.handler.SetVocabularyGrammarPoints)

	// Group 4: Folders (Protected)
	v1.POST("/folders", module.handler.CreateFolder)
	v1.GET("/folders", module.handler.ListFolders)
	v1.PUT("/folders/:id", module.handler.UpdateFolder)
	v1.DELETE("/folders/:id", module.handler.DeleteFolder)
	v1.POST("/folders/:id/vocabularies", module.handler.AddVocabularyToFolder)
	v1.DELETE("/folders/:id/vocabularies/:vocab_id", module.handler.RemoveVocabularyFromFolder)
	v1.GET("/folders/:id/vocabularies", module.handler.ListFolderVocabularies)
}
