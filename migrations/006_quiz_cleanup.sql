CREATE INDEX IF NOT EXISTS idx_quiz_sessions_cleanup
    ON quiz_sessions(status, expires_at, completed_at);
