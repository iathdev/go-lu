package domain

// Example is a value object representing a usage example for a vocabulary word.
type Example struct {
	SentenceCN string `json:"sentence_cn"`
	SentenceVI string `json:"sentence_vi"`
	AudioURL   string `json:"audio_url,omitempty"`
}
