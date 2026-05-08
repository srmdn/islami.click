package model

type QuizCategory struct {
	Slug        string
	Label       string
	Description string
	Order       int
}

type QuizQuestion struct {
	ID          int
	Category    string
	Difficulty  string
	Question    string
	Options     []string
	Answer      int
	Explanation string
}

type QuizScore struct {
	ID           int
	CategorySlug string
	PlayerName   string
	Score        int
	CorrectCount int
	TotalCount   int
	Difficulty   string
	PlayedAt     string
}

type QuizHomeData struct {
	Meta       PageMeta
	Categories []QuizCategory
}

type QuizCategoryData struct {
	Meta       PageMeta
	Category   QuizCategory
	Categories []QuizCategory
}
