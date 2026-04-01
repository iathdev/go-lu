package constant

// PrepCategoryID maps to Prep platform's ProductCategoryId enum.
type PrepCategoryID int

const (
	PrepCategoryIELTS PrepCategoryID = 1
	PrepCategoryTOEIC PrepCategoryID = 4
	PrepCategoryHSK   PrepCategoryID = 5
	PrepCategoryHSKV3 PrepCategoryID = 13
)

var prepCategoryNames = map[PrepCategoryID]string{
	PrepCategoryIELTS: "IELTS",
	PrepCategoryTOEIC: "TOEIC",
	PrepCategoryHSK:   "HSK",
	PrepCategoryHSKV3: "HSK 3.0",
}

func (id PrepCategoryID) String() string {
	if name, ok := prepCategoryNames[id]; ok {
		return name
	}
	return "Unknown"
}

func (id PrepCategoryID) Int() int { return int(id) }

func (id PrepCategoryID) IsValid() bool {
	_, ok := prepCategoryNames[id]
	return ok
}
