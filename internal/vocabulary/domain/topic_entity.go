package domain

import "time"

// Topic is a system-defined classification entity for vocabularies.
type Topic struct {
	ID        TopicID
	NameCN    string
	NameVI    string
	NameEN    string
	Slug      string
	SortOrder int
	CreatedAt time.Time
}

func NewTopic(slug, nameCN, nameVI, nameEN string, sortOrder int) (*Topic, error) {
	if slug == "" {
		return nil, ErrTopicSlugRequired
	}

	return &Topic{
		ID:        NewTopicID(),
		NameCN:    nameCN,
		NameVI:    nameVI,
		NameEN:    nameEN,
		Slug:      slug,
		SortOrder: sortOrder,
	}, nil
}
