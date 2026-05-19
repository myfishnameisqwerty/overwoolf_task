package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"tracker/internal/storage"
	"tracker/internal/types"
)

type Handler struct {
	Store         storage.Storage
	WidgetPath    string
	DashboardPath string
}

func writeSuccess(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(types.SuccessResponse{Status: "success", Data: data})
}

func writeError(w http.ResponseWriter, code int, message string) {
	if code >= 500 {
		slog.Error("handler error", "code", code, "message", message)
	} else {
		slog.Warn("client error", "code", code, "message", message)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(types.ErrorResponse{Status: "error", Message: message})
}

// validateClientID is the extension point for client authorization.
// Replace the body with a real check (DB lookup, JWT, allowlist) when needed.
func validateClientID(clientID string) bool {
	return clientID != ""
}
