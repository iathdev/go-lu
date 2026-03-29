package domain

import "github.com/google/uuid"

// FolderID uniquely identifies a Folder aggregate.
type FolderID uuid.UUID

func NewFolderID() FolderID {
	return FolderID(uuid.Must(uuid.NewV7()))
}

func ParseFolderID(raw string) (FolderID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return FolderID{}, err
	}
	return FolderID(parsed), nil
}

func FolderIDFromUUID(id uuid.UUID) FolderID { return FolderID(id) }
func (id FolderID) UUID() uuid.UUID          { return uuid.UUID(id) }
func (id FolderID) String() string           { return uuid.UUID(id).String() }
func (id FolderID) IsZero() bool             { return uuid.UUID(id) == uuid.Nil }
