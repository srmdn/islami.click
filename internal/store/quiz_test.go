package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/model"
	"github.com/srmdn/islami.click/internal/store"
)

func TestQuizLeaderboardIsScopedToMonth(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "quiz.db")

	contentStore, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer contentStore.Close()

	err = contentStore.SaveQuizScore(ctx, model.QuizScore{
		CategorySlug: "aqidah",
		PlayerName:   "LastMonth",
		Score:        200,
		CorrectCount: 10,
		TotalCount:   10,
		Difficulty:   "basic",
		PlayedMonth:  "2026-04",
	})
	if err != nil {
		t.Fatalf("save old month score: %v", err)
	}

	err = contentStore.SaveQuizScore(ctx, model.QuizScore{
		CategorySlug: "aqidah",
		PlayerName:   "ThisMonth",
		Score:        150,
		CorrectCount: 9,
		TotalCount:   10,
		Difficulty:   "basic",
		PlayedMonth:  "2026-05",
	})
	if err != nil {
		t.Fatalf("save current month score: %v", err)
	}

	scores, err := contentStore.QuizLeaderboard(ctx, "aqidah", "basic", "2026-05", 10)
	if err != nil {
		t.Fatalf("load leaderboard: %v", err)
	}
	if len(scores) != 1 {
		t.Fatalf("leaderboard count = %d, want 1", len(scores))
	}
	if scores[0].PlayerName != "ThisMonth" {
		t.Fatalf("leaderboard player = %q, want %q", scores[0].PlayerName, "ThisMonth")
	}
	if scores[0].PlayedMonth != "2026-05" {
		t.Fatalf("leaderboard month = %q, want %q", scores[0].PlayedMonth, "2026-05")
	}
}

func TestCleanupQuizSessionsRemovesOldState(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "quiz.db")

	contentStore, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer contentStore.Close()

	now := time.Now().UTC()
	oldSession, _, err := contentStore.CreateQuizSession(ctx, "aqidah", "basic", "Old", 10, now.Add(-48*time.Hour))
	if err != nil {
		t.Fatalf("create old session: %v", err)
	}
	recentSession, _, err := contentStore.CreateQuizSession(ctx, "aqidah", "basic", "Recent", 10, now)
	if err != nil {
		t.Fatalf("create recent session: %v", err)
	}

	if err := contentStore.CleanupQuizSessions(ctx, now.Add(-24*time.Hour)); err != nil {
		t.Fatalf("cleanup sessions: %v", err)
	}
	oldAfterCleanup, err := contentStore.QuizSessionByToken(ctx, oldSession.Token, "aqidah")
	if err != nil {
		t.Fatalf("lookup old session: %v", err)
	}
	if oldAfterCleanup != nil {
		t.Fatal("old session still exists after cleanup")
	}
	recentAfterCleanup, err := contentStore.QuizSessionByToken(ctx, recentSession.Token, "aqidah")
	if err != nil {
		t.Fatalf("lookup recent session: %v", err)
	}
	if recentAfterCleanup == nil {
		t.Fatal("recent session was removed by cleanup")
	}
}
