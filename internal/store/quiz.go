package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

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

const quizSessionTTL = 15 * time.Minute

func quizSessionToken() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("read random token: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

func (s *Store) CreateQuizSession(ctx context.Context, categorySlug, difficulty, playerName string, limit int, now time.Time) (*model.QuizSession, []model.QuizQuestionPublic, error) {
	questions, err := s.QuizQuestions(ctx, categorySlug, difficulty, limit)
	if err != nil {
		return nil, nil, err
	}
	if len(questions) == 0 {
		return nil, nil, fmt.Errorf("no quiz questions for %s/%s", categorySlug, difficulty)
	}

	token, err := quizSessionToken()
	if err != nil {
		return nil, nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("begin quiz session tx: %w", err)
	}

	expiresAt := now.Add(quizSessionTTL).UTC()
	result, err := tx.ExecContext(ctx,
		`INSERT INTO quiz_sessions (token, category_slug, player_name, difficulty, status, question_count, current_index, started_at, expires_at)
		 VALUES (?, ?, ?, ?, 'active', ?, 0, ?, ?)`,
		token, categorySlug, playerName, difficulty, len(questions), now.UTC().Format(time.RFC3339Nano), expiresAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("insert quiz session: %w", err)
	}

	sessionID, err := result.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("read quiz session id: %w", err)
	}

	publicQuestions := make([]model.QuizQuestionPublic, len(questions))
	for i, q := range questions {
		presentedAt := now.UTC()
		if i > 0 {
			presentedAt = time.Time{}
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO quiz_session_questions (session_id, position, question_id, presented_at)
			 VALUES (?, ?, ?, ?)`,
			sessionID, i, q.ID, presentedAt.Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return nil, nil, fmt.Errorf("insert session question %d: %w", i, err)
		}
		publicQuestions[i] = model.QuizQuestionPublic{
			ID:       q.ID,
			Question: q.Question,
			Options:  q.Options,
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit quiz session: %w", err)
	}

	return &model.QuizSession{
		ID:            int(sessionID),
		Token:         token,
		CategorySlug:  categorySlug,
		PlayerName:    playerName,
		Difficulty:    difficulty,
		Status:        "active",
		QuestionCount: len(questions),
		CurrentIndex:  0,
		StartedAt:     now.UTC(),
		ExpiresAt:     expiresAt,
	}, publicQuestions, nil
}

func (s *Store) QuizSessionByToken(ctx context.Context, token, categorySlug string) (*model.QuizSession, error) {
	var session model.QuizSession
	var startedAt string
	var expiresAt string
	var completedAt sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, token, category_slug, player_name, difficulty, status, question_count, current_index, started_at, expires_at, completed_at
		 FROM quiz_sessions
		 WHERE token = ? AND category_slug = ?`,
		token, categorySlug,
	).Scan(
		&session.ID,
		&session.Token,
		&session.CategorySlug,
		&session.PlayerName,
		&session.Difficulty,
		&session.Status,
		&session.QuestionCount,
		&session.CurrentIndex,
		&startedAt,
		&expiresAt,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("quiz session lookup: %w", err)
	}
	session.StartedAt, err = time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return nil, fmt.Errorf("parse session started_at: %w", err)
	}
	session.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("parse session expires_at: %w", err)
	}
	if completedAt.Valid {
		ts, err := time.Parse(time.RFC3339Nano, completedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse session completed_at: %w", err)
		}
		session.CompletedAt = &ts
	}
	return &session, nil
}

func (s *Store) ExpireQuizSession(ctx context.Context, sessionID int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE quiz_sessions
		 SET status = 'expired'
		 WHERE id = ? AND status = 'active'`,
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("expire quiz session: %w", err)
	}
	return nil
}

func (s *Store) CurrentQuizSessionQuestion(ctx context.Context, sessionID int) (*model.QuizSessionQuestion, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT qsq.session_id, qsq.position, qsq.question_id, qq.question, qq.options_json, qq.answer_index, qq.explanation,
		        qsq.presented_at, qsq.selected_index, qsq.answered_at, qsq.is_correct, qsq.score_awarded
		 FROM quiz_session_questions qsq
		 JOIN quiz_questions qq ON qq.id = qsq.question_id
		 JOIN quiz_sessions qs ON qs.id = qsq.session_id
		 WHERE qsq.session_id = ? AND qsq.position = qs.current_index`,
		sessionID,
	)
	return scanQuizSessionQuestion(row)
}

func (s *Store) NextQuizSessionQuestion(ctx context.Context, sessionID int) (*model.QuizSessionQuestion, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT qsq.session_id, qsq.position, qsq.question_id, qq.question, qq.options_json, qq.answer_index, qq.explanation,
		        qsq.presented_at, qsq.selected_index, qsq.answered_at, qsq.is_correct, qsq.score_awarded
		 FROM quiz_session_questions qsq
		 JOIN quiz_questions qq ON qq.id = qsq.question_id
		 JOIN quiz_sessions qs ON qs.id = qsq.session_id
		 WHERE qsq.session_id = ? AND qsq.position = qs.current_index`,
		sessionID,
	)
	return scanQuizSessionQuestion(row)
}

func scanQuizSessionQuestion(row *sql.Row) (*model.QuizSessionQuestion, error) {
	var q model.QuizSessionQuestion
	var optionsJSON string
	var presentedAt string
	var selectedIndex sql.NullInt64
	var answeredAt sql.NullString
	var isCorrect sql.NullInt64
	err := row.Scan(
		&q.SessionID,
		&q.Position,
		&q.QuestionID,
		&q.Question,
		&optionsJSON,
		&q.Answer,
		&q.Explanation,
		&presentedAt,
		&selectedIndex,
		&answeredAt,
		&isCorrect,
		&q.ScoreAwarded,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan quiz session question: %w", err)
	}
	if err := json.Unmarshal([]byte(optionsJSON), &q.Options); err != nil {
		return nil, fmt.Errorf("parse session question options: %w", err)
	}
	q.PresentedAt, err = time.Parse(time.RFC3339Nano, presentedAt)
	if err != nil {
		return nil, fmt.Errorf("parse session presented_at: %w", err)
	}
	if selectedIndex.Valid {
		value := int(selectedIndex.Int64)
		q.SelectedIndex = &value
	}
	if answeredAt.Valid {
		ts, err := time.Parse(time.RFC3339Nano, answeredAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse session answered_at: %w", err)
		}
		q.AnsweredAt = &ts
	}
	if isCorrect.Valid {
		value := isCorrect.Int64 != 0
		q.IsCorrect = &value
	}
	return &q, nil
}

func (s *Store) RecordQuizAnswer(ctx context.Context, sessionID, questionID, selectedIndex, scoreAwarded int, isCorrect bool, answeredAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin answer tx: %w", err)
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE quiz_session_questions
		 SET selected_index = ?, answered_at = ?, is_correct = ?, score_awarded = ?
		 WHERE session_id = ? AND question_id = ? AND answered_at IS NULL`,
		selectedIndex, answeredAt.UTC().Format(time.RFC3339Nano), boolToInt(isCorrect), scoreAwarded, sessionID, questionID,
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("update session answer: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("answer rows affected: %w", err)
	}
	if rows != 1 {
		_ = tx.Rollback()
		return fmt.Errorf("quiz answer was not recorded")
	}

	nextResult, err := tx.ExecContext(ctx,
		`UPDATE quiz_sessions
		 SET current_index = current_index + 1
		 WHERE id = ? AND status = 'active'`,
		sessionID,
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("advance session index: %w", err)
	}
	if rows, err = nextResult.RowsAffected(); err != nil || rows != 1 {
		_ = tx.Rollback()
		if err != nil {
			return fmt.Errorf("advance session rows affected: %w", err)
		}
		return fmt.Errorf("quiz session was not advanced")
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE quiz_session_questions
		 SET presented_at = ?
		 WHERE session_id = ? AND position = (
		     SELECT current_index FROM quiz_sessions WHERE id = ?
		 ) AND presented_at = ?`,
		answeredAt.UTC().Format(time.RFC3339Nano), sessionID, sessionID, time.Time{}.Format(time.RFC3339Nano),
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("present next session question: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit session answer: %w", err)
	}
	return nil
}

func (s *Store) CompleteQuizSession(ctx context.Context, sessionID int, completedAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE quiz_sessions
		 SET status = 'completed', completed_at = ?, current_index = question_count
		 WHERE id = ? AND status = 'active'`,
		completedAt.UTC().Format(time.RFC3339Nano), sessionID,
	)
	if err != nil {
		return fmt.Errorf("complete quiz session: %w", err)
	}
	return nil
}

func (s *Store) QuizSessionResults(ctx context.Context, sessionID int) ([]model.QuizSessionQuestion, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT qsq.session_id, qsq.position, qsq.question_id, qq.question, qq.options_json, qq.answer_index, qq.explanation,
		        qsq.presented_at, qsq.selected_index, qsq.answered_at, qsq.is_correct, qsq.score_awarded
		 FROM quiz_session_questions qsq
		 JOIN quiz_questions qq ON qq.id = qsq.question_id
		 WHERE qsq.session_id = ?
		 ORDER BY qsq.position`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("quiz session results: %w", err)
	}
	defer rows.Close()

	results := make([]model.QuizSessionQuestion, 0)
	for rows.Next() {
		var q model.QuizSessionQuestion
		var optionsJSON string
		var presentedAt string
		var selectedIndex sql.NullInt64
		var answeredAt sql.NullString
		var isCorrect sql.NullInt64
		if err := rows.Scan(
			&q.SessionID,
			&q.Position,
			&q.QuestionID,
			&q.Question,
			&optionsJSON,
			&q.Answer,
			&q.Explanation,
			&presentedAt,
			&selectedIndex,
			&answeredAt,
			&isCorrect,
			&q.ScoreAwarded,
		); err != nil {
			return nil, fmt.Errorf("scan session result: %w", err)
		}
		if err := json.Unmarshal([]byte(optionsJSON), &q.Options); err != nil {
			return nil, fmt.Errorf("parse session result options: %w", err)
		}
		q.PresentedAt, err = time.Parse(time.RFC3339Nano, presentedAt)
		if err != nil {
			return nil, fmt.Errorf("parse session result presented_at: %w", err)
		}
		if selectedIndex.Valid {
			value := int(selectedIndex.Int64)
			q.SelectedIndex = &value
		}
		if answeredAt.Valid {
			ts, err := time.Parse(time.RFC3339Nano, answeredAt.String)
			if err != nil {
				return nil, fmt.Errorf("parse session result answered_at: %w", err)
			}
			q.AnsweredAt = &ts
		}
		if isCorrect.Valid {
			value := isCorrect.Int64 != 0
			q.IsCorrect = &value
		}
		results = append(results, q)
	}
	return results, rows.Err()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
