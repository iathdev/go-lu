package domain

import "github.com/google/uuid"

// UserID identifies a user across modules. Defined here to avoid cross-module imports.
type UserID uuid.UUID

func ParseUserID(raw string) (UserID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return UserID{}, err
	}
	return UserID(parsed), nil
}

func UserIDFromUUID(id uuid.UUID) UserID { return UserID(id) }
func (id UserID) UUID() uuid.UUID        { return uuid.UUID(id) }
func (id UserID) String() string          { return uuid.UUID(id).String() }
func (id UserID) IsZero() bool            { return uuid.UUID(id) == uuid.Nil }
