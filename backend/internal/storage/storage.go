package storage

import (
	"context"
	"errors"
	"time"
	"tracker/internal/types"
)

var ErrNotFound = errors.New("not found")

type Storage interface {
	GetClient(ctx context.Context, clientID string) (types.ClientConfig, error)
	InsertEvents(ctx context.Context, events []types.Event) error
	GetActiveSessions(ctx context.Context, since time.Time, before *time.Time, limit int) ([]types.SessionSummary, int, error)
	GetSessionEvents(ctx context.Context, sessionID string, before *time.Time, limit int) ([]types.EventRecord, int, error)
	GetStats(ctx context.Context, clientID string, since time.Time) (types.Stats, error)
	RegisterClient(ctx context.Context, clientID string) (types.ClientConfig, error)
	Close() error
}
