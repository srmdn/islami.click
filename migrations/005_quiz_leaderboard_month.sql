ALTER TABLE quiz_scores ADD COLUMN leaderboard_month TEXT NOT NULL DEFAULT '';

UPDATE quiz_scores
SET leaderboard_month = substr(played_at, 1, 7)
WHERE leaderboard_month = '';

DROP INDEX IF EXISTS idx_quiz_scores_leaderboard;

CREATE INDEX IF NOT EXISTS idx_quiz_scores_leaderboard_month
    ON quiz_scores(category_slug, difficulty, leaderboard_month, score DESC, played_at);
