package handler

import (
	"net/http"
	"time"
	"tracker/internal/types"

	"github.com/go-chi/chi/v5"
)

const sessionsPageSize = 100

func (h *Handler) GetSessions(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-5 * time.Minute)

	var before *time.Time
	if raw := r.URL.Query().Get("before"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid before: must be RFC3339")
			return
		}
		before = &t
	}

	sessions, total, err := h.Store.GetActiveSessions(r.Context(), since, before, sessionsPageSize+1)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	page := types.SessionsPage{Total: total, Items: sessions}
	if len(sessions) > sessionsPageSize {
		page.Items = sessions[:sessionsPageSize]
		cursor := sessions[sessionsPageSize-1].LastSeen
		page.NextCursor = &cursor
	}
	if page.Items == nil {
		page.Items = []types.SessionSummary{}
	}

	writeSuccess(w, http.StatusOK, page)
}

func (h *Handler) GetSessionByID(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")

	var before *time.Time
	if raw := r.URL.Query().Get("before"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid before: must be RFC3339")
			return
		}
		before = &t
	}

	events, total, err := h.Store.GetSessionEvents(r.Context(), sessionID, before, sessionsPageSize+1)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	page := types.EventsPage{Total: total, Items: events}
	if len(events) > sessionsPageSize {
		page.Items = events[:sessionsPageSize]
		cursor := events[sessionsPageSize-1].Timestamp
		page.NextCursor = &cursor
	}
	if page.Items == nil {
		page.Items = []types.EventRecord{}
	}

	writeSuccess(w, http.StatusOK, page)
}
