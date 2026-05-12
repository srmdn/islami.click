package handler

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

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
	quizMeta := pageMeta(r, "Quiz Islami", "Uji pengetahuan Islammu dengan kuis interaktif: aqidah, Al-Quran, hadits, sirah, fiqh, dan lebih banyak lagi.")
	quizMeta.JSONLD = breadcrumbJSONLD(homeCrumb(), crumb(2, "Quiz Islami", siteURL+"/quiz"))
	h.render(w, "quiz.html", model.QuizHomeData{
		Meta:       quizMeta,
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

	catMeta := pageMeta(r, "Quiz "+cat.Label, "Uji pengetahuan Islammu tentang "+cat.Label+": pilih tingkat kesulitan dan mulai kuis.")
	catMeta.JSONLD = breadcrumbJSONLD(homeCrumb(), crumb(2, "Quiz Islami", siteURL+"/quiz"), crumb(3, cat.Label, siteURL+"/quiz/"+slug))
	h.render(w, "quiz-category.html", model.QuizCategoryData{
		Meta:       catMeta,
		Category:   *cat,
		Categories: cats,
	})
}

func (h *Handler) QuizStartAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug, ok := quizSlugFromAPIPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	var req struct {
		PlayerName string `json:"player_name"`
		Difficulty string `json:"difficulty"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.PlayerName)
	if len(name) == 0 || len(name) > 30 {
		http.Error(w, "player name must be 1-30 characters", http.StatusBadRequest)
		return
	}
	if !validQuizDifficulty(req.Difficulty) {
		http.Error(w, "invalid difficulty", http.StatusBadRequest)
		return
	}

	limit := quizQuestionsPerDifficulty[req.Difficulty]
	session, questions, err := h.contentStore.CreateQuizSession(r.Context(), slug, req.Difficulty, name, limit, time.Now())
	if err != nil {
		log.Printf("quiz start %s/%s: %v", slug, req.Difficulty, err)
		http.Error(w, "Failed to start quiz", http.StatusInternalServerError)
		return
	}
	if len(questions) == 0 {
		http.Error(w, "Failed to start quiz", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_token":   session.Token,
		"question_number": 1,
		"total_questions": len(questions),
		"question":        questions[0],
	})
}

func (h *Handler) QuizAnswerAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug, ok := quizSlugFromAPIPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	var req struct {
		SessionToken string `json:"session_token"`
		QuestionID   int    `json:"question_id"`
		Selected     int    `json:"selected"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.SessionToken) == "" {
		http.Error(w, "session token is required", http.StatusBadRequest)
		return
	}

	session, err := h.contentStore.QuizSessionByToken(r.Context(), strings.TrimSpace(req.SessionToken), slug)
	if err != nil {
		log.Printf("quiz session lookup %s: %v", slug, err)
		http.Error(w, "failed to process quiz", http.StatusInternalServerError)
		return
	}
	if session == nil {
		http.Error(w, "quiz session not found", http.StatusNotFound)
		return
	}
	if session.Status != "active" {
		http.Error(w, "quiz session is not active", http.StatusConflict)
		return
	}
	now := time.Now().UTC()
	if now.After(session.ExpiresAt) {
		if err := h.contentStore.ExpireQuizSession(r.Context(), session.ID); err != nil {
			log.Printf("quiz expire session %d: %v", session.ID, err)
		}
		http.Error(w, "quiz session expired", http.StatusGone)
		return
	}

	current, err := h.contentStore.CurrentQuizSessionQuestion(r.Context(), session.ID)
	if err != nil {
		log.Printf("quiz current question %d: %v", session.ID, err)
		http.Error(w, "failed to process quiz", http.StatusInternalServerError)
		return
	}
	if current == nil {
		http.Error(w, "quiz session is complete", http.StatusConflict)
		return
	}
	if current.QuestionID != req.QuestionID {
		http.Error(w, "question does not match active session", http.StatusConflict)
		return
	}

	selected := req.Selected
	if selected < -1 || selected > 3 {
		http.Error(w, "invalid answer selection", http.StatusBadRequest)
		return
	}

	elapsed := now.Sub(current.PresentedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > quizSessionTimeout() {
		selected = -1
	}
	isCorrect := selected == current.Answer
	scoreAwarded := 0
	if isCorrect {
		timeLeft := quizTimerSeconds - int(math.Floor(elapsed.Seconds()))
		if timeLeft < 0 {
			timeLeft = 0
		}
		if timeLeft > quizTimerSeconds {
			timeLeft = quizTimerSeconds
		}
		timeBonus := int(math.Round(float64(timeLeft) / float64(quizTimerSeconds) * float64(quizTimeBonusMax)))
		scoreAwarded = quizScorePerCorrect + timeBonus
	}

	if err := h.contentStore.RecordQuizAnswer(r.Context(), session.ID, current.QuestionID, selected, scoreAwarded, isCorrect, now); err != nil {
		log.Printf("quiz answer %d/%d: %v", session.ID, current.QuestionID, err)
		http.Error(w, "failed to process quiz", http.StatusInternalServerError)
		return
	}

	if current.Position+1 >= session.QuestionCount {
		h.quizFinishSession(w, r, session, slug, now)
		return
	}

	nextQuestion, err := h.contentStore.NextQuizSessionQuestion(r.Context(), session.ID)
	if err != nil {
		log.Printf("quiz next question %d: %v", session.ID, err)
		http.Error(w, "failed to process quiz", http.StatusInternalServerError)
		return
	}
	if nextQuestion == nil {
		http.Error(w, "next question not found", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"done":            false,
		"question_number": nextQuestion.Position + 1,
		"total_questions": session.QuestionCount,
		"question": model.QuizQuestionPublic{
			ID:       nextQuestion.QuestionID,
			Question: nextQuestion.Question,
			Options:  nextQuestion.Options,
		},
	})
}

func (h *Handler) QuizLeaderboardAPI(w http.ResponseWriter, r *http.Request) {
	slug, ok := quizSlugFromAPIPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
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

	writeJSON(w, http.StatusOK, leaderboardOut(scores))
}

func (h *Handler) quizFinishSession(w http.ResponseWriter, r *http.Request, session *model.QuizSession, slug string, now time.Time) {
	if err := h.contentStore.CompleteQuizSession(r.Context(), session.ID, now); err != nil {
		log.Printf("quiz complete session %d: %v", session.ID, err)
		http.Error(w, "failed to save score", http.StatusInternalServerError)
		return
	}

	results, err := h.contentStore.QuizSessionResults(r.Context(), session.ID)
	if err != nil {
		log.Printf("quiz session results %d: %v", session.ID, err)
		http.Error(w, "failed to load results", http.StatusInternalServerError)
		return
	}

	var score int
	var correct int
	resultOut := make([]map[string]interface{}, 0, len(results))
	for _, result := range results {
		score += result.ScoreAwarded
		if result.IsCorrect != nil && *result.IsCorrect {
			correct++
		}
		selected := -1
		if result.SelectedIndex != nil {
			selected = *result.SelectedIndex
		}
		resultOut = append(resultOut, map[string]interface{}{
			"question_id":   result.QuestionID,
			"question":      result.Question,
			"options":       result.Options,
			"selected":      selected,
			"correct":       result.IsCorrect != nil && *result.IsCorrect,
			"correct_index": result.Answer,
			"explanation":   result.Explanation,
			"score_awarded": result.ScoreAwarded,
		})
	}

	if err := h.contentStore.SaveQuizScore(r.Context(), model.QuizScore{
		CategorySlug: slug,
		PlayerName:   session.PlayerName,
		Score:        score,
		CorrectCount: correct,
		TotalCount:   len(results),
		Difficulty:   session.Difficulty,
	}); err != nil {
		log.Printf("quiz submit save score %s: %v", slug, err)
		http.Error(w, "failed to save score", http.StatusInternalServerError)
		return
	}

	leaderboard, err := h.contentStore.QuizLeaderboard(r.Context(), slug, session.Difficulty, quizLeaderboardLimit)
	if err != nil {
		log.Printf("quiz submit leaderboard %s/%s: %v", slug, session.Difficulty, err)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"done":        true,
		"score":       score,
		"correct":     correct,
		"total":       len(results),
		"results":     resultOut,
		"leaderboard": leaderboardOut(leaderboard),
	})
}

func quizSlugFromAPIPath(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 || parts[0] != "api" || parts[1] != "quiz" {
		return "", false
	}
	return parts[2], true
}

func validQuizDifficulty(difficulty string) bool {
	return difficulty == "basic" || difficulty == "intermediate" || difficulty == "advanced"
}

func quizSessionTimeout() time.Duration {
	return time.Duration(quizTimerSeconds) * time.Second
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
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
