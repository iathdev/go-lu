package domain

import "github.com/google/uuid"

// GrammarPointID uniquely identifies a GrammarPoint entity.
type GrammarPointID uuid.UUID

func NewGrammarPointID() GrammarPointID {
	return GrammarPointID(uuid.Must(uuid.NewV7()))
}

func ParseGrammarPointID(raw string) (GrammarPointID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return GrammarPointID{}, err
	}
	return GrammarPointID(parsed), nil
}

func GrammarPointIDFromUUID(id uuid.UUID) GrammarPointID { return GrammarPointID(id) }
func (id GrammarPointID) UUID() uuid.UUID                { return uuid.UUID(id) }
func (id GrammarPointID) String() string                  { return uuid.UUID(id).String() }
func (id GrammarPointID) IsZero() bool                    { return uuid.UUID(id) == uuid.Nil }
