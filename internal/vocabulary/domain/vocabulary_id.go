package domain

import "github.com/google/uuid"

// VocabularyID uniquely identifies a Vocabulary aggregate.
type VocabularyID uuid.UUID

func NewVocabularyID() VocabularyID {
	return VocabularyID(uuid.Must(uuid.NewV7()))
}

func ParseVocabularyID(raw string) (VocabularyID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return VocabularyID{}, err
	}
	return VocabularyID(parsed), nil
}

func VocabularyIDFromUUID(id uuid.UUID) VocabularyID { return VocabularyID(id) }
func (id VocabularyID) UUID() uuid.UUID              { return uuid.UUID(id) }
func (id VocabularyID) String() string                { return uuid.UUID(id).String() }
func (id VocabularyID) IsZero() bool                  { return uuid.UUID(id) == uuid.Nil }
