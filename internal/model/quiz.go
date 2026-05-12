package model

import "time"

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

type QuizAnswerKey struct {
	CorrectIndex int
	Explanation  string
}

type QuizSession struct {
	ID            int
	Token         string
	CategorySlug  string
	PlayerName    string
	Difficulty    string
	Status        string
	QuestionCount int
	CurrentIndex  int
	StartedAt     time.Time
	ExpiresAt     time.Time
	CompletedAt   *time.Time
}

type QuizSessionQuestion struct {
	SessionID     int
	Position      int
	QuestionID    int
	Question      string
	Options       []string
	Answer        int
	Explanation   string
	PresentedAt   time.Time
	SelectedIndex *int
	AnsweredAt    *time.Time
	IsCorrect     *bool
	ScoreAwarded  int
}

type QuizQuestionPublic struct {
	ID       int      `json:"id"`
	Question string   `json:"question"`
	Options  []string `json:"options"`
}
