package dto

type TopicResponse struct {
	ID         string            `json:"id"`
	CategoryID string            `json:"category_id"`
	Slug       string            `json:"slug"`
	Names      map[string]string `json:"names"`
	Offset     int               `json:"offset"`
}

type SetTopicsRequest struct {
	TopicIDs []string `json:"topic_ids" binding:"required"`
}
