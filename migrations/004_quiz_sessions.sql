CREATE TABLE IF NOT EXISTS quiz_sessions (
    id INTEGER PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    category_slug TEXT NOT NULL REFERENCES quiz_categories(slug) ON DELETE CASCADE,
    player_name TEXT NOT NULL,
    difficulty TEXT NOT NULL CHECK (difficulty IN ('basic', 'intermediate', 'advanced')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'expired')),
    question_count INTEGER NOT NULL,
    current_index INTEGER NOT NULL DEFAULT 0,
    started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TEXT NOT NULL,
    completed_at TEXT
);

CREATE TABLE IF NOT EXISTS quiz_session_questions (
    session_id INTEGER NOT NULL REFERENCES quiz_sessions(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    question_id INTEGER NOT NULL REFERENCES quiz_questions(id) ON DELETE CASCADE,
    presented_at TEXT NOT NULL,
    selected_index INTEGER,
    answered_at TEXT,
    is_correct INTEGER,
    score_awarded INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (session_id, position),
    UNIQUE (session_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_quiz_sessions_lookup
    ON quiz_sessions(token, category_slug, difficulty, status);

CREATE INDEX IF NOT EXISTS idx_quiz_session_questions_session
    ON quiz_session_questions(session_id, position);
