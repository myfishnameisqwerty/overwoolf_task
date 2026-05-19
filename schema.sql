CREATE TABLE IF NOT EXISTS clients (
    id             TEXT    PRIMARY KEY,
    flush_interval INTEGER NOT NULL,
    tracked_events TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id  TEXT    NOT NULL,
    session_id TEXT    NOT NULL,
    event_type TEXT    NOT NULL,
    timestamp  TEXT    NOT NULL,
    metadata   TEXT
);

CREATE INDEX IF NOT EXISTS idx_events_timestamp  ON events (timestamp);
CREATE INDEX IF NOT EXISTS idx_events_session_id ON events (session_id);
CREATE INDEX IF NOT EXISTS idx_events_client_id  ON events (client_id);

INSERT OR IGNORE INTO clients (id, flush_interval, tracked_events)
VALUES ('demo-client', 5000, 'pageview,click');
