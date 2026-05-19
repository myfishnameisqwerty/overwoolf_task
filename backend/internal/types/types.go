package types

import "encoding/json"

type ClientConfig struct {
	ClientID      string   `json:"clientId"`
	FlushInterval int      `json:"flushInterval"`
	TrackedEvents []string `json:"trackedEvents"`
}

type Event struct {
	EventID   string          `json:"eventId"`
	ClientID  string          `json:"clientId"`
	SessionID string          `json:"sessionId"`
	EventType string          `json:"eventType"`
	Timestamp string          `json:"timestamp"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type EventRecord struct {
	ID        int64           `json:"id"`
	EventID   string          `json:"eventId"`
	ClientID  string          `json:"clientId"`
	SessionID string          `json:"sessionId"`
	EventType string          `json:"eventType"`
	Timestamp string          `json:"timestamp"`
	ServerTS  string          `json:"serverTs"`
	Metadata  json.RawMessage `json:"metadata"`
}

type SessionSummary struct {
	SessionID  string `json:"sessionId"`
	ClientID   string `json:"clientId"`
	EventCount int    `json:"eventCount"`
	LastSeen   string `json:"lastSeen"`
}

type EventsPage struct {
	Total      int           `json:"total"`
	Items      []EventRecord `json:"items"`
	NextCursor *string       `json:"nextCursor"`
}

type SessionsPage struct {
	Total      int              `json:"total"`
	Items      []SessionSummary `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}

type Stats struct {
	ClientID     string         `json:"clientId"`
	SessionCount int            `json:"sessionCount"`
	Breakdown    map[string]int `json:"breakdown"`
}

type SuccessResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
