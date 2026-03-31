package usecase

import (
	"context"
	"learning-go/internal/auth/application/dto"
	"learning-go/internal/auth/application/port"
	"learning-go/internal/auth/domain"
	apperr "learning-go/internal/shared/error"

	"github.com/google/uuid"
)

type AuthUseCase struct {
	userRepo port.UserRepositoryPort
}

func NewAuthUseCase(userRepo port.UserRepositoryPort) port.AuthUseCasePort {
	return &AuthUseCase{userRepo: userRepo}
}

func (useCase *AuthUseCase) GetMe(ctx context.Context, userID uuid.UUID, prepUser *domain.PrepUser) (*dto.AuthMeResponse, error) {
	user, err := useCase.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_error", err)
	}
	if user == nil {
		return nil, apperr.NotFound("auth.not_found")
	}

	return &dto.AuthMeResponse{
		ID:                  user.ID.String(),
		PrepUserID:          prepUser.PrepUserID,
		Name:                prepUser.Name,
		Email:               prepUser.Email,
		IsFirstLogin:        prepUser.IsFirstLogin,
		ForceUpdatePassword: prepUser.ForceUpdatePassword,
		CreatedAt:           user.CreatedAt,
		UpdatedAt:           user.UpdatedAt,
	}, nil
}

func (useCase *AuthUseCase) UpsertFromPrepUser(ctx context.Context, prepUser *domain.PrepUser) (*domain.User, error) {
	existing, err := useCase.userRepo.FindByPrepUserID(ctx, prepUser.PrepUserID)
	if err != nil {
		return nil, apperr.InternalServerError("common.internal_error", err)
	}

	if existing == nil {
		user := domain.NewUser(prepUser.PrepUserID, prepUser.Email, prepUser.Name)
		if err := useCase.userRepo.Upsert(ctx, user); err != nil {
			return nil, apperr.InternalServerError("common.internal_error", err)
		}
		return user, nil
	}

	if existing.Email != prepUser.Email || existing.Name != prepUser.Name {
		existing.Email = prepUser.Email
		existing.Name = prepUser.Name
		if err := useCase.userRepo.Update(ctx, existing); err != nil {
			return nil, apperr.InternalServerError("common.internal_error", err)
		}
	}

	return existing, nil
}
