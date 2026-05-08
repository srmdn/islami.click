CREATE TABLE IF NOT EXISTS quiz_categories (
    slug TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source_checksum TEXT NOT NULL DEFAULT '',
    display_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS quiz_questions (
    id INTEGER PRIMARY KEY,
    category_slug TEXT NOT NULL REFERENCES quiz_categories(slug) ON DELETE CASCADE,
    difficulty TEXT NOT NULL CHECK (difficulty IN ('basic', 'intermediate', 'advanced')),
    question TEXT NOT NULL,
    options_json TEXT NOT NULL,
    answer_index INTEGER NOT NULL CHECK (answer_index BETWEEN 0 AND 3),
    explanation TEXT NOT NULL DEFAULT '',
    display_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS quiz_scores (
    id INTEGER PRIMARY KEY,
    category_slug TEXT NOT NULL REFERENCES quiz_categories(slug) ON DELETE CASCADE,
    player_name TEXT NOT NULL,
    score INTEGER NOT NULL DEFAULT 0,
    correct_count INTEGER NOT NULL DEFAULT 0,
    total_count INTEGER NOT NULL DEFAULT 0,
    difficulty TEXT NOT NULL CHECK (difficulty IN ('basic', 'intermediate', 'advanced')),
    played_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quiz_questions_category_difficulty
    ON quiz_questions(category_slug, difficulty, display_order);

CREATE INDEX IF NOT EXISTS idx_quiz_scores_leaderboard
    ON quiz_scores(category_slug, difficulty, score DESC, played_at);
