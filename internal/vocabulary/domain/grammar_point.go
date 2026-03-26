package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrGrammarPointNotFound = errors.New("grammar point not found")
)

type GrammarPoint struct {
	ID            uuid.UUID
	Code          string
	Pattern       string
	ExampleCN     string
	ExampleVI     string
	Rule          string
	CommonMistake string
	HSKLevel      int
	CreatedAt     time.Time
}
