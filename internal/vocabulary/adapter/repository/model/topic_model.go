package model

import (
	"learning-go/internal/vocabulary/domain"
	"time"

	"github.com/google/uuid"
)

type TopicModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	NameCN    string    `gorm:"column:name_cn;not null"`
	NameVI    string    `gorm:"column:name_vi;not null"`
	NameEN    string    `gorm:"column:name_en;not null"`
	Slug      string    `gorm:"not null;uniqueIndex"`
	SortOrder int       `gorm:"not null;default:0"`
	CreatedAt time.Time
}

func (TopicModel) TableName() string { return "topics" }

func (m *TopicModel) ToEntity() *domain.Topic {
	return &domain.Topic{
		ID:        domain.TopicIDFromUUID(m.ID),
		NameCN:    m.NameCN,
		NameVI:    m.NameVI,
		NameEN:    m.NameEN,
		Slug:      m.Slug,
		SortOrder: m.SortOrder,
		CreatedAt: m.CreatedAt,
	}
}
