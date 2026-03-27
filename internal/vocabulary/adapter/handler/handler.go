package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"learning-go/internal/shared/dto"
	"learning-go/internal/shared/response"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/port"
)

type VocabularyHandler struct {
	vocabCmd  port.VocabularyCommandPort
	vocabQry  port.VocabularyQueryPort
	folderCmd port.FolderCommandPort
	folderQry port.FolderQueryPort
	topicQry  port.TopicQueryPort
	importCmd port.ImportCommandPort
	ocrCmd    port.OCRScannerPort
}

func NewVocabularyHandler(
	vocabCmd port.VocabularyCommandPort,
	vocabQry port.VocabularyQueryPort,
	folderCmd port.FolderCommandPort,
	folderQry port.FolderQueryPort,
	topicQry port.TopicQueryPort,
	importCmd port.ImportCommandPort,
	ocrCmd port.OCRScannerPort,
) *VocabularyHandler {
	return &VocabularyHandler{
		vocabCmd:  vocabCmd,
		vocabQry:  vocabQry,
		folderCmd: folderCmd,
		folderQry: folderQry,
		topicQry:  topicQry,
		importCmd: importCmd,
		ocrCmd:    ocrCmd,
	}
}

// --- Vocabulary endpoints ---

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
	res, err := handler.vocabQry.GetVocabulary(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) GetVocabularyDetail(c *gin.Context) {
	res, err := handler.vocabQry.GetVocabularyDetail(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, res)
}

func (handler *VocabularyHandler) ListByHSKLevel(c *gin.Context) {
	level, err := strconv.Atoi(c.Param("level"))
	if err != nil || level < 1 || level > 9 {
		response.BadRequest(c, "common.bad_request")
		return
	}

	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := handler.vocabQry.ListByHSKLevel(c.Request.Context(), level, pagination)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessWithMetadata(c, http.StatusOK, res.Items, res.Metadata)
}

func (handler *VocabularyHandler) ListByTopic(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		response.BadRequest(c, "common.bad_request")
		return
	}

	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := handler.vocabQry.ListByTopic(c.Request.Context(), slug, pagination)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessWithMetadata(c, http.StatusOK, res.Items, res.Metadata)
}

func (handler *VocabularyHandler) SearchVocabulary(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		response.BadRequest(c, "common.bad_request")
		return
	}

	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := handler.vocabQry.SearchVocabulary(c.Request.Context(), query, pagination)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessWithMetadata(c, http.StatusOK, res.Items, res.Metadata)
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
	response.Success(c, http.StatusOK, nil, "common.deleted_successfully")
}

// --- Topic endpoints ---

func (handler *VocabularyHandler) ListTopics(c *gin.Context) {
	res, err := handler.topicQry.ListTopics(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, res)
}

// --- OCR endpoints ---

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
		response.BadRequest(ctx, "ocr.image_required")
		return
	}
	defer file.Close()

	if header.Size > maxImageSize {
		response.BadRequest(ctx, "ocr.image_too_large")
		return
	}

	imageBytes, err := io.ReadAll(io.LimitReader(file, maxImageSize+1))
	if err != nil {
		response.BadRequest(ctx, "ocr.image_read_failed")
		return
	}

	contentType := http.DetectContentType(imageBytes)
	if !strings.HasPrefix(contentType, "image/") {
		response.BadRequest(ctx, "ocr.invalid_image_type")
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

// --- Import endpoints ---

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

// --- Folder endpoints ---

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
	res, err := handler.folderQry.ListFolders(c.Request.Context(), userID)
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
	response.Success(c, http.StatusOK, nil, "common.deleted_successfully")
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
	response.Success(c, http.StatusOK, nil)
}

func (handler *VocabularyHandler) RemoveVocabularyFromFolder(c *gin.Context) {
	userID := c.GetString("user_id")
	if err := handler.folderCmd.RemoveVocabulary(c.Request.Context(), c.Param("id"), c.Param("vocab_id"), userID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, nil, "common.deleted_successfully")
}

func (handler *VocabularyHandler) ListFolderVocabularies(c *gin.Context) {
	userID := c.GetString("user_id")
	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		response.ValidationError(c, err)
		return
	}

	res, err := handler.folderQry.ListVocabularies(c.Request.Context(), c.Param("id"), userID, pagination)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessWithMetadata(c, http.StatusOK, res.Items, res.Metadata)
}
