package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientId")
	if !validateClientID(clientID) {
		writeError(w, http.StatusBadRequest, "invalid client ID")
		return
	}
	since := time.Now().Add(-5 * time.Minute)
	stats, err := h.Store.GetStats(r.Context(), clientID, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeSuccess(w, http.StatusOK, stats)
}
