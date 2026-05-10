package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/srmdn/islami.click/internal/model"
)

const quizLeaderboardLimit = 10

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

	questions, err := h.contentStore.QuizQuestions(r.Context(), slug, difficulty)
	if err != nil {
		log.Printf("quiz questions api %s/%s: %v", slug, difficulty, err)
		http.Error(w, "Failed to load questions", http.StatusInternalServerError)
		return
	}

	type questionOut struct {
		ID          int      `json:"id"`
		Question    string   `json:"question"`
		Options     []string `json:"options"`
		Answer      int      `json:"answer"`
		Explanation string   `json:"explanation"`
	}

	out := make([]questionOut, len(questions))
	for i, q := range questions {
		out[i] = questionOut{
			ID:          q.ID,
			Question:    q.Question,
			Options:     q.Options,
			Answer:      q.Answer,
			Explanation: q.Explanation,
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

	type scoreOut struct {
		Rank       int    `json:"rank"`
		PlayerName string `json:"player_name"`
		Score      int    `json:"score"`
		Correct    int    `json:"correct"`
		Total      int    `json:"total"`
		PlayedAt   string `json:"played_at"`
	}

	out := make([]scoreOut, len(scores))
	for i, s := range scores {
		out[i] = scoreOut{
			Rank:       i + 1,
			PlayerName: s.PlayerName,
			Score:      s.Score,
			Correct:    s.CorrectCount,
			Total:      s.TotalCount,
			PlayedAt:   s.PlayedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Printf("quiz leaderboard encode: %v", err)
	}
}

func (h *Handler) QuizScoreAPI(w http.ResponseWriter, r *http.Request) {
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
		Score      int    `json:"score"`
		Correct    int    `json:"correct"`
		Total      int    `json:"total"`
		Difficulty string `json:"difficulty"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
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
	if req.Score < 0 || req.Correct < 0 || req.Total < 0 || req.Total > 20 {
		http.Error(w, "invalid score values", http.StatusBadRequest)
		return
	}

	sc := model.QuizScore{
		CategorySlug: slug,
		PlayerName:   name,
		Score:        req.Score,
		CorrectCount: req.Correct,
		TotalCount:   req.Total,
		Difficulty:   req.Difficulty,
	}
	if err := h.contentStore.SaveQuizScore(r.Context(), sc); err != nil {
		log.Printf("save quiz score %s: %v", slug, err)
		http.Error(w, "Failed to save score", http.StatusInternalServerError)
		return
	}

	scores, err := h.contentStore.QuizLeaderboard(r.Context(), slug, req.Difficulty, quizLeaderboardLimit)
	if err != nil {
		log.Printf("leaderboard after save %s: %v", slug, err)
		http.Error(w, "Failed to load leaderboard", http.StatusInternalServerError)
		return
	}

	type scoreOut struct {
		Rank       int    `json:"rank"`
		PlayerName string `json:"player_name"`
		Score      int    `json:"score"`
		Correct    int    `json:"correct"`
		Total      int    `json:"total"`
		PlayedAt   string `json:"played_at"`
	}
	out := make([]scoreOut, len(scores))
	for i, s := range scores {
		out[i] = scoreOut{
			Rank:       i + 1,
			PlayerName: s.PlayerName,
			Score:      s.Score,
			Correct:    s.CorrectCount,
			Total:      s.TotalCount,
			PlayedAt:   s.PlayedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Printf("quiz score response encode: %v", err)
	}
}
