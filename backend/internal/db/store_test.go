package db_test

import (
	"context"
	"testing"
	"time"
	"tracker/internal/db"
	"tracker/internal/storage"
	"tracker/internal/types"
)

func openTestDB(t *testing.T) storage.Storage {
	t.Helper()
	store, err := db.New(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func evt(clientID, sessionID, eventType string, ts time.Time) types.Event {
	return types.Event{
		ClientID:  clientID,
		SessionID: sessionID,
		EventType: eventType,
		Timestamp: ts.UTC().Format(time.RFC3339),
	}
}

// TestGetClient_Found verifies the seeded demo-client row is returned correctly.
func TestGetClient_Found(t *testing.T) {
	s := openTestDB(t)
	cfg, err := s.GetClient(context.Background(), "demo-client")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ClientID != "demo-client" {
		t.Errorf("clientId = %q, want demo-client", cfg.ClientID)
	}
	if cfg.FlushInterval != 5000 {
		t.Errorf("flushInterval = %d, want 5000", cfg.FlushInterval)
	}
	if len(cfg.TrackedEvents) != 2 || cfg.TrackedEvents[0] != "pageview" || cfg.TrackedEvents[1] != "click" {
		t.Errorf("trackedEvents = %v, want [pageview click]", cfg.TrackedEvents)
	}
}

func TestGetClient_NotFound(t *testing.T) {
	s := openTestDB(t)
	_, err := s.GetClient(context.Background(), "no-such-client")
	if err != storage.ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestInsertEvents_Single(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	if err := s.InsertEvents(ctx, []types.Event{evt("demo-client", "sess-1", "pageview", time.Now())}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	records, _, err := s.GetSessionEvents(ctx, "sess-1", nil, 1000)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len = %d, want 1", len(records))
	}
	if records[0].EventType != "pageview" {
		t.Errorf("eventType = %q, want pageview", records[0].EventType)
	}
}

func TestInsertEvents_Batch(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	now := time.Now()

	batch := []types.Event{
		evt("demo-client", "sess-batch", "pageview", now),
		evt("demo-client", "sess-batch", "click", now.Add(1*time.Second)),
		evt("demo-client", "sess-batch", "click", now.Add(2*time.Second)),
		evt("demo-client", "sess-batch", "custom", now.Add(3*time.Second)),
		evt("demo-client", "sess-batch", "custom", now.Add(4*time.Second)),
	}
	if err := s.InsertEvents(ctx, batch); err != nil {
		t.Fatalf("insert: %v", err)
	}

	records, _, err := s.GetSessionEvents(ctx, "sess-batch", nil, 1000)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(records) != 5 {
		t.Errorf("len = %d, want 5", len(records))
	}
}

func TestGetActiveSessions_Empty(t *testing.T) {
	s := openTestDB(t)
	sessions, _, err := s.GetActiveSessions(context.Background(), time.Now().Add(-5*time.Minute), nil, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessions == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(sessions) != 0 {
		t.Errorf("len = %d, want 0", len(sessions))
	}
}

func TestGetActiveSessions_WithinWindow(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	now := time.Now()

	s.InsertEvents(ctx, []types.Event{evt("demo-client", "sess-active", "pageview", now)})

	sessions, _, err := s.GetActiveSessions(ctx, now.Add(-5*time.Minute), nil, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len = %d, want 1", len(sessions))
	}
	if sessions[0].SessionID != "sess-active" {
		t.Errorf("sessionId = %q, want sess-active", sessions[0].SessionID)
	}
	if sessions[0].EventCount != 1 {
		t.Errorf("eventCount = %d, want 1", sessions[0].EventCount)
	}
}

func TestGetActiveSessions_OutsideWindow(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	now := time.Now()
	old := now.Add(-10 * time.Minute)

	s.InsertEvents(ctx, []types.Event{evt("demo-client", "sess-old", "pageview", old)})

	sessions, _, err := s.GetActiveSessions(ctx, now.Add(-5*time.Minute), nil, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions outside window, got %d", len(sessions))
	}
}

func TestGetActiveSessions_Ordering(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	now := time.Now()
	earlier := now.Add(-1 * time.Minute)

	s.InsertEvents(ctx, []types.Event{evt("demo-client", "sess-earlier", "click", earlier)})
	s.InsertEvents(ctx, []types.Event{evt("demo-client", "sess-later", "click", now)})

	sessions, _, err := s.GetActiveSessions(ctx, now.Add(-5*time.Minute), nil, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("len = %d, want 2", len(sessions))
	}
	if sessions[0].SessionID != "sess-later" {
		t.Errorf("first = %q, want sess-later (most recent first)", sessions[0].SessionID)
	}
}

func TestGetActiveSessions_Pagination(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	now := time.Now()

	// Insert 3 sessions with distinct timestamps, 1 second apart.
	s.InsertEvents(ctx, []types.Event{evt("demo-client", "sess-1", "pageview", now.Add(-2*time.Second))})
	s.InsertEvents(ctx, []types.Event{evt("demo-client", "sess-2", "pageview", now.Add(-1*time.Second))})
	s.InsertEvents(ctx, []types.Event{evt("demo-client", "sess-3", "pageview", now)})

	// Page 1: limit=2, no cursor → most recent 2 sessions.
	page1, _, err := s.GetActiveSessions(ctx, now.Add(-5*time.Minute), nil, 2)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1 len = %d, want 2", len(page1))
	}
	if page1[0].SessionID != "sess-3" {
		t.Errorf("page1[0] = %q, want sess-3", page1[0].SessionID)
	}
	if page1[1].SessionID != "sess-2" {
		t.Errorf("page1[1] = %q, want sess-2", page1[1].SessionID)
	}

	// Page 2: cursor = last_seen of page1[1].
	cursor, _ := time.Parse(time.RFC3339, page1[1].LastSeen)
	page2, _, err := s.GetActiveSessions(ctx, now.Add(-5*time.Minute), &cursor, 2)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("page2 len = %d, want 1", len(page2))
	}
	if page2[0].SessionID != "sess-1" {
		t.Errorf("page2[0] = %q, want sess-1", page2[0].SessionID)
	}
}

func TestGetSessionEvents_Ordering(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	now := time.Now()

	t1 := now.Add(-2 * time.Minute)
	t2 := now.Add(-1 * time.Minute)
	t3 := now

	// Insert out of order to verify ORDER BY takes effect.
	s.InsertEvents(ctx, []types.Event{
		evt("demo-client", "sess-ord", "click", t3),
		evt("demo-client", "sess-ord", "pageview", t1),
		evt("demo-client", "sess-ord", "custom", t2),
	})

	records, _, err := s.GetSessionEvents(ctx, "sess-ord", nil, 1000)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("len = %d, want 3", len(records))
	}
	if records[0].Timestamp > records[1].Timestamp || records[1].Timestamp > records[2].Timestamp {
		t.Errorf("events not ascending: %s %s %s", records[0].Timestamp, records[1].Timestamp, records[2].Timestamp)
	}
}

func TestGetSessionEvents_Pagination(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	now := time.Now()

	s.InsertEvents(ctx, []types.Event{
		evt("demo-client", "sess-p", "pageview", now.Add(-2*time.Second)),
		evt("demo-client", "sess-p", "click", now.Add(-1*time.Second)),
		evt("demo-client", "sess-p", "custom", now),
	})

	// Page 1: limit=2, no cursor → 2 oldest events (ASC).
	page1, _, err := s.GetSessionEvents(ctx, "sess-p", nil, 2)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1 len = %d, want 2", len(page1))
	}
	if page1[0].EventType != "pageview" {
		t.Errorf("page1[0] = %q, want pageview (oldest first)", page1[0].EventType)
	}
	if page1[1].EventType != "click" {
		t.Errorf("page1[1] = %q, want click", page1[1].EventType)
	}

	// Page 2: cursor = timestamp of page1[1] (newest item on page 1).
	cursor, _ := time.Parse(time.RFC3339, page1[1].Timestamp)
	page2, _, err := s.GetSessionEvents(ctx, "sess-p", &cursor, 2)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("page2 len = %d, want 1", len(page2))
	}
	if page2[0].EventType != "custom" {
		t.Errorf("page2[0] = %q, want custom (newest)", page2[0].EventType)
	}
}

func TestGetSessionEvents_UnknownSession(t *testing.T) {
	s := openTestDB(t)
	records, _, err := s.GetSessionEvents(context.Background(), "nonexistent", nil, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(records) != 0 {
		t.Errorf("len = %d, want 0", len(records))
	}
}

func TestGetStats_Empty(t *testing.T) {
	s := openTestDB(t)
	stats, err := s.GetStats(context.Background(), "demo-client", time.Now().Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.SessionCount != 0 {
		t.Errorf("sessionCount = %d, want 0", stats.SessionCount)
	}
	if stats.Breakdown == nil {
		t.Error("breakdown is nil, want empty map")
	}
	if len(stats.Breakdown) != 0 {
		t.Errorf("breakdown len = %d, want 0", len(stats.Breakdown))
	}
}

func TestGetStats_Populated(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	now := time.Now()

	s.InsertEvents(ctx, []types.Event{
		evt("demo-client", "sess-a", "pageview", now),
		evt("demo-client", "sess-a", "click", now),
		evt("demo-client", "sess-b", "pageview", now),
	})

	stats, err := s.GetStats(ctx, "demo-client", now.Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.SessionCount != 2 {
		t.Errorf("sessionCount = %d, want 2", stats.SessionCount)
	}
	if stats.Breakdown["pageview"] != 2 {
		t.Errorf("pageview = %d, want 2", stats.Breakdown["pageview"])
	}
	if stats.Breakdown["click"] != 1 {
		t.Errorf("click = %d, want 1", stats.Breakdown["click"])
	}
}

func TestRegisterClient_CreatesDefault(t *testing.T) {
	s := openTestDB(t)
	cfg, err := s.RegisterClient(context.Background(), "new-site.com")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if cfg.ClientID != "new-site.com" {
		t.Errorf("clientId = %q, want new-site.com", cfg.ClientID)
	}
	if cfg.FlushInterval != 5000 {
		t.Errorf("flushInterval = %d, want 5000", cfg.FlushInterval)
	}
	if len(cfg.TrackedEvents) != 2 {
		t.Errorf("trackedEvents len = %d, want 2", len(cfg.TrackedEvents))
	}

	// Must be retrievable via GetClient now.
	got, err := s.GetClient(context.Background(), "new-site.com")
	if err != nil {
		t.Fatalf("get after register: %v", err)
	}
	if got.ClientID != "new-site.com" {
		t.Errorf("get clientId = %q, want new-site.com", got.ClientID)
	}
}

func TestRegisterClient_Idempotent(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	first, err := s.RegisterClient(ctx, "idempotent.com")
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	second, err := s.RegisterClient(ctx, "idempotent.com")
	if err != nil {
		t.Fatalf("second register: %v", err)
	}
	if first.ClientID != second.ClientID || first.FlushInterval != second.FlushInterval {
		t.Errorf("second register returned different config: %+v vs %+v", first, second)
	}
}

func TestGetStats_WindowFiltering(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	now := time.Now()
	old := now.Add(-10 * time.Minute)

	s.InsertEvents(ctx, []types.Event{evt("demo-client", "sess-old", "pageview", old)})
	s.InsertEvents(ctx, []types.Event{evt("demo-client", "sess-new", "pageview", now)})

	stats, err := s.GetStats(ctx, "demo-client", now.Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.SessionCount != 1 {
		t.Errorf("sessionCount = %d, want 1 (old event filtered)", stats.SessionCount)
	}
	if stats.Breakdown["pageview"] != 1 {
		t.Errorf("pageview = %d, want 1", stats.Breakdown["pageview"])
	}
}
