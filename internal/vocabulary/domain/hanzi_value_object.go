package domain

// Hanzi represents a validated Chinese character or word.
type Hanzi string

func NewHanzi(raw string) (Hanzi, error) {
	if raw == "" {
		return "", ErrHanziRequired
	}
	return Hanzi(raw), nil
}

func (hanzi Hanzi) String() string { return string(hanzi) }
