package response

import (
	"errors"
	"net/http"
	"time"

	apperr "learning-go/internal/shared/error"
	"learning-go/internal/shared/i18n"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// --- Response envelope (v2) ---

// APIResponse is the unified envelope for all API responses.
// Fields with omitempty are omitted when not applicable (data on error, error on success).
type APIResponse struct {
	Success bool         `json:"success"`
	Data    interface{}  `json:"data,omitempty"`
	Error   *ErrorObject `json:"error,omitempty"`
	Meta    Meta         `json:"meta"`
}

// ErrorObject is the structured error payload.
// Details is omitted for simple errors.
type ErrorObject struct {
	Code    apperr.Code    `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Meta carries request tracing info and optional pagination.
type Meta struct {
	RequestID  string `json:"request_id"`
	Timestamp  string `json:"timestamp"`
	Total      *int64 `json:"total,omitempty"`
	Page       *int   `json:"page,omitempty"`
	PageSize   *int   `json:"page_size,omitempty"`
	TotalPages *int   `json:"total_pages,omitempty"`
}

// PaginationMeta holds pagination values for building Meta.
type PaginationMeta struct {
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// --- Success helpers ---

// Success sends a success response with data.
func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, APIResponse{
		Success: true,
		Data:    data,
		Meta:    buildMeta(c),
	})
}

// SuccessNoContent sends a success response without data (e.g. DELETE).
func SuccessNoContent(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Meta:    buildMeta(c),
	})
}

// SuccessList sends a success response with paginated data.
func SuccessList(c *gin.Context, data interface{}, pagination PaginationMeta) {
	meta := buildMeta(c)
	meta.Total = &pagination.Total
	meta.Page = &pagination.Page
	meta.PageSize = &pagination.PageSize
	meta.TotalPages = &pagination.TotalPages

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// --- Error helpers ---

// HandleError maps an AppError to the v2 error response.
func HandleError(c *gin.Context, err error) {
	var domErr *apperr.AppError
	if !errors.As(err, &domErr) {
		sendError(c, http.StatusInternalServerError, apperr.CodeInternalServerError, "common.internal_error", nil)
		return
	}

	status, ok := codeToStatus[domErr.Code()]
	if !ok {
		status = http.StatusInternalServerError
	}

	lang := getLang(c)
	msg := i18n.TranslateText(lang, domErr.Message())

	c.JSON(status, APIResponse{
		Success: false,
		Error: &ErrorObject{
			Code:    domErr.Code(),
			Message: msg,
			Details: domErr.Data(),
		},
		Meta: buildMeta(c),
	})
}

// ValidationError formats Gin binding/validation errors into field-level details (HTTP 422).
func ValidationError(c *gin.Context, err error) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make(map[string]any, len(ve))
		for _, fe := range ve {
			fields[fe.Field()] = fe.Tag()
		}
		sendError(c, http.StatusUnprocessableEntity, apperr.CodeValidationFailed, "common.validation_failed", fields)
		return
	}
	sendError(c, http.StatusUnprocessableEntity, apperr.CodeValidationFailed, "common.validation_failed", nil)
}

// Shorthand error helpers for direct use in middleware/router (no AppError).

func BadRequest(c *gin.Context, messageKey string) {
	sendError(c, http.StatusBadRequest, apperr.CodeBadRequest, messageKey)
}

func Unauthorized(c *gin.Context, messageKey string) {
	sendError(c, http.StatusUnauthorized, apperr.CodeUnauthorized, messageKey)
}

func Forbidden(c *gin.Context, messageKey string) {
	sendError(c, http.StatusForbidden, apperr.CodeForbidden, messageKey)
}

func NotFound(c *gin.Context, messageKey string) {
	sendError(c, http.StatusNotFound, apperr.CodeNotFound, messageKey)
}

func Conflict(c *gin.Context, messageKey string) {
	sendError(c, http.StatusConflict, apperr.CodeConflict, messageKey)
}

func TooManyRequests(c *gin.Context, messageKey string) {
	sendError(c, http.StatusTooManyRequests, apperr.CodeTooManyRequests, messageKey)
}

func InternalServerError(c *gin.Context, messageKey string) {
	if messageKey == "" {
		messageKey = "common.internal_error"
	}
	sendError(c, http.StatusInternalServerError, apperr.CodeInternalServerError, messageKey)
}

func ServiceUnavailable(c *gin.Context, messageKey string) {
	sendError(c, http.StatusServiceUnavailable, apperr.CodeServiceUnavailable, messageKey)
}

func MethodNotAllowed(c *gin.Context, messageKey string) {
	sendError(c, http.StatusMethodNotAllowed, apperr.CodeBadRequest, messageKey)
}

// --- Internal ---

var codeToStatus = map[apperr.Code]int{
	apperr.CodeBadRequest:          http.StatusBadRequest,
	apperr.CodeUnauthorized:        http.StatusUnauthorized,
	apperr.CodeForbidden:           http.StatusForbidden,
	apperr.CodeNotFound:            http.StatusNotFound,
	apperr.CodeConflict:            http.StatusConflict,
	apperr.CodeValidationFailed: http.StatusUnprocessableEntity,
	apperr.CodeTooManyRequests:     http.StatusTooManyRequests,
	apperr.CodeInternalServerError: http.StatusInternalServerError,
	apperr.CodeServiceUnavailable:  http.StatusServiceUnavailable,
}

// sendError builds and sends a v2 error response.
func sendError(c *gin.Context, status int, code apperr.Code, messageKey string, details ...map[string]any) {
	lang := getLang(c)
	msg := i18n.TranslateText(lang, messageKey)

	var det map[string]any
	if len(details) > 0 {
		det = details[0]
	}

	c.JSON(status, APIResponse{
		Success: false,
		Error: &ErrorObject{
			Code:    code,
			Message: msg,
			Details: det,
		},
		Meta: buildMeta(c),
	})
}

func buildMeta(c *gin.Context) Meta {
	return Meta{
		RequestID: c.GetString("request_id"),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func getLang(c *gin.Context) string {
	lang := c.GetString("lang")
	if lang != "" {
		return i18n.Normalize(lang)
	}
	return i18n.FromAcceptLanguage(c.GetHeader("Accept-Language"))
}
