package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTopicNotFound = errors.New("topic not found")
)

type Topic struct {
	ID        uuid.UUID
	NameCN    string
	NameVI    string
	NameEN    string
	Slug      string
	SortOrder int
	CreatedAt time.Time
}
