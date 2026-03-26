package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrFolderNameRequired = errors.New("folder name is required")
)

type Folder struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewFolder(userID uuid.UUID, name, description string) (*Folder, error) {
	if name == "" {
		return nil, ErrFolderNameRequired
	}

	return &Folder{
		ID:          uuid.Must(uuid.NewV7()),
		UserID:      userID,
		Name:        name,
		Description: description,
	}, nil
}

func (folder *Folder) Update(name, description string) error {
	if name == "" {
		return ErrFolderNameRequired
	}
	folder.Name = name
	folder.Description = description
	return nil
}
