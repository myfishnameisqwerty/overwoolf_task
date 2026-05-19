package handler

import (
	"errors"
	"net/http"
	"tracker/internal/storage"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientId")
	if !validateClientID(clientID) {
		writeError(w, http.StatusBadRequest, "invalid client ID")
		return
	}
	cfg, err := h.Store.GetClient(r.Context(), clientID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			cfg, err = h.Store.RegisterClient(r.Context(), clientID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}
			writeSuccess(w, http.StatusOK, cfg)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeSuccess(w, http.StatusOK, cfg)
}
