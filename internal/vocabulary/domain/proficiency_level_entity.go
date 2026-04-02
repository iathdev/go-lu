package domain

import "time"

// ProficiencyLevel represents a level within a proficiency category (e.g. HSK 1, JLPT N5).
type ProficiencyLevel struct {
	ID            LevelID
	CategoryID    CategoryID
	PrepLevelID   *int
	Code          string
	Name          string
	Target        float64
	DisplayTarget string
	Offset        int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
