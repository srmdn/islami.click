package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/store"
)

func TestQuizAPIServerManagedSessionFlow(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "quiz.db")

	contentStore, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer contentStore.Close()

	h := New(nil, nil, contentStore)

	startBody := []byte(`{"player_name":"Tester","difficulty":"basic"}`)
	startReq := httptest.NewRequest(http.MethodPost, "/api/quiz/aqidah/start", bytes.NewReader(startBody))
	startRec := httptest.NewRecorder()
	h.QuizStartAPI(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start status = %d, body = %s", startRec.Code, startRec.Body.String())
	}

	var startResp struct {
		SessionToken   string `json:"session_token"`
		QuestionNumber int    `json:"question_number"`
		TotalQuestions int    `json:"total_questions"`
		Question       struct {
			ID int `json:"id"`
		} `json:"question"`
	}
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	if startResp.SessionToken == "" {
		t.Fatal("expected session token")
	}
	if startResp.QuestionNumber != 1 {
		t.Fatalf("question number = %d, want 1", startResp.QuestionNumber)
	}
	if startResp.TotalQuestions != quizQuestionsPerDifficulty["basic"] {
		t.Fatalf("total questions = %d, want %d", startResp.TotalQuestions, quizQuestionsPerDifficulty["basic"])
	}

	answerKeys, err := contentStore.QuizAnswerKeys(ctx, "aqidah", "basic")
	if err != nil {
		t.Fatalf("load answer keys: %v", err)
	}

	currentQuestionID := startResp.Question.ID
	for i := 0; i < startResp.TotalQuestions; i++ {
		key, ok := answerKeys[currentQuestionID]
		if !ok {
			t.Fatalf("missing answer key for question %d", currentQuestionID)
		}
		answerBody, err := json.Marshal(map[string]interface{}{
			"session_token": startResp.SessionToken,
			"question_id":   currentQuestionID,
			"selected":      key.CorrectIndex,
		})
		if err != nil {
			t.Fatalf("marshal answer request: %v", err)
		}

		answerReq := httptest.NewRequest(http.MethodPost, "/api/quiz/aqidah/answer", bytes.NewReader(answerBody))
		answerRec := httptest.NewRecorder()
		h.QuizAnswerAPI(answerRec, answerReq)
		if answerRec.Code != http.StatusOK {
			t.Fatalf("answer %d status = %d, body = %s", i, answerRec.Code, answerRec.Body.String())
		}

		if i == startResp.TotalQuestions-1 {
			var finishResp struct {
				Done    bool `json:"done"`
				Score   int  `json:"score"`
				Correct int  `json:"correct"`
				Total   int  `json:"total"`
			}
			if err := json.Unmarshal(answerRec.Body.Bytes(), &finishResp); err != nil {
				t.Fatalf("decode finish response: %v", err)
			}
			if !finishResp.Done {
				t.Fatal("expected done response on final answer")
			}
			if finishResp.Total != startResp.TotalQuestions {
				t.Fatalf("finish total = %d, want %d", finishResp.Total, startResp.TotalQuestions)
			}
			if finishResp.Correct != startResp.TotalQuestions {
				t.Fatalf("finish correct = %d, want %d", finishResp.Correct, startResp.TotalQuestions)
			}
			if finishResp.Score < startResp.TotalQuestions*quizScorePerCorrect {
				t.Fatalf("finish score = %d, want at least %d", finishResp.Score, startResp.TotalQuestions*quizScorePerCorrect)
			}
			break
		}

		var nextResp struct {
			Done           bool `json:"done"`
			QuestionNumber int  `json:"question_number"`
			Question       struct {
				ID int `json:"id"`
			} `json:"question"`
		}
		if err := json.Unmarshal(answerRec.Body.Bytes(), &nextResp); err != nil {
			t.Fatalf("decode next response: %v", err)
		}
		if nextResp.Done {
			t.Fatalf("unexpected done response on question %d", i)
		}
		if nextResp.QuestionNumber != i+2 {
			t.Fatalf("next question number = %d, want %d", nextResp.QuestionNumber, i+2)
		}
		currentQuestionID = nextResp.Question.ID
	}
}

func TestQuizAPIRejectsForgedQuestionID(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "quiz.db")

	contentStore, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer contentStore.Close()

	h := New(nil, nil, contentStore)

	startBody := []byte(`{"player_name":"Tester","difficulty":"basic"}`)
	startReq := httptest.NewRequest(http.MethodPost, "/api/quiz/aqidah/start", bytes.NewReader(startBody))
	startRec := httptest.NewRecorder()
	h.QuizStartAPI(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start status = %d, body = %s", startRec.Code, startRec.Body.String())
	}

	var startResp struct {
		SessionToken string `json:"session_token"`
		Question     struct {
			ID int `json:"id"`
		} `json:"question"`
	}
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("decode start response: %v", err)
	}

	forgedQuestionID := startResp.Question.ID + 999999
	answerBody, err := json.Marshal(map[string]interface{}{
		"session_token": startResp.SessionToken,
		"question_id":   forgedQuestionID,
		"selected":      0,
	})
	if err != nil {
		t.Fatalf("marshal forged request: %v", err)
	}

	answerReq := httptest.NewRequest(http.MethodPost, "/api/quiz/aqidah/answer", bytes.NewReader(answerBody))
	answerRec := httptest.NewRecorder()
	h.QuizAnswerAPI(answerRec, answerReq)
	if answerRec.Code != http.StatusConflict {
		t.Fatalf("forged answer status = %d, want %d, body = %s", answerRec.Code, http.StatusConflict, answerRec.Body.String())
	}
}
