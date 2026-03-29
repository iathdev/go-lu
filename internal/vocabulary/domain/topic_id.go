package domain

import "github.com/google/uuid"

// TopicID uniquely identifies a Topic entity.
type TopicID uuid.UUID

func NewTopicID() TopicID {
	return TopicID(uuid.Must(uuid.NewV7()))
}

func ParseTopicID(raw string) (TopicID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return TopicID{}, err
	}
	return TopicID(parsed), nil
}

func TopicIDFromUUID(id uuid.UUID) TopicID { return TopicID(id) }
func (id TopicID) UUID() uuid.UUID        { return uuid.UUID(id) }
func (id TopicID) String() string          { return uuid.UUID(id).String() }
func (id TopicID) IsZero() bool            { return uuid.UUID(id) == uuid.Nil }
