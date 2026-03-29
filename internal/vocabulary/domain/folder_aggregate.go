package domain

import "time"

// Folder is the aggregate root for user-created vocabulary decks.
// Invariants: Name is required, owned by a single user.
type Folder struct {
	ID          FolderID
	UserID      UserID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewFolder(userID UserID, name, description string) (*Folder, error) {
	if name == "" {
		return nil, ErrFolderNameRequired
	}

	return &Folder{
		ID:          NewFolderID(),
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
