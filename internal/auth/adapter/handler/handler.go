package handler

import (
	"errors"
	"learning-go/internal/auth/application/port"
	"learning-go/internal/auth/domain"
	"learning-go/internal/shared/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authUseCase port.AuthUseCasePort
}

func NewAuthHandler(authUseCase port.AuthUseCasePort) *AuthHandler {
	return &AuthHandler{authUseCase: authUseCase}
}

func (handler *AuthHandler) GetMe(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Unauthorized(c, "common.unauthorized")
		return
	}

	prepUser, _ := c.Get("prep_user")

	res, err := handler.authUseCase.GetMe(c.Request.Context(), userID, prepUser.(*domain.PrepUser))
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, res)
}

func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user_id not in context")
	}
	return uuid.Parse(userIDStr.(string))
}
