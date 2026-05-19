package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tracker/internal/db"
	"tracker/internal/handler"
	"tracker/internal/server"
	"tracker/internal/types"
)

const testFileMode = 0644

func newTestServer(t *testing.T) (*httptest.Server, *handler.Handler) {
	t.Helper()
	store, err := db.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	h := &handler.Handler{
		Store:         store,
		WidgetPath:    widgetFile(t),
		DashboardPath: dashboardFile(t),
	}

	srv := httptest.NewServer(server.New(h))
	t.Cleanup(srv.Close)
	return srv, h
}

func widgetFile(t *testing.T) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "widget.js")
	os.WriteFile(f, []byte("// widget"), testFileMode)
	return f
}

func dashboardFile(t *testing.T) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "dashboard.html")
	os.WriteFile(f, []byte("<html></html>"), testFileMode)
	return f
}

func postEvents(t *testing.T, srv *httptest.Server, events []types.Event) *http.Response {
	t.Helper()
	body, _ := json.Marshal(events)
	resp, err := http.Post(srv.URL+"/events", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post events: %v", err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	resp.Body.Close()
	return m
}

func seedEvent(t *testing.T, srv *httptest.Server, sessionID, eventType string, ts time.Time) {
	t.Helper()
	events := []types.Event{{
		ClientID:  "demo-client",
		SessionID: sessionID,
		EventType: eventType,
		Timestamp: ts.UTC().Format(time.RFC3339),
	}}
	resp := postEvents(t, srv, events)
	resp.Body.Close()
}

// GET /config/:clientId

func TestGetConfig_Success(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/config/demo-client")
	body := decodeBody(t, resp)

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if body["status"] != "success" {
		t.Errorf("status = %v, want success", body["status"])
	}
	data, _ := body["data"].(map[string]any)
	if data["clientId"] != "demo-client" {
		t.Errorf("clientId = %v, want demo-client", data["clientId"])
	}
}

func TestGetConfig_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/config/brand-new-client")
	body := decodeBody(t, resp)

	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want error", body["status"])
	}
}

func TestGetConfig_TrackedEventsIsArray(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/config/demo-client")
	body := decodeBody(t, resp)

	data, _ := body["data"].(map[string]any)
	events, ok := data["trackedEvents"].([]any)
	if !ok {
		t.Fatalf("trackedEvents is not an array: %T", data["trackedEvents"])
	}
	if len(events) != 2 {
		t.Errorf("len(trackedEvents) = %d, want 2", len(events))
	}
}

// POST /events

func TestPostEvents_Valid(t *testing.T) {
	srv, _ := newTestServer(t)
	events := []types.Event{{
		ClientID: "demo-client", SessionID: "s1", EventType: "pageview",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}}
	resp := postEvents(t, srv, events)
	body := decodeBody(t, resp)

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if body["status"] != "success" {
		t.Errorf("status = %v, want success", body["status"])
	}
	if body["data"] != nil {
		t.Errorf("data = %v, want null", body["data"])
	}
}

func TestPostEvents_EmptyBody(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Post(srv.URL+"/events", "application/json", strings.NewReader(""))
	body := decodeBody(t, resp)

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want error", body["status"])
	}
}

func TestPostEvents_MalformedJSON(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Post(srv.URL+"/events", "application/json", strings.NewReader("{not json}"))
	body := decodeBody(t, resp)

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want error", body["status"])
	}
}

func TestPostEvents_NotAnArray(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Post(srv.URL+"/events", "application/json", strings.NewReader(`{"clientId":"demo"}`))
	body := decodeBody(t, resp)

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want error", body["status"])
	}
}

// GET /sessions

func sessionsPage(t *testing.T, body map[string]any) (items []any, nextCursor any) {
	t.Helper()
	d, _ := body["data"].(map[string]any)
	items, _ = d["items"].([]any)
	nextCursor = d["nextCursor"]
	return
}

func TestGetSessions_Empty(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/sessions")
	body := decodeBody(t, resp)

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	items, _ := sessionsPage(t, body)
	if len(items) != 0 {
		t.Errorf("len = %d, want 0", len(items))
	}
}

func TestGetSessions_Active(t *testing.T) {
	srv, _ := newTestServer(t)
	seedEvent(t, srv, "sess-live", "pageview", time.Now())

	resp, _ := http.Get(srv.URL + "/sessions")
	body := decodeBody(t, resp)

	items, _ := sessionsPage(t, body)
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	s, _ := items[0].(map[string]any)
	if s["sessionId"] != "sess-live" {
		t.Errorf("sessionId = %v, want sess-live", s["sessionId"])
	}
}

func TestGetSessions_OldEventsExcluded(t *testing.T) {
	srv, _ := newTestServer(t)
	old := time.Now().Add(-10 * time.Minute)
	seedEvent(t, srv, "sess-old", "pageview", old)

	resp, _ := http.Get(srv.URL + "/sessions")
	body := decodeBody(t, resp)

	items, _ := sessionsPage(t, body)
	if len(items) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(items))
	}
}

func TestGetSessions_NoCursorWhenFewItems(t *testing.T) {
	srv, _ := newTestServer(t)
	seedEvent(t, srv, "sess-a", "pageview", time.Now())

	resp, _ := http.Get(srv.URL + "/sessions")
	body := decodeBody(t, resp)

	_, cursor := sessionsPage(t, body)
	if cursor != nil {
		t.Errorf("nextCursor = %v, want null", cursor)
	}
}

func TestGetSessions_InvalidCursor(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/sessions?before=not-a-timestamp")
	body := decodeBody(t, resp)

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want error", body["status"])
	}
}

// GET /sessions/:sessionId

func eventsPage(t *testing.T, body map[string]any) (items []any, nextCursor any) {
	t.Helper()
	d, _ := body["data"].(map[string]any)
	items, _ = d["items"].([]any)
	nextCursor = d["nextCursor"]
	return
}

func TestGetSessionByID_Exists(t *testing.T) {
	srv, _ := newTestServer(t)
	now := time.Now()
	seedEvent(t, srv, "sess-x", "pageview", now)
	seedEvent(t, srv, "sess-x", "click", now.Add(1*time.Second))

	resp, _ := http.Get(srv.URL + "/sessions/sess-x")
	body := decodeBody(t, resp)

	items, _ := eventsPage(t, body)
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}
}

func TestGetSessionByID_Unknown(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/sessions/nonexistent")
	body := decodeBody(t, resp)

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200 (not 404)", resp.StatusCode)
	}
	items, _ := eventsPage(t, body)
	if len(items) != 0 {
		t.Errorf("len = %d, want 0", len(items))
	}
}

func TestGetSessionByID_FullHistory(t *testing.T) {
	srv, _ := newTestServer(t)
	old := time.Now().Add(-10 * time.Minute)
	recent := time.Now()
	seedEvent(t, srv, "sess-h", "pageview", old)
	seedEvent(t, srv, "sess-h", "click", recent)

	resp, _ := http.Get(srv.URL + "/sessions/sess-h")
	body := decodeBody(t, resp)

	items, _ := eventsPage(t, body)
	if len(items) != 2 {
		t.Errorf("len = %d, want 2 (full history, not filtered by 5-min window)", len(items))
	}
}

func TestGetSessionByID_InvalidCursor(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/sessions/any?after=bad-ts")
	body := decodeBody(t, resp)

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want error", body["status"])
	}
}

// GET /stats/:clientId

func TestGetStats_NoData(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/stats/demo-client")
	body := decodeBody(t, resp)

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	data, _ := body["data"].(map[string]any)
	if data["sessionCount"].(float64) != 0 {
		t.Errorf("sessionCount = %v, want 0", data["sessionCount"])
	}
	breakdown, ok := data["breakdown"].(map[string]any)
	if !ok {
		t.Fatalf("breakdown is not an object: %T", data["breakdown"])
	}
	if len(breakdown) != 0 {
		t.Errorf("breakdown len = %d, want 0", len(breakdown))
	}
}

func TestGetStats_WithData(t *testing.T) {
	srv, _ := newTestServer(t)
	now := time.Now()
	seedEvent(t, srv, "s1", "pageview", now)
	seedEvent(t, srv, "s1", "click", now)
	seedEvent(t, srv, "s2", "pageview", now)

	resp, _ := http.Get(srv.URL + "/stats/demo-client")
	body := decodeBody(t, resp)

	data, _ := body["data"].(map[string]any)
	if data["sessionCount"].(float64) != 2 {
		t.Errorf("sessionCount = %v, want 2", data["sessionCount"])
	}
	breakdown, _ := data["breakdown"].(map[string]any)
	if breakdown["pageview"].(float64) != 2 {
		t.Errorf("pageview = %v, want 2", breakdown["pageview"])
	}
}

// CORS

func TestCORS_SuccessResponse(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/config/demo-client")
	resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS header missing on success response")
	}
}

func TestCORS_ErrorResponse(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/config/unknown")
	resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS header missing on 404 response")
	}
}

func TestCORS_Options(t *testing.T) {
	srv, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/events", nil)
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	if resp.StatusCode != 204 {
		t.Errorf("status = %d, want 204", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS origin header missing on OPTIONS")
	}
	if resp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Errorf("CORS methods header missing on OPTIONS")
	}
}

// Static files

func TestWidgetJS_ContentType(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/widget.js")
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/javascript") {
		t.Errorf("Content-Type = %q, want application/javascript", ct)
	}
}

func TestDashboard_ContentType(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/dashboard")
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
}
