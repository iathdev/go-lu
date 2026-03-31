package usecase

import (
	"context"

	apperr "learning-go/internal/shared/error"
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/application/mapper"
	"learning-go/internal/vocabulary/application/port"
	"learning-go/internal/vocabulary/domain"
)

type FolderCommand struct {
	folderRepo port.FolderRepositoryPort
	vocabRepo  port.VocabularyRepositoryPort
}

func NewFolderCommand(folderRepo port.FolderRepositoryPort, vocabRepo port.VocabularyRepositoryPort) port.FolderCommandPort {
	return &FolderCommand{folderRepo: folderRepo, vocabRepo: vocabRepo}
}

func (useCase *FolderCommand) CreateFolder(ctx context.Context, userID string, req vdto.CreateFolderRequest) (*vdto.FolderResponse, error) {
	uid, err := domain.ParseUserID(userID)
	if err != nil {
		return nil, apperr.BadRequest("folder.invalid_user_id")
	}

	langID, err := domain.ParseLanguageID(req.LanguageID)
	if err != nil {
		return nil, apperr.BadRequest("folder.invalid_language_id")
	}

	folder, err := domain.NewFolder(uid, langID, req.Name, req.Description)
	if err != nil {
		return nil, apperr.ValidationFailed("folder.invalid_input")
	}

	if err := useCase.folderRepo.Save(ctx, folder); err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	return mapper.ToFolderResponse(folder, 0), nil
}

func (useCase *FolderCommand) UpdateFolder(ctx context.Context, id string, userID string, req vdto.UpdateFolderRequest) (*vdto.FolderResponse, error) {
	folder, err := getOwnedFolder(ctx, useCase.folderRepo, id, userID)
	if err != nil {
		return nil, err
	}

	if err := folder.Update(req.Name, req.Description); err != nil {
		return nil, apperr.ValidationFailed("folder.invalid_input")
	}

	if err := useCase.folderRepo.Update(ctx, folder); err != nil {
		return nil, apperr.InternalServerError("common.internal_server_error", err)
	}

	// Fetch vocab count for updated folder
	countMap, err := useCase.folderRepo.CountVocabulariesByFolderIDs(ctx, []domain.FolderID{folder.ID})
	if err != nil {
		return mapper.ToFolderResponse(folder, 0), nil
	}

	return mapper.ToFolderResponse(folder, countMap[folder.ID]), nil
}

func (useCase *FolderCommand) DeleteFolder(ctx context.Context, id string, userID string) error {
	folder, err := getOwnedFolder(ctx, useCase.folderRepo, id, userID)
	if err != nil {
		return err
	}

	if err := useCase.folderRepo.Delete(ctx, folder.ID); err != nil {
		return apperr.InternalServerError("common.internal_server_error", err)
	}

	return nil
}

func (useCase *FolderCommand) AddVocabulary(ctx context.Context, folderID string, vocabID string, userID string) error {
	folder, err := getOwnedFolder(ctx, useCase.folderRepo, folderID, userID)
	if err != nil {
		return err
	}

	vid, err := domain.ParseVocabularyID(vocabID)
	if err != nil {
		return apperr.BadRequest("vocabulary.invalid_id")
	}

	vocab, err := useCase.vocabRepo.FindByID(ctx, vid)
	if err != nil {
		return apperr.InternalServerError("common.internal_server_error", err)
	}
	if vocab == nil {
		return apperr.NotFound("vocabulary.not_found")
	}

	if err := useCase.folderRepo.AddVocabulary(ctx, folder.ID, vid); err != nil {
		return apperr.InternalServerError("common.internal_server_error", err)
	}

	return nil
}

func (useCase *FolderCommand) RemoveVocabulary(ctx context.Context, folderID string, vocabID string, userID string) error {
	folder, err := getOwnedFolder(ctx, useCase.folderRepo, folderID, userID)
	if err != nil {
		return err
	}

	vid, err := domain.ParseVocabularyID(vocabID)
	if err != nil {
		return apperr.BadRequest("vocabulary.invalid_id")
	}

	if err := useCase.folderRepo.RemoveVocabulary(ctx, folder.ID, vid); err != nil {
		return apperr.InternalServerError("common.internal_server_error", err)
	}
	return nil
}
