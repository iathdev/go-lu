package domain

// HSKLevel represents a validated HSK proficiency level (1-9).
type HSKLevel int

func NewHSKLevel(raw int) (HSKLevel, error) {
	if raw < 1 || raw > 9 {
		return 0, ErrInvalidHSKLevel
	}
	return HSKLevel(raw), nil
}

func (level HSKLevel) Int() int { return int(level) }
