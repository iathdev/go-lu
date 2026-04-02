package domain

import "github.com/google/uuid"

// LevelID uniquely identifies a ProficiencyLevel entity.
type LevelID uuid.UUID

func NewLevelID() LevelID {
	return LevelID(uuid.Must(uuid.NewV7()))
}

func ParseLevelID(raw string) (LevelID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return LevelID{}, err
	}
	return LevelID(parsed), nil
}

func LevelIDFromUUID(id uuid.UUID) LevelID { return LevelID(id) }
func (id LevelID) UUID() uuid.UUID          { return uuid.UUID(id) }
func (id LevelID) String() string            { return uuid.UUID(id).String() }
func (id LevelID) IsZero() bool              { return uuid.UUID(id) == uuid.Nil }
