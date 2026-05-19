package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"tracker/internal/types"
)

func (h *Handler) PostEvents(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "request body required")
		return
	}

	var events []types.Event
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if events == nil {
		writeError(w, http.StatusBadRequest, "body must be a JSON array")
		return
	}

	for i, e := range events {
		if e.EventID == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("event[%d]: eventId required", i))
			return
		}
		if !validateClientID(e.ClientID) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("event[%d]: invalid client ID", i))
			return
		}
		if e.SessionID == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("event[%d]: sessionId required", i))
			return
		}
		if e.EventType == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("event[%d]: eventType required", i))
			return
		}
		if e.Timestamp == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("event[%d]: timestamp required", i))
			return
		}
		if _, err := time.Parse(time.RFC3339, e.Timestamp); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("event[%d]: timestamp must be RFC3339", i))
			return
		}
	}

	if err := h.Store.InsertEvents(r.Context(), events); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store events")
		return
	}

	writeSuccess(w, http.StatusOK, nil)
}
