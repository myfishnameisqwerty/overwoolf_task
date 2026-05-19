package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
	"tracker/internal/storage"
	"tracker/internal/types"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) GetClient(ctx context.Context, clientID string) (types.ClientConfig, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, flush_interval, tracked_events FROM clients WHERE id = ?`, clientID)

	var id, trackedRaw string
	var flushInterval int
	if err := row.Scan(&id, &flushInterval, &trackedRaw); err != nil {
		if err == sql.ErrNoRows {
			return types.ClientConfig{}, storage.ErrNotFound
		}
		return types.ClientConfig{}, err
	}

	events := []string{}
	for _, e := range strings.Split(trackedRaw, ",") {
		if t := strings.TrimSpace(e); t != "" {
			events = append(events, t)
		}
	}

	return types.ClientConfig{
		ClientID:      id,
		FlushInterval: flushInterval,
		TrackedEvents: events,
	}, nil
}

func (s *Store) InsertEvents(ctx context.Context, events []types.Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO events (client_id, session_id, event_type, timestamp, metadata)
		 VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		var meta *string
		if len(e.Metadata) > 0 && string(e.Metadata) != "null" {
			m := string(e.Metadata)
			meta = &m
		}
		if _, err := stmt.ExecContext(ctx, e.ClientID, e.SessionID, e.EventType, e.Timestamp, meta); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) RegisterClient(ctx context.Context, clientID string) (types.ClientConfig, error) {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO clients (id, flush_interval, tracked_events) VALUES (?, ?, ?)`,
		clientID, 5000, "pageview,click")
	if err != nil {
		return types.ClientConfig{}, err
	}
	return s.GetClient(ctx, clientID)
}

func (s *Store) GetActiveSessions(ctx context.Context, since time.Time, before *time.Time, limit int) ([]types.SessionSummary, int, error) {
	sinceStr := since.UTC().Format(time.RFC3339)

	var total int
	row := s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT session_id) FROM events WHERE timestamp > ?`, sinceStr)
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT session_id, MAX(client_id) AS client_id, COUNT(*) AS event_count, MAX(timestamp) AS last_seen
	          FROM events
	          WHERE timestamp > ?
	          GROUP BY session_id`
	args := []any{sinceStr}

	if before != nil {
		query += ` HAVING last_seen < ?`
		args = append(args, before.UTC().Format(time.RFC3339))
	}

	query += ` ORDER BY last_seen DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := []types.SessionSummary{}
	for rows.Next() {
		var s types.SessionSummary
		if err := rows.Scan(&s.SessionID, &s.ClientID, &s.EventCount, &s.LastSeen); err != nil {
			return nil, 0, err
		}
		result = append(result, s)
	}
	return result, total, rows.Err()
}

func (s *Store) GetSessionEvents(ctx context.Context, sessionID string, before *time.Time, limit int) ([]types.EventRecord, int, error) {
	var total int
	row := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM events WHERE session_id = ?`, sessionID)
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, client_id, session_id, event_type, timestamp, metadata
	          FROM events
	          WHERE session_id = ?`
	args := []any{sessionID}

	if before != nil {
		query += ` AND timestamp < ?`
		args = append(args, before.UTC().Format(time.RFC3339Nano))
	}

	query += ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := []types.EventRecord{}
	for rows.Next() {
		var r types.EventRecord
		var meta sql.NullString
		if err := rows.Scan(&r.ID, &r.ClientID, &r.SessionID, &r.EventType, &r.Timestamp, &meta); err != nil {
			return nil, 0, err
		}
		if meta.Valid && meta.String != "" {
			r.Metadata = json.RawMessage(meta.String)
		} else {
			r.Metadata = json.RawMessage("{}")
		}
		result = append(result, r)
	}
	return result, total, rows.Err()
}

func (s *Store) GetStats(ctx context.Context, clientID string, since time.Time) (types.Stats, error) {
	stats := types.Stats{
		ClientID:  clientID,
		Breakdown: map[string]int{},
	}

	row := s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT session_id) FROM events
		 WHERE client_id = ? AND timestamp > ?`,
		clientID, since.UTC().Format(time.RFC3339))
	if err := row.Scan(&stats.SessionCount); err != nil {
		return stats, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT event_type, COUNT(*) FROM events
		 WHERE client_id = ? AND timestamp > ?
		 GROUP BY event_type`,
		clientID, since.UTC().Format(time.RFC3339))
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return stats, err
		}
		stats.Breakdown[eventType] = count
	}
	return stats, rows.Err()
}
