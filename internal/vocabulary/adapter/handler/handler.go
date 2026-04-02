package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"learning-go/internal/shared/dto"
	"learning-go/internal/shared/response"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/port"
)

type VocabularyHandler struct {
	langQry   port.LanguageQueryPort
	catQry    port.CategoryQueryPort
	plQry     port.ProficiencyLevelQueryPort
	vocabCmd  port.VocabularyCommandPort
	vocabQry  port.VocabularyQueryPort
	folderCmd port.FolderCommandPort
	folderQry port.FolderQueryPort
	topicQry  port.TopicQueryPort
	gpQry     port.GrammarPointQueryPort
	importCmd port.ImportCommandPort
	ocrCmd    port.OCRScannerPort
}

func NewVocabularyHandler(
	langQry port.LanguageQueryPort,
	catQry port.CategoryQueryPort,
	plQry port.ProficiencyLevelQueryPort,
	vocabCmd port.VocabularyCommandPort,
	vocabQry port.VocabularyQueryPort,
	folderCmd port.FolderCommandPort,
	folderQry port.FolderQueryPort,
	topicQry port.TopicQueryPort,
	gpQry port.GrammarPointQueryPort,
	importCmd port.ImportCommandPort,
	ocrCmd port.OCRScannerPort,
) *VocabularyHandler {
	return &VocabularyHandler{
		langQry:   langQry,
		catQry:    catQry,
		plQry:     plQry,
		vocabCmd:  vocabCmd,
		vocabQry:  vocabQry,
		folderCmd: folderCmd,
		folderQry: folderQry,
		topicQry:  topicQry,
		gpQry:     gpQry,
		importCmd: importCmd,
		ocrCmd:    ocrCmd,
	}
}

// --- Group 1: Language & Proficiency ---

func (handler *VocabularyHandler) ListLanguages(c *gin.Context) {
	activeOnly := true
	if c.Query("active_only") == "false" {
		activeOnly = false
	}

	res, err := handler.langQry.ListLanguages(c.Request.Context(), activeOnly)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) GetLanguage(c *gin.Context) {
	res, err := handler.langQry.GetLanguage(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) ListCategories(c *gin.Context) {
	languageID := c.Query("language_id")

	var isPublic *bool
	if v := c.Query("is_public"); v != "" {
		val := v == "true"
		isPublic = &val
	}

	res, err := handler.catQry.ListCategories(c.Request.Context(), languageID, isPublic)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) GetCategory(c *gin.Context) {
	res, err := handler.catQry.GetCategory(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) ListProficiencyLevels(c *gin.Context) {
	categoryID := c.Query("category_id")

	res, err := handler.plQry.ListProficiencyLevels(c.Request.Context(), categoryID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) GetProficiencyLevel(c *gin.Context) {
	res, err := handler.plQry.GetProficiencyLevel(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

// --- Group 2: Vocabulary ---

func (handler *VocabularyHandler) CreateVocabulary(c *gin.Context) {
	var req vdto.CreateVocabularyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := handler.vocabCmd.CreateVocabulary(c.Request.Context(), req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, res)
}

func (handler *VocabularyHandler) GetVocabulary(c *gin.Context) {
	meaningLang := c.Query("meaning_lang")

	res, err := handler.vocabQry.GetVocabulary(c.Request.Context(), c.Param("id"), meaningLang)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) GetVocabularyDetail(c *gin.Context) {
	meaningLang := c.Query("meaning_lang")

	res, err := handler.vocabQry.GetVocabularyDetail(c.Request.Context(), c.Param("id"), meaningLang)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) ListVocabularies(c *gin.Context) {
	var filter vdto.VocabularyFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.ValidationError(c, err)
		return
	}

	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := handler.vocabQry.ListVocabularies(c.Request.Context(), filter, pagination)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	sendList(c, res)
}

func (handler *VocabularyHandler) SearchVocabulary(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		response.BadRequest(c, "common.bad_request")
		return
	}

	languageID := c.Query("language_id")
	meaningLang := c.Query("meaning_lang")

	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := handler.vocabQry.SearchVocabulary(c.Request.Context(), query, languageID, meaningLang, pagination)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	sendList(c, res)
}

func (handler *VocabularyHandler) UpdateVocabulary(c *gin.Context) {
	var req vdto.UpdateVocabularyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := handler.vocabCmd.UpdateVocabulary(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) DeleteVocabulary(c *gin.Context) {
	if err := handler.vocabCmd.DeleteVocabulary(c.Request.Context(), c.Param("id")); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessNoContent(c)
}

func (handler *VocabularyHandler) ImportVocabularies(c *gin.Context) {
	var req vdto.BulkImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := handler.importCmd.ImportVocabularies(c.Request.Context(), req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) ProcessOCRScan(ctx *gin.Context) {
	const maxImageSize = 10 << 20 // 10MB

	// Parse multipart form fields (type, language, engine)
	var httpReq vdto.OCRScanHTTPRequest
	if err := ctx.ShouldBind(&httpReq); err != nil {
		response.ValidationError(ctx, err)
		return
	}

	// Read uploaded image file
	file, header, err := ctx.Request.FormFile("image")
	if err != nil {
		response.BadRequest(ctx, "vocabulary.ocr_image_required")
		return
	}
	defer file.Close()

	if header.Size > maxImageSize {
		response.BadRequest(ctx, "vocabulary.ocr_image_too_large")
		return
	}

	imageBytes, err := io.ReadAll(io.LimitReader(file, maxImageSize+1))
	if err != nil {
		response.BadRequest(ctx, "vocabulary.ocr_image_read_failed")
		return
	}

	contentType := http.DetectContentType(imageBytes)
	if !strings.HasPrefix(contentType, "image/") {
		response.BadRequest(ctx, "vocabulary.ocr_invalid_image_type")
		return
	}

	ocrType := httpReq.Type
	if ocrType == "" {
		ocrType = "auto"
	}

	language := httpReq.Language
	if language == "" {
		language = "zh"
	}

	req := port.OCRScanInput{
		Image:    imageBytes,
		Type:     ocrType,
		Language: language,
		Engine:   httpReq.Engine,
	}

	result, err := handler.ocrCmd.ProcessScan(ctx.Request.Context(), req)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	response.Success(ctx, http.StatusOK, result)
}

func (handler *VocabularyHandler) ExtractText(ctx *gin.Context) {
	const maxImageSize = 10 << 20 // 10MB

	var httpReq vdto.OCRScanHTTPRequest
	if err := ctx.ShouldBind(&httpReq); err != nil {
		response.ValidationError(ctx, err)
		return
	}

	file, header, err := ctx.Request.FormFile("image")
	if err != nil {
		response.BadRequest(ctx, "vocabulary.ocr_image_required")
		return
	}
	defer file.Close()

	if header.Size > maxImageSize {
		response.BadRequest(ctx, "vocabulary.ocr_image_too_large")
		return
	}

	imageBytes, err := io.ReadAll(io.LimitReader(file, maxImageSize+1))
	if err != nil {
		response.BadRequest(ctx, "vocabulary.ocr_image_read_failed")
		return
	}

	contentType := http.DetectContentType(imageBytes)
	if !strings.HasPrefix(contentType, "image/") {
		response.BadRequest(ctx, "vocabulary.ocr_invalid_image_type")
		return
	}

	ocrType := httpReq.Type
	if ocrType == "" {
		ocrType = "auto"
	}

	language := httpReq.Language
	if language == "" {
		language = "zh"
	}

	req := port.OCRScanInput{
		Image:    imageBytes,
		Type:     ocrType,
		Language: language,
		Engine:   httpReq.Engine,
	}

	result, err := handler.ocrCmd.ExtractText(ctx.Request.Context(), req)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	response.Success(ctx, http.StatusOK, result)
}

// --- Group 3: Classification ---

func (handler *VocabularyHandler) ListTopics(c *gin.Context) {
	categoryID := c.Query("category_id")

	res, err := handler.topicQry.ListTopics(c.Request.Context(), categoryID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) GetTopic(c *gin.Context) {
	res, err := handler.topicQry.GetTopic(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) ListGrammarPoints(c *gin.Context) {
	categoryID := c.Query("category_id")
	proficiencyLevelID := c.Query("level_id")

	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := handler.gpQry.ListGrammarPoints(c.Request.Context(), categoryID, proficiencyLevelID, pagination)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	sendList(c, res)
}

func (handler *VocabularyHandler) GetGrammarPoint(c *gin.Context) {
	res, err := handler.gpQry.GetGrammarPoint(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) SetVocabularyTopics(c *gin.Context) {
	var req vdto.SetTopicsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	if err := handler.vocabCmd.SetTopics(c.Request.Context(), c.Param("id"), req.TopicIDs); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessNoContent(c)
}

func (handler *VocabularyHandler) SetVocabularyGrammarPoints(c *gin.Context) {
	var req vdto.SetGrammarPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	if err := handler.vocabCmd.SetGrammarPoints(c.Request.Context(), c.Param("id"), req.GrammarPointIDs); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessNoContent(c)
}

// --- Group 4: Folders ---

func (handler *VocabularyHandler) CreateFolder(c *gin.Context) {
	var req vdto.CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	userID := c.GetString("user_id")
	res, err := handler.folderCmd.CreateFolder(c.Request.Context(), userID, req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, res)
}

func (handler *VocabularyHandler) ListFolders(c *gin.Context) {
	userID := c.GetString("user_id")
	languageID := c.Query("language_id")

	res, err := handler.folderQry.ListFolders(c.Request.Context(), userID, languageID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) UpdateFolder(c *gin.Context) {
	var req vdto.UpdateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	userID := c.GetString("user_id")
	res, err := handler.folderCmd.UpdateFolder(c.Request.Context(), c.Param("id"), userID, req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) DeleteFolder(c *gin.Context) {
	userID := c.GetString("user_id")
	if err := handler.folderCmd.DeleteFolder(c.Request.Context(), c.Param("id"), userID); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessNoContent(c)
}

func (handler *VocabularyHandler) AddVocabularyToFolder(c *gin.Context) {
	var req vdto.FolderVocabularyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	userID := c.GetString("user_id")
	if err := handler.folderCmd.AddVocabulary(c.Request.Context(), c.Param("id"), req.VocabularyID, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessNoContent(c)
}

func (handler *VocabularyHandler) RemoveVocabularyFromFolder(c *gin.Context) {
	userID := c.GetString("user_id")
	if err := handler.folderCmd.RemoveVocabulary(c.Request.Context(), c.Param("id"), c.Param("vocab_id"), userID); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessNoContent(c)
}

func (handler *VocabularyHandler) ListFolderVocabularies(c *gin.Context) {
	userID := c.GetString("user_id")
	meaningLang := c.Query("meaning_lang")

	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := handler.folderQry.ListVocabularies(c.Request.Context(), c.Param("id"), userID, meaningLang, pagination)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	sendList(c, res)
}

// sendList converts a ListResult into a SuccessList response.
func sendList[T any](c *gin.Context, result *dto.ListResult[T]) {
	response.SuccessList(c, result.Items, response.PaginationMeta{
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	})
}
