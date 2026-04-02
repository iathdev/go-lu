package dto

type GrammarPointResponse struct {
	ID                 string         `json:"id"`
	CategoryID         string         `json:"category_id"`
	LevelID string         `json:"level_id"`
	Code               string         `json:"code"`
	Pattern            string         `json:"pattern"`
	Examples           map[string]any `json:"examples"`
	Rule               map[string]any `json:"rule"`
	CommonMistakes     map[string]any `json:"common_mistakes"`
}

type SetGrammarPointsRequest struct {
	GrammarPointIDs []string `json:"grammar_point_ids" binding:"required"`
}
