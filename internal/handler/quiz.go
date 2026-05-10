package handler

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/srmdn/islami.click/internal/model"
)

const (
	quizLeaderboardLimit = 10
	quizTimerSeconds     = 30
	quizScorePerCorrect  = 10
	quizTimeBonusMax     = 10
)

var quizQuestionsPerDifficulty = map[string]int{
	"basic":        10,
	"intermediate": 15,
	"advanced":     15,
}

func (h *Handler) QuizHome(w http.ResponseWriter, r *http.Request) {
	cats, err := h.contentStore.QuizCategories(r.Context())
	if err != nil {
		log.Printf("quiz home: %v", err)
		http.Error(w, "Failed to load quiz categories", http.StatusInternalServerError)
		return
	}
	h.render(w, "quiz.html", model.QuizHomeData{
		Meta:       pageMeta(r, "Quiz Islami", "Uji pengetahuan Islammu dengan kuis interaktif: aqidah, Al-Quran, hadits, sirah, fiqh, dan lebih banyak lagi."),
		Categories: cats,
	})
}

func (h *Handler) QuizCategory(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/quiz/")
	slug = strings.TrimSpace(slug)
	if slug == "" {
		http.Redirect(w, r, "/quiz", http.StatusSeeOther)
		return
	}

	cat, err := h.contentStore.QuizCategory(r.Context(), slug)
	if err != nil {
		log.Printf("quiz category %s: %v", slug, err)
		http.Error(w, "Failed to load quiz", http.StatusInternalServerError)
		return
	}
	if cat == nil {
		http.NotFound(w, r)
		return
	}

	cats, err := h.contentStore.QuizCategories(r.Context())
	if err != nil {
		log.Printf("quiz categories for %s: %v", slug, err)
		http.Error(w, "Failed to load quiz", http.StatusInternalServerError)
		return
	}

	h.render(w, "quiz-category.html", model.QuizCategoryData{
		Meta:       pageMeta(r, "Quiz "+cat.Label, "Uji pengetahuan Islammu tentang "+cat.Label+": pilih tingkat kesulitan dan mulai kuis."),
		Category:   *cat,
		Categories: cats,
	})
}

func (h *Handler) QuizQuestionsAPI(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.NotFound(w, r)
		return
	}
	slug := parts[2]
	difficulty := r.URL.Query().Get("difficulty")
	if difficulty == "" {
		difficulty = "basic"
	}
	if difficulty != "basic" && difficulty != "intermediate" && difficulty != "advanced" {
		http.Error(w, "invalid difficulty", http.StatusBadRequest)
		return
	}

	limit := quizQuestionsPerDifficulty[difficulty]
	questions, err := h.contentStore.QuizQuestions(r.Context(), slug, difficulty, limit)
	if err != nil {
		log.Printf("quiz questions api %s/%s: %v", slug, difficulty, err)
		http.Error(w, "Failed to load questions", http.StatusInternalServerError)
		return
	}

	type questionOut struct {
		ID       int      `json:"id"`
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}

	out := make([]questionOut, len(questions))
	for i, q := range questions {
		out[i] = questionOut{
			ID:       q.ID,
			Question: q.Question,
			Options:  q.Options,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Printf("quiz questions encode: %v", err)
	}
}

func (h *Handler) QuizLeaderboardAPI(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.NotFound(w, r)
		return
	}
	slug := parts[2]
	difficulty := r.URL.Query().Get("difficulty")
	if difficulty == "" {
		difficulty = "basic"
	}

	scores, err := h.contentStore.QuizLeaderboard(r.Context(), slug, difficulty, quizLeaderboardLimit)
	if err != nil {
		log.Printf("quiz leaderboard api %s/%s: %v", slug, difficulty, err)
		http.Error(w, "Failed to load leaderboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(leaderboardOut(scores)); err != nil {
		log.Printf("quiz leaderboard encode: %v", err)
	}
}

func (h *Handler) QuizSubmitAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.NotFound(w, r)
		return
	}
	slug := parts[2]

	var req struct {
		PlayerName string `json:"player_name"`
		Difficulty string `json:"difficulty"`
		Answers    []struct {
			QuestionID int `json:"question_id"`
			Selected   int `json:"selected"`
			TimeLeft   int `json:"time_left"`
		} `json:"answers"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.PlayerName)
	if len(name) == 0 || len(name) > 30 {
		http.Error(w, "player name must be 1-30 characters", http.StatusBadRequest)
		return
	}
	if req.Difficulty != "basic" && req.Difficulty != "intermediate" && req.Difficulty != "advanced" {
		http.Error(w, "invalid difficulty", http.StatusBadRequest)
		return
	}
	maxQ := quizQuestionsPerDifficulty[req.Difficulty]
	if len(req.Answers) == 0 || len(req.Answers) > maxQ {
		http.Error(w, "invalid number of answers", http.StatusBadRequest)
		return
	}

	// Deduplicate by question ID (take first occurrence)
	seen := make(map[int]bool, len(req.Answers))
	answers := req.Answers[:0]
	for _, a := range req.Answers {
		if !seen[a.QuestionID] {
			seen[a.QuestionID] = true
			answers = append(answers, a)
		}
	}

	answerKeys, err := h.contentStore.QuizAnswerKeys(r.Context(), slug, req.Difficulty)
	if err != nil {
		log.Printf("quiz submit answer keys %s/%s: %v", slug, req.Difficulty, err)
		http.Error(w, "failed to process quiz", http.StatusInternalServerError)
		return
	}

	type resultOut struct {
		QuestionID   int    `json:"question_id"`
		Correct      bool   `json:"correct"`
		CorrectIndex int    `json:"correct_index"`
		Explanation  string `json:"explanation"`
	}

	var score, correct int
	results := make([]resultOut, 0, len(answers))

	for _, ans := range answers {
		key, ok := answerKeys[ans.QuestionID]
		if !ok {
			continue
		}
		timeLeft := ans.TimeLeft
		if timeLeft < 0 {
			timeLeft = 0
		}
		if timeLeft > quizTimerSeconds {
			timeLeft = quizTimerSeconds
		}
		isCorrect := ans.Selected == key.CorrectIndex
		if isCorrect {
			correct++
			timeBonus := int(math.Round(float64(timeLeft) / float64(quizTimerSeconds) * float64(quizTimeBonusMax)))
			score += quizScorePerCorrect + timeBonus
		}
		results = append(results, resultOut{
			QuestionID:   ans.QuestionID,
			Correct:      isCorrect,
			CorrectIndex: key.CorrectIndex,
			Explanation:  key.Explanation,
		})
	}
	total := len(results)

	if err := h.contentStore.SaveQuizScore(r.Context(), model.QuizScore{
		CategorySlug: slug,
		PlayerName:   name,
		Score:        score,
		CorrectCount: correct,
		TotalCount:   total,
		Difficulty:   req.Difficulty,
	}); err != nil {
		log.Printf("quiz submit save score %s: %v", slug, err)
		http.Error(w, "failed to save score", http.StatusInternalServerError)
		return
	}

	leaderboard, err := h.contentStore.QuizLeaderboard(r.Context(), slug, req.Difficulty, quizLeaderboardLimit)
	if err != nil {
		log.Printf("quiz submit leaderboard %s/%s: %v", slug, req.Difficulty, err)
	}

	resp := struct {
		Score       int         `json:"score"`
		Correct     int         `json:"correct"`
		Total       int         `json:"total"`
		Results     []resultOut `json:"results"`
		Leaderboard interface{} `json:"leaderboard"`
	}{
		Score:       score,
		Correct:     correct,
		Total:       total,
		Results:     results,
		Leaderboard: leaderboardOut(leaderboard),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("quiz submit encode: %v", err)
	}
}

func leaderboardOut(scores []model.QuizScore) []map[string]interface{} {
	out := make([]map[string]interface{}, len(scores))
	for i, s := range scores {
		out[i] = map[string]interface{}{
			"rank":        i + 1,
			"player_name": s.PlayerName,
			"score":       s.Score,
			"correct":     s.CorrectCount,
			"total":       s.TotalCount,
			"played_at":   s.PlayedAt,
		}
	}
	return out
}
