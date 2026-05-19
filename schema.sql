-- Canonical schema. Mirror this file to backend/internal/db/schema.sql when changed
-- (go:embed cannot cross directory boundaries; both files must stay byte-identical).

CREATE TABLE IF NOT EXISTS clients (
    id             TEXT    PRIMARY KEY,
    flush_interval INTEGER NOT NULL,
    tracked_events TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id   TEXT    NOT NULL UNIQUE,
    client_id  TEXT    NOT NULL,
    session_id TEXT    NOT NULL,
    event_type TEXT    NOT NULL,
    timestamp  TEXT    NOT NULL,
    server_ts  TEXT    NOT NULL,
    metadata   TEXT
);

CREATE INDEX IF NOT EXISTS idx_events_server_ts         ON events (server_ts);
CREATE INDEX IF NOT EXISTS idx_events_client_server_ts  ON events (client_id, server_ts);
CREATE INDEX IF NOT EXISTS idx_events_session_timestamp ON events (session_id, timestamp DESC);
