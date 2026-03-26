package mapper

import (
	vdto "learning-go/internal/vocabulary/application/dto"
	"learning-go/internal/vocabulary/domain"
)

func ToFolderResponse(folder *domain.Folder) *vdto.FolderResponse {
	return &vdto.FolderResponse{
		ID:          folder.ID.String(),
		Name:        folder.Name,
		Description: folder.Description,
		CreatedAt:   folder.CreatedAt,
	}
}
