CREATE TABLE IF NOT EXISTS focus_sessions (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    app_class         TEXT NOT NULL,
    title             TEXT NOT NULL DEFAULT '',
    pid               INTEGER,
    started_at        INTEGER NOT NULL,
    last_seen_at      INTEGER NOT NULL,
    ended_at          INTEGER,
    duration_seconds  INTEGER
);

CREATE INDEX IF NOT EXISTS idx_sessions_started_at ON focus_sessions(started_at);
CREATE INDEX IF NOT EXISTS idx_sessions_app_class   ON focus_sessions(app_class);
