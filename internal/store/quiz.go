package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/srmdn/islami.click/internal/model"
)

type quizQuestionJSON struct {
	Q           string   `json:"q"`
	Options     []string `json:"options"`
	Answer      int      `json:"answer"`
	Explanation string   `json:"explanation"`
}

type quizCategoryJSON struct {
	Slug         string `json:"slug"`
	Label        string `json:"label"`
	Description  string `json:"description"`
	DisplayOrder int    `json:"display_order"`
	Questions    struct {
		Basic        []quizQuestionJSON `json:"basic"`
		Intermediate []quizQuestionJSON `json:"intermediate"`
		Advanced     []quizQuestionJSON `json:"advanced"`
	} `json:"questions"`
}

func (s *Store) SeedQuiz(ctx context.Context, contentFS embed.FS) error {
	entries, err := contentFS.ReadDir("content/quiz")
	if err != nil {
		return fmt.Errorf("read quiz dir: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || len(name) < 6 || name[len(name)-5:] != ".json" {
			continue
		}

		path := "content/quiz/" + name
		data, err := contentFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		var cat quizCategoryJSON
		if err := json.Unmarshal(data, &cat); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		sum := checksum(data)
		var existingChecksum string
		dbErr := s.db.QueryRowContext(ctx,
			"SELECT source_checksum FROM quiz_categories WHERE slug = ?", cat.Slug,
		).Scan(&existingChecksum)
		if dbErr == nil && existingChecksum == sum {
			continue
		}

		if err := s.seedQuizCategory(ctx, cat, sum); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) seedQuizCategory(ctx context.Context, cat quizCategoryJSON, sum string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin quiz seed tx for %s: %w", cat.Slug, err)
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM quiz_categories WHERE slug = ?", cat.Slug); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("clear quiz category %s: %w", cat.Slug, err)
	}

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO quiz_categories (slug, label, description, source_checksum, display_order) VALUES (?, ?, ?, ?, ?)",
		cat.Slug, cat.Label, cat.Description, sum, cat.DisplayOrder,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("insert quiz category %s: %w", cat.Slug, err)
	}

	difficulties := []struct {
		name      string
		questions []quizQuestionJSON
	}{
		{"basic", cat.Questions.Basic},
		{"intermediate", cat.Questions.Intermediate},
		{"advanced", cat.Questions.Advanced},
	}

	for _, d := range difficulties {
		for i, q := range d.questions {
			opts, err := json.Marshal(q.Options)
			if err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("marshal options %s/%s/%d: %w", cat.Slug, d.name, i, err)
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO quiz_questions (category_slug, difficulty, question, options_json, answer_index, explanation, display_order)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				cat.Slug, d.name, q.Q, string(opts), q.Answer, q.Explanation, i+1,
			); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("insert quiz question %s/%s/%d: %w", cat.Slug, d.name, i, err)
			}
		}
	}

	return tx.Commit()
}

func (s *Store) QuizCategories(ctx context.Context) ([]model.QuizCategory, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT slug, label, description, display_order FROM quiz_categories ORDER BY display_order",
	)
	if err != nil {
		return nil, fmt.Errorf("quiz categories: %w", err)
	}
	defer rows.Close()

	var cats []model.QuizCategory
	for rows.Next() {
		var c model.QuizCategory
		if err := rows.Scan(&c.Slug, &c.Label, &c.Description, &c.Order); err != nil {
			return nil, fmt.Errorf("scan quiz category: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (s *Store) QuizCategory(ctx context.Context, slug string) (*model.QuizCategory, error) {
	var c model.QuizCategory
	err := s.db.QueryRowContext(ctx,
		"SELECT slug, label, description, display_order FROM quiz_categories WHERE slug = ?", slug,
	).Scan(&c.Slug, &c.Label, &c.Description, &c.Order)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("quiz category %s: %w", slug, err)
	}
	return &c, nil
}

func (s *Store) QuizQuestions(ctx context.Context, categorySlug, difficulty string, limit int) ([]model.QuizQuestion, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, question, options_json, answer_index, explanation
		 FROM quiz_questions
		 WHERE category_slug = ? AND difficulty = ?
		 ORDER BY RANDOM()
		 LIMIT ?`,
		categorySlug, difficulty, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("quiz questions: %w", err)
	}
	defer rows.Close()

	var questions []model.QuizQuestion
	for rows.Next() {
		var q model.QuizQuestion
		var optsJSON string
		if err := rows.Scan(&q.ID, &q.Question, &optsJSON, &q.Answer, &q.Explanation); err != nil {
			return nil, fmt.Errorf("scan quiz question: %w", err)
		}
		if err := json.Unmarshal([]byte(optsJSON), &q.Options); err != nil {
			return nil, fmt.Errorf("parse options: %w", err)
		}
		q.Category = categorySlug
		q.Difficulty = difficulty
		questions = append(questions, q)
	}
	return questions, rows.Err()
}

func (s *Store) QuizAnswerKeys(ctx context.Context, categorySlug, difficulty string) (map[int]model.QuizAnswerKey, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, answer_index, explanation FROM quiz_questions WHERE category_slug = ? AND difficulty = ?`,
		categorySlug, difficulty,
	)
	if err != nil {
		return nil, fmt.Errorf("quiz answer keys: %w", err)
	}
	defer rows.Close()

	keys := make(map[int]model.QuizAnswerKey)
	for rows.Next() {
		var id, answerIndex int
		var explanation string
		if err := rows.Scan(&id, &answerIndex, &explanation); err != nil {
			return nil, fmt.Errorf("scan quiz answer key: %w", err)
		}
		keys[id] = model.QuizAnswerKey{CorrectIndex: answerIndex, Explanation: explanation}
	}
	return keys, rows.Err()
}

func (s *Store) QuizLeaderboard(ctx context.Context, categorySlug, difficulty string, limit int) ([]model.QuizScore, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, player_name, score, correct_count, total_count, difficulty, played_at
		 FROM quiz_scores
		 WHERE category_slug = ? AND difficulty = ?
		 ORDER BY score DESC, played_at ASC
		 LIMIT ?`,
		categorySlug, difficulty, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("quiz leaderboard: %w", err)
	}
	defer rows.Close()

	var scores []model.QuizScore
	for rows.Next() {
		var sc model.QuizScore
		if err := rows.Scan(&sc.ID, &sc.PlayerName, &sc.Score, &sc.CorrectCount, &sc.TotalCount, &sc.Difficulty, &sc.PlayedAt); err != nil {
			return nil, fmt.Errorf("scan quiz score: %w", err)
		}
		sc.CategorySlug = categorySlug
		scores = append(scores, sc)
	}
	return scores, rows.Err()
}

func (s *Store) SaveQuizScore(ctx context.Context, sc model.QuizScore) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO quiz_scores (category_slug, player_name, score, correct_count, total_count, difficulty)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		sc.CategorySlug, sc.PlayerName, sc.Score, sc.CorrectCount, sc.TotalCount, sc.Difficulty,
	)
	if err != nil {
		return fmt.Errorf("save quiz score: %w", err)
	}
	return nil
}
