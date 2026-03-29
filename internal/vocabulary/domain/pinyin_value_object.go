package domain

// Pinyin represents a validated romanization of a Chinese word.
type Pinyin string

func NewPinyin(raw string) (Pinyin, error) {
	if raw == "" {
		return "", ErrPinyinRequired
	}
	return Pinyin(raw), nil
}

func (pinyin Pinyin) String() string { return string(pinyin) }
