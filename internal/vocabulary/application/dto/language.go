package dto

type LanguageResponse struct {
	ID         string         `json:"id"`
	Code       string         `json:"code"`
	NameEN     string         `json:"name_en"`
	NameNative string         `json:"name_native"`
	IsActive   bool           `json:"is_active"`
	Config     map[string]any `json:"config"`
}
